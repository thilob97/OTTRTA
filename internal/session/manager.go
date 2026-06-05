package session

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/handyfun97/ottrta/internal/event"
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
		items[i] = sessions[i]
		if len(sessions[i].Args) > 0 {
			items[i].Args = append([]string(nil), sessions[i].Args...)
		}
		if len(sessions[i].Logs) > 0 {
			items[i].Logs = append([]string(nil), sessions[i].Logs...)
		}
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
		items[i] = m.sessions[i]
		if len(m.sessions[i].Args) > 0 {
			items[i].Args = append([]string(nil), m.sessions[i].Args...)
		}
		if len(m.sessions[i].Logs) > 0 {
			items[i].Logs = append([]string(nil), m.sessions[i].Logs...)
		}
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

func (m *Manager) ClampIndex(index int) int {
	if len(m.sessions) == 0 || index < 0 {
		return 0
	}
	if index >= len(m.sessions) {
		return len(m.sessions) - 1
	}
	return index
}

func (m *Manager) Toggle(index int) bool {
	s, ok := m.Session(index)
	if !ok || s.Kind != SessionKindFake {
		return false
	}
	if s.Status == StatusRunning {
		s.Status = StatusStopped
	} else {
		s.Status = StatusRunning
	}
	return true
}

func (m *Manager) AppendLogsToRunning() int {
	appended := 0
	for i := range m.sessions {
		if m.sessions[i].Kind != SessionKindFake || !m.sessions[i].Running() {
			continue
		}
		m.sessions[i].logCounter++
		line := fmt.Sprintf("[%03d] %s produced fake output", m.sessions[i].logCounter, m.sessions[i].Name)
		m.appendLog(&m.sessions[i], line)
		appended++
	}
	return appended
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
	return nil
}

func (m *Manager) StartPTYSession(ctx context.Context, id string, cols, rows int) (<-chan PTYEvent, error) {
	s, ok := m.SessionByID(id)
	if !ok {
		return nil, fmt.Errorf("session %q not found", id)
	}
	if s.Kind != SessionKindPTY {
		return nil, fmt.Errorf("session %q is not a PTY session", id)
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
	if s.Kind != SessionKindPTY {
		return fmt.Errorf("session %q is not a PTY session", id)
	}
	runtime, ok := m.ptys[id]
	if !ok {
		return fmt.Errorf("session %q is not running", id)
	}

	if err := runtime.Stop(); err != nil {
		m.appendLog(s, fmt.Sprintf("[system] stop failed: %v", err))
	}
	s.Status = StatusStopped
	m.appendLog(s, "[system] stopped")
	delete(m.ptys, id)
	return nil
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
		return command
	}
	return strings.Join(append([]string{command}, args...), " ")
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
		case 'G':
			m.setOutputCol(s, firstANSIParam(params, 1)-1)
		case 'H', 'f':
			row, col := cursorPosition(params)
			if row == 1 && col == 1 && logsAreBlank(s.Logs) {
				s.Logs = nil
			}
			m.setOutputRow(s, row-1)
			m.setOutputCol(s, col-1)
		}
		return i + 1
	}
	return len(text)
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
	line := []rune(s.Logs[s.outputRow])
	switch params {
	case "2":
		s.Logs[s.outputRow] = ""
		s.outputCol = 0
	case "1":
		if s.outputCol > len(line) {
			s.outputCol = len(line)
		}
		for i := 0; i < s.outputCol; i++ {
			line[i] = ' '
		}
		s.Logs[s.outputRow] = string(line)
	default:
		if s.outputCol < len(line) {
			s.Logs[s.outputRow] = string(line[:s.outputCol])
		}
	}
}

func (m *Manager) eraseDisplay(s *Session, params string) {
	if params != "2" && params != "3" {
		return
	}
	s.Logs = nil
	s.outputRow = 0
	s.outputCol = 0
}

func (m *Manager) writeOutputRune(s *Session, r rune) {
	m.ensureOutputLine(s)
	line := []rune(s.Logs[s.outputRow])
	for len(line) < s.outputCol {
		line = append(line, ' ')
	}
	if s.outputCol < len(line) {
		line[s.outputCol] = r
	} else {
		line = append(line, r)
	}
	s.outputCol++
	s.Logs[s.outputRow] = string(line)
}

func (m *Manager) deleteLastLogRune(s *Session) {
	m.ensureOutputLine(s)
	line := []rune(s.Logs[s.outputRow])
	if s.outputCol > 0 {
		s.outputCol--
	}
	if s.outputCol < len(line) {
		line = append(line[:s.outputCol], line[s.outputCol+1:]...)
	} else if len(line) > 0 {
		line = line[:len(line)-1]
	}
	s.Logs[s.outputRow] = string(line)
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
		return
	}
	copy(s.Logs, s.Logs[1:])
	s.Logs[len(s.Logs)-1] = line
}
