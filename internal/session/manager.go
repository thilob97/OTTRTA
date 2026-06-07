package session

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/thilob97/ottrta/internal/event"
)

const DefaultMaxLogs = 1000

// Manager owns session state and process/PTY runtimes for the TUI.
type Manager struct {
	sessions []Session
	maxLogs  int
	runtimes map[string]*ProcessRuntime
	ptys     map[string]PTYSession
}

func NewManager(sessions []Session) Manager {
	items := make([]Session, len(sessions))
	for i := range sessions {
		items[i] = cloneSession(sessions[i])
	}

	return Manager{
		sessions: items,
		maxLogs:  DefaultMaxLogs,
		runtimes: make(map[string]*ProcessRuntime),
		ptys:     make(map[string]PTYSession),
	}
}

func (m *Manager) Count() int {
	return len(m.sessions)
}

func (m *Manager) Sessions() []Session {
	items := make([]Session, len(m.sessions))
	for i := range m.sessions {
		items[i] = cloneSession(m.sessions[i])
	}
	return items
}

func (m *Manager) Session(index int) (*Session, bool) {
	if index < 0 || index >= len(m.sessions) {
		return nil, false
	}
	return &m.sessions[index], true
}

func (m *Manager) SessionByID(id string) (*Session, bool) {
	for i := range m.sessions {
		if m.sessions[i].ID == id {
			return &m.sessions[i], true
		}
	}
	return nil, false
}

func (m *Manager) RenameSession(id, name string) bool {
	s, ok := m.SessionByID(id)
	if !ok {
		return false
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	s.Name = name
	return true
}

func (m *Manager) ClampIndex(index int) int {
	if len(m.sessions) == 0 || index < 0 {
		return 0
	}
	if index >= len(m.sessions) {
		return len(m.sessions) - 1
	}
	return index
}
func (m *Manager) AddSession(s Session) {
	m.sessions = append(m.sessions, cloneSession(s))
}

func (m *Manager) RemoveSession(index int) error {
	if index < 0 || index >= len(m.sessions) {
		return nil
	}
	s := &m.sessions[index]
	id := s.ID
	switch s.Kind {
	case SessionKindProcess:
		if _, ok := m.runtimes[id]; ok {
			if err := m.StopSession(id); err != nil {
				return err
			}
		}
	case SessionKindPTY, SessionKindAgent:
		if _, ok := m.ptys[id]; ok {
			if err := m.StopPTYSession(id); err != nil {
				return err
			}
		}
	}
	m.sessions = append(m.sessions[:index], m.sessions[index+1:]...)
	return nil
}

func (m *Manager) AppendLog(id string, line string) bool {
	s, ok := m.SessionByID(id)
	if !ok {
		return false
	}
	m.appendLog(s, line)
	return true
}

func (m *Manager) AppendOutput(id string, text string) bool {
	s, ok := m.SessionByID(id)
	if !ok {
		return false
	}
	m.appendOutput(s, text)
	return true
}

func (m *Manager) MarkStopped(id string) bool {
	s, ok := m.SessionByID(id)
	if !ok {
		return false
	}
	s.Status = StatusStopped
	delete(m.runtimes, id)
	delete(m.ptys, id)
	return true
}
func (m *Manager) SetStatus(id string, status Status) bool {
	s, ok := m.SessionByID(id)
	if !ok {
		return false
	}
	s.Status = status
	if status != StatusRunning {
		delete(m.runtimes, id)
		delete(m.ptys, id)
	}
	return true
}
func (m *Manager) SetAttention(id string, attention bool) bool {
	s, ok := m.SessionByID(id)
	if !ok {
		return false
	}
	s.NeedsAttention = attention
	return true
}

func (m *Manager) StartSession(ctx context.Context, id string) (<-chan event.ProcessMsg, error) {
	s, ok := m.SessionByID(id)
	if !ok {
		return nil, fmt.Errorf("session %q not found", id)
	}
	if s.Kind != SessionKindProcess {
		return nil, fmt.Errorf("session %q is not a process session", id)
	}
	if s.Status == StatusRunning {
		return nil, fmt.Errorf("session %q is already running", id)
	}
	if _, ok := m.runtimes[id]; ok {
		return nil, fmt.Errorf("session %q already has an active process", id)
	}

	runtime, err := StartProcess(ctx, *s)
	if err != nil {
		return nil, err
	}

	s.Status = StatusRunning
	m.runtimes[id] = runtime
	m.appendLog(s, fmt.Sprintf("[system] started: %s", runtime.CommandLine()))
	return runtime.Events(), nil
}

func (m *Manager) StopSession(id string) error {
	s, ok := m.SessionByID(id)
	if !ok {
		return fmt.Errorf("session %q not found", id)
	}
	if s.Kind != SessionKindProcess {
		return fmt.Errorf("session %q is not a process session", id)
	}
	runtime, ok := m.runtimes[id]
	if !ok {
		return fmt.Errorf("session %q is not running", id)
	}

	runtime.Stop()
	s.Status = StatusStopped
	m.appendLog(s, "[system] stopped")
	delete(m.runtimes, id)
	return nil
}

func (m *Manager) StartPTYSession(ctx context.Context, id string, cols, rows int) (<-chan PTYEvent, error) {
	s, ok := m.SessionByID(id)
	if !ok {
		return nil, fmt.Errorf("session %q not found", id)
	}
	if s.Kind != SessionKindPTY && s.Kind != SessionKindAgent {
		return nil, fmt.Errorf("session %q is not a PTY/Agent session", id)
	}
	if s.Status == StatusRunning {
		return nil, fmt.Errorf("session %q is already running", id)
	}
	if _, ok := m.ptys[id]; ok {
		return nil, fmt.Errorf("session %q already has an active PTY", id)
	}

	runtime := newPTYSession(id)
	if err := runtime.Start(ctx, PTYSpec{
		Command: s.Command,
		Args:    append([]string(nil), s.Args...),
		WorkDir: s.WorkDir,
		Cols:    cols,
		Rows:    rows,
	}); err != nil {
		return nil, err
	}

	s.Status = StatusRunning
	m.ptys[id] = runtime
	m.appendLog(s, fmt.Sprintf("[system] started PTY: %s", commandLine(s.Command, s.Args)))
	return runtime.Events(), nil
}

func (m *Manager) WritePTYSession(id string, data []byte) error {
	runtime, ok := m.ptys[id]
	if !ok {
		return fmt.Errorf("session %q is not running", id)
	}
	return runtime.Write(data)
}

func (m *Manager) ResizePTYSession(id string, cols, rows int) error {
	runtime, ok := m.ptys[id]
	if !ok {
		return fmt.Errorf("session %q is not running", id)
	}
	return runtime.Resize(cols, rows)
}

func (m *Manager) StopPTYSession(id string) error {
	s, ok := m.SessionByID(id)
	if !ok {
		return fmt.Errorf("session %q not found", id)
	}
	if s.Kind != SessionKindPTY && s.Kind != SessionKindAgent {
		return fmt.Errorf("session %q is not a PTY/Agent session", id)
	}
	runtime, ok := m.ptys[id]
	if !ok {
		return fmt.Errorf("session %q is not running", id)
	}

	stopErr := runtime.Stop()
	s.Status = StatusStopped
	m.appendLog(s, "[system] stopped")
	delete(m.ptys, id)
	return stopErr
}

func (m *Manager) StopAllProcesses() {
	for id, runtime := range m.runtimes {
		runtime.Stop()
		if s, ok := m.SessionByID(id); ok {
			s.Status = StatusStopped
			m.appendLog(s, "[system] stopped")
		}
		delete(m.runtimes, id)
	}
	for id, runtime := range m.ptys {
		_ = runtime.Stop()
		if s, ok := m.SessionByID(id); ok {
			s.Status = StatusStopped
			m.appendLog(s, "[system] stopped")
		}
		delete(m.ptys, id)
	}
}

func (m *Manager) AppendExitLog(id string, err error) bool {
	if err == nil {
		return m.AppendLog(id, "[system] exited with code 0")
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		code := exitErr.ExitCode()
		if code >= 0 {
			return m.AppendLog(id, fmt.Sprintf("[system] exited with code %d", code))
		}
	}
	return m.AppendLog(id, "[system] exited")
}

func commandLine(command string, args []string) string {
	if len(args) == 0 {
		return quoteCommandPart(command)
	}
	var builder strings.Builder
	builder.WriteString(quoteCommandPart(command))
	for _, arg := range args {
		builder.WriteByte(' ')
		builder.WriteString(quoteCommandPart(arg))
	}
	return builder.String()
}

func quoteCommandPart(value string) string {
	if !needsCommandQuote(value) {
		return value
	}
	return strconv.Quote(value)
}

func needsCommandQuote(value string) bool {
	if value == "" {
		return true
	}
	for _, r := range value {
		if unicode.IsSpace(r) || strings.ContainsRune(`"'\\$&|;()<>*?![]{}~`+"`", r) {
			return true
		}
	}
	return false
}

func (m *Manager) appendOutput(s *Session, text string) {
	if text == "" {
		return
	}
	for i := 0; i < len(text); {
		switch text[i] {
		case '\x1b':
			i = m.handleEscape(s, text, i)
		case '\r':
			s.outputCol = 0
			i++
		case '\n':
			s.outputRow++
			s.outputCol = 0
			m.ensureOutputLine(s)
			i++
		case '\b', 0x7f:
			m.deleteLastLogRune(s)
			i++
		default:
			r, size := rune(text[i]), 1
			if text[i] >= 0x80 {
				r, size = utf8.DecodeRuneInString(text[i:])
			}
			m.writeOutputRune(s, r)
			i += size
		}
	}
}

func (m *Manager) handleEscape(s *Session, text string, start int) int {
	if start+1 >= len(text) {
		return len(text)
	}
	next := text[start+1]
	if next == '[' {
		return m.handleCSI(s, text, start+2)
	}
	if next == ']' {
		return skipOSC(text, start+2)
	}
	return start + 2
}

func (m *Manager) handleCSI(s *Session, text string, start int) int {
	for i := start; i < len(text); i++ {
		b := text[i]
		if b < 0x40 || b > 0x7e {
			continue
		}
		params := text[start:i]
		switch b {
		case 'K':
			m.eraseLine(s, params)
		case 'J':
			m.eraseDisplay(s, params)
		case 'A':
			m.setOutputRow(s, s.outputRow-firstANSIParam(params, 1))
		case 'B':
			m.setOutputRow(s, s.outputRow+firstANSIParam(params, 1))
		case 'C':
			m.setOutputCol(s, s.outputCol+firstANSIParam(params, 1))
		case 'D':
			m.setOutputCol(s, s.outputCol-firstANSIParam(params, 1))
		case 'E':
			m.setOutputRow(s, s.outputRow+firstANSIParam(params, 1))
			m.setOutputCol(s, 0)
		case 'F':
			m.setOutputRow(s, s.outputRow-firstANSIParam(params, 1))
			m.setOutputCol(s, 0)
		case 'd':
			m.setOutputRow(s, firstANSIParam(params, 1)-1)
		case 'G':
			m.setOutputCol(s, firstANSIParam(params, 1)-1)
		case 'H', 'f':
			row, col := cursorPosition(params)
			m.setOutputRow(s, row-1)
			m.setOutputCol(s, col-1)
		case 'm':
			if params == "" || params == "0" || params == "00" {
				s.CurrentSGR = ""
			} else {
				if s.CurrentSGR == "" {
					s.CurrentSGR = compactSGR(params)
				} else {
					s.CurrentSGR = compactSGR(s.CurrentSGR + ";" + params)
				}
			}
		case 's':
			s.savedRow = s.outputRow
			s.savedCol = s.outputCol
		case 'u':
			m.setOutputRow(s, s.savedRow)
			m.setOutputCol(s, s.savedCol)
		}
		return i + 1
	}
	return len(text)
}

func compactSGR(sgr string) string {
	parts := strings.Split(sgr, ";")
	var fg []string
	var bg []string
	var attrs []string

	for i := 0; i < len(parts); {
		p := parts[i]
		if p == "0" || p == "00" || p == "" {
			fg, bg, attrs = nil, nil, nil
			i++
			continue
		}
		if p == "38" || p == "48" {
			if i+2 < len(parts) && parts[i+1] == "5" {
				if p == "38" {
					fg = parts[i : i+3]
				} else {
					bg = parts[i : i+3]
				}
				i += 3
				continue
			}
			if i+4 < len(parts) && parts[i+1] == "2" {
				if p == "38" {
					fg = parts[i : i+5]
				} else {
					bg = parts[i : i+5]
				}
				i += 5
				continue
			}
		}

		val, _ := strconv.Atoi(p)
		if (val >= 30 && val <= 37) || (val >= 90 && val <= 97) || val == 39 {
			fg = []string{p}
		} else if (val >= 40 && val <= 47) || (val >= 100 && val <= 107) || val == 49 {
			bg = []string{p}
		} else {
			attrs = append(attrs, p)
		}
		i++
	}

	if len(attrs) > 10 {
		attrs = attrs[len(attrs)-10:]
	}

	var res []string
	res = append(res, attrs...)
	res = append(res, fg...)
	res = append(res, bg...)
	return strings.Join(res, ";")
}

func skipOSC(text string, start int) int {
	for i := start; i < len(text); i++ {
		if text[i] == '\a' {
			return i + 1
		}
		if text[i] == '\x1b' && i+1 < len(text) && text[i+1] == '\\' {
			return i + 2
		}
	}
	return len(text)
}

func (m *Manager) eraseLine(s *Session, params string) {
	m.ensureOutputLine(s)
	line := s.Cells[s.outputRow]
	switch params {
	case "2":
		line = nil
		s.outputCol = 0
	case "1":
		if s.outputCol > len(line) {
			s.outputCol = len(line)
		}
		for i := 0; i < s.outputCol; i++ {
			line[i] = Cell{Char: ' '}
		}
	default:
		if s.outputCol < len(line) {
			line = line[:s.outputCol]
		}
	}
	s.Cells[s.outputRow] = line
	s.Logs[s.outputRow] = renderCells(line)
}
func (m *Manager) eraseDisplay(s *Session, params string) {
	switch params {
	case "2", "3":
		s.Logs = nil
		s.Cells = nil
		s.outputRow = 0
		s.outputCol = 0
	case "1":
		// clear from start of screen to cursor
		if s.outputRow > 0 && s.outputRow < len(s.Logs) {
			s.Logs = s.Logs[s.outputRow:]
			s.Cells = s.Cells[s.outputRow:]
		}
		s.outputRow = 0
		m.ensureOutputLine(s)
		line := s.Cells[s.outputRow]
		for i := 0; i < s.outputCol && i < len(line); i++ {
			line[i] = Cell{Char: ' '}
		}
		s.Cells[s.outputRow] = line
		s.Logs[s.outputRow] = renderCells(line)
	case "0", "":
		// clear from cursor to end of screen
		m.ensureOutputLine(s)
		line := s.Cells[s.outputRow]
		if s.outputCol < len(line) {
			line = line[:s.outputCol]
		}
		s.Cells[s.outputRow] = line
		s.Logs[s.outputRow] = renderCells(line)
		if s.outputRow+1 < len(s.Logs) {
			s.Logs = s.Logs[:s.outputRow+1]
			s.Cells = s.Cells[:s.outputRow+1]
		}
	}
}

func renderCells(cells []Cell) string {
	var out strings.Builder
	currentSGR := ""
	for _, c := range cells {
		if c.SGR != currentSGR {
			out.WriteString("\x1b[0m")
			if c.SGR != "" {
				out.WriteString("\x1b[" + c.SGR + "m")
			}
			currentSGR = c.SGR
		}
		out.WriteRune(c.Char)
	}
	if currentSGR != "" {
		out.WriteString("\x1b[0m")
	}
	return out.String()
}

func (m *Manager) writeOutputRune(s *Session, r rune) {
	m.ensureOutputLine(s)
	line := s.Cells[s.outputRow]
	for len(line) < s.outputCol {
		line = append(line, Cell{Char: ' '})
	}
	cell := Cell{Char: r, SGR: s.CurrentSGR}
	if s.outputCol < len(line) {
		line[s.outputCol] = cell
	} else {
		line = append(line, cell)
	}
	s.outputCol++
	s.Cells[s.outputRow] = line
	s.Logs[s.outputRow] = renderCells(line)
}

func (m *Manager) deleteLastLogRune(s *Session) {
	m.ensureOutputLine(s)
	if s.outputCol == 0 {
		return
	}
	line := s.Cells[s.outputRow]
	s.outputCol--
	if s.outputCol < len(line) {
		line = append(line[:s.outputCol], line[s.outputCol+1:]...)
	}
	s.Cells[s.outputRow] = line
	s.Logs[s.outputRow] = renderCells(line)
}

func (m *Manager) ensureOutputLine(s *Session) {
	if len(s.Logs) > 0 && strings.HasPrefix(s.Logs[len(s.Logs)-1], "[system]") && s.outputRow >= len(s.Logs)-1 {
		s.outputRow = len(s.Logs)
		s.outputCol = 0
	}
	for len(s.Logs) <= s.outputRow {
		m.appendLog(s, "")
		if maxLogs := m.effectiveMaxLogs(); len(s.Logs) >= maxLogs && s.outputRow >= maxLogs {
			s.outputRow = maxLogs - 1
		}
	}
	for len(s.Cells) < len(s.Logs) {
		s.Cells = append(s.Cells, nil)
	}
}

func (m *Manager) setOutputRow(s *Session, row int) {
	if row < 0 {
		row = 0
	}
	s.outputRow = row
	m.ensureOutputLine(s)
}

func (m *Manager) setOutputCol(s *Session, col int) {
	if col < 0 {
		col = 0
	}
	s.outputCol = col
}

func (m *Manager) effectiveMaxLogs() int {
	if m.maxLogs <= 0 {
		return DefaultMaxLogs
	}
	return m.maxLogs
}

func firstANSIParam(params string, fallback int) int {
	if params == "" {
		return fallback
	}
	first := params
	if idx := strings.IndexByte(params, ';'); idx >= 0 {
		first = params[:idx]
	}
	value, err := strconv.Atoi(first)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func cursorPosition(params string) (int, int) {
	if params == "" {
		return 1, 1
	}
	parts := strings.Split(params, ";")
	row, col := 1, 1
	if len(parts) > 0 && parts[0] != "" {
		if value, err := strconv.Atoi(parts[0]); err == nil && value > 0 {
			row = value
		}
	}
	if len(parts) > 1 && parts[1] != "" {
		if value, err := strconv.Atoi(parts[1]); err == nil && value > 0 {
			col = value
		}
	}
	return row, col
}

func logsAreBlank(logs []string) bool {
	for _, line := range logs {
		if line != "" {
			return false
		}
	}
	return true
}
func (m *Manager) appendLog(s *Session, line string) {
	maxLogs := m.maxLogs
	if maxLogs <= 0 {
		maxLogs = DefaultMaxLogs
	}
	if len(s.Logs) < maxLogs {
		s.Logs = append(s.Logs, line)
		for len(s.Cells) < len(s.Logs) {
			s.Cells = append(s.Cells, nil)
		}
		return
	}
	copy(s.Logs, s.Logs[1:])
	s.Logs[len(s.Logs)-1] = line
	if len(s.Cells) > 0 {
		copy(s.Cells, s.Cells[1:])
		s.Cells[len(s.Cells)-1] = nil
	}
}

func cloneSession(s Session) Session {
	s.Args = append([]string(nil), s.Args...)
	s.Logs = append([]string(nil), s.Logs...)
	s.Cells = cloneCells(s.Cells)
	return s
}

func cloneCells(cells [][]Cell) [][]Cell {
	if len(cells) == 0 {
		return nil
	}

	out := make([][]Cell, len(cells))
	for i := range cells {
		out[i] = append([]Cell(nil), cells[i]...)
	}
	return out
}
