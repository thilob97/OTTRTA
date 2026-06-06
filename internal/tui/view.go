package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/handyfun97/ottrta/internal/session"
	"github.com/handyfun97/ottrta/internal/task"
)

var impArts = [...][4]string{
	{
		`  (\_/)  `,
		` / ^_^ \ `,
		`(  "*"  )`,
		" `--`--` ",
	},
	{
		` /\_/\  `,
		`( o.o ) `,
		` / ^ \  `,
		`  v v   `,
	},
	{
		`  /\/\  `,
		` ( •• ) `,
		` /(><)\ `,
		`  "  "  `,
	},
	{
		`  .-.-. `,
		` ( 0_0 )`,
		`<(  :  )`,
		`  ^^ ^^ `,
	},
	{
		`  /\=/\ `,
		` ( -_- )`,
		` /|:::|\`,
		`  /   \ `,
	},
	{
		`  (\ /) `,
		`  (x_x) `,
		` <( " )>`,
		`  /_|_\ `,
	},
	{
		`  /^ ^\ `,
		` ( @ @ )`,
		` /  *  \`,
		`  m---m `,
	},
	{
		`  (\w/) `,
		` ( >.< )`,
		` /(   )\`,
		`  d   b `,
	},
	{
		`  _/\_  `,
		` ( $.$ )`,
		` /|slp|\`,
		`  /___\ `,
	},
	{
		`  (\/)  `,
		` (o_o)  `,
		` <(~~~)>`,
		`  /   \ `,
	},
	{
		`  /\_/\ `,
		` ( @_@ )`,
		` /{:::}\`,
		`   u u  `,
	},
	{
		`  (^^^) `,
		` ( -o- )`,
		` /|+++|\`,
		"  /`-'\\ ",
	},
}

var impColors = [...]lipgloss.Color{
	lipgloss.Color("39"),
	lipgloss.Color("42"),
	lipgloss.Color("45"),
	lipgloss.Color("81"),
	lipgloss.Color("99"),
	lipgloss.Color("135"),
	lipgloss.Color("171"),
	lipgloss.Color("202"),
	lipgloss.Color("208"),
	lipgloss.Color("213"),
	lipgloss.Color("220"),
	lipgloss.Color("228"),
}

const (
	sessionCardMaxWidth      = 20
	sessionPanelTargetWidth  = 2*sessionCardMaxWidth + 7
	sessionPanelMinimumWidth = 42
	logPanelMinimumWidth     = 30
)

func (m Model) View() string {
	width := m.width
	if width < 60 {
		width = 90
	}
	height := m.height
	if height < 12 {
		height = 24
	}

	// Reserve space for footer and margins
	footerHeight := 1
	panelHeight := height - footerHeight - 2
	if panelHeight < 8 {
		panelHeight = 8
	}

	if m.mode == UIModeAttach {
		return m.renderAttachView(width, height, panelHeight)
	}

	leftWidth := sessionPanelTargetWidth
	if leftWidth < sessionPanelMinimumWidth {
		leftWidth = sessionPanelMinimumWidth
	}
	rightWidth := width - leftWidth - 4
	if rightWidth < logPanelMinimumWidth {
		rightWidth = logPanelMinimumWidth
		leftWidth = width - rightWidth - 4
		if leftWidth < sessionPanelMinimumWidth {
			leftWidth = sessionPanelMinimumWidth
		}
	}

	taskInfo := m.renderTaskInfo()
	taskInfoLines := strings.Count(taskInfo, "\n") + 1
	// Calculate available height for session list (panel height minus task info, title, and margins)
	availableSessionHeight := panelHeight - taskInfoLines - 2 // -2 for "Sessions" title and newline
	if availableSessionHeight < 5 {
		availableSessionHeight = 5
	}
	
	left := panelStyle(m.focus == focusSessions).
		Width(leftWidth).
		Height(panelHeight).
		Render(taskInfo + "\n" + m.renderSessionList(leftWidth, availableSessionHeight))
	right := termPanelStyle(m.focus == focusLogs).
		Width(rightWidth).
		Height(panelHeight).
		Render(m.renderLogPanel(rightWidth, panelHeight))

	footer := mutedStyle.Render(m.renderFooter())
	view := lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top, left, right),
		footer,
	)
	if modal := m.renderInputModal(); modal != "" {
		return centerOverlay(view, modal, width, height)
	}
	return view
}

func (m Model) renderAttachView(width, height, panelHeight int) string {
	s, ok := m.manager.Session(m.selectedSession)
	if !ok {
		return ""
	}

	termWidth := width - 4
	if termWidth < 30 {
		termWidth = 30
	}
	contentWidth := termWidth - 4

	headerLeft := titleStyle.Render(fmt.Sprintf(" ATTACHED: %s ", s.Name))
	headerRight := mutedStyle.Render("  esc detach")
	headerPad := contentWidth - lipgloss.Width(headerLeft) - lipgloss.Width(headerRight)
	if headerPad < 0 {
		headerPad = 0
	}
	header := headerLeft + strings.Repeat(" ", headerPad) + headerRight

	visible := panelHeight - 1
	if visible < 1 {
		visible = 1
	}
	displayLogs := displayLogLines(s.Logs)
	if len(displayLogs) > visible {
		displayLogs = displayLogs[len(displayLogs)-visible:]
	}

	var lines []string
	for _, line := range displayLogs {
		lines = append(lines, truncateANSI(line, contentWidth))
	}
	for len(lines) < visible {
		lines = append(lines, "")
	}

	content := strings.Join(lines, "\n")
	term := attachPanelStyle.
		Width(termWidth).
		Height(panelHeight).
		Render(header + "\n" + content)

	return lipgloss.JoinVertical(lipgloss.Left, term)
}

func (m Model) renderSessionList(width int, maxHeight int) string {
	sessions := m.manager.Sessions()
	lines := []string{titleStyle.Render("Sessions")}
	if len(sessions) == 0 {
		return strings.Join(append(lines, mutedStyle.Render("No sessions")), "\n")
	}

	cardWidth := (width - 7) / 2
	if cardWidth > sessionCardMaxWidth {
		cardWidth = sessionCardMaxWidth
	}
	if cardWidth < 16 {
		cardWidth = 16
	}
	
	usedHeight := 1 // "Sessions" title
	for i := 0; i < len(sessions); i += 2 {
		left := m.renderSessionCard(&sessions[i], cardWidth, i == m.selectedSession)
		leftHeight := strings.Count(left, "\n") + 1
		
		var row string
		var rowHeight int
		if i+1 >= len(sessions) {
			row = left
			rowHeight = leftHeight
		} else {
			right := m.renderSessionCard(&sessions[i+1], cardWidth, i+1 == m.selectedSession)
			row = lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)
			rightHeight := strings.Count(right, "\n") + 1
			if leftHeight > rightHeight {
				rowHeight = leftHeight
			} else {
				rowHeight = rightHeight
			}
		}
		
		if maxHeight > 0 && usedHeight+rowHeight > maxHeight {
			break
		}
		lines = append(lines, row)
		usedHeight += rowHeight
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderSessionCard(s *session.Session, width int, selected bool) string {
	status := renderStatus(s.Status)
	if s.NeedsAttention {
		status += " !"
	}

	rows := []string{impArt(s.ID, s.Status, m.animationFrame)}
	rows = append(rows, cardNameLines(s.Name, width-4)...)
	rows = append(rows, mutedStyle.Render(s.ID), status)
	style := sessionCardStyle
	if selected {
		style = selectedSessionCardStyle
	}
	return style.Width(width).Render(strings.Join(rows, "\n"))
}

var runningImpBanners = [...]string{
	"* slop *",
	"✦ slop ✦",
	"> slop <",
	"~ slop ~",
}

func impArt(sessionID string, status session.Status, frame int) string {
	index := stableImpIndex(sessionID)
	avatar := impArts[index]
	color := impColors[index%len(impColors)]
	banner := "       "
	if status == session.StatusStopped {
		banner = " zZzZ  "
	} else if status == session.StatusRunning {
		banner = runningImpBanners[frame%len(runningImpBanners)]
	} else if status == session.StatusFailed {
		banner = "  !!!  "
	}
	art := banner + "\n" + strings.Join(avatar[:], "\n")
	return fmt.Sprintf("\x1b[38;5;%sm%s\x1b[0m", string(color), art)
}

func cardNameLines(name string, width int) []string {
	if width < 8 {
		width = 8
	}
	if lipgloss.Width(name) <= width {
		return []string{titleStyle.Render(name)}
	}
	words := strings.Fields(name)
	if len(words) < 2 {
		return []string{titleStyle.Render(truncateANSI(name, width))}
	}
	first := words[0]
	second := strings.Join(words[1:], " ")
	return []string{
		titleStyle.Render(truncateANSI(first, width)),
		titleStyle.Render(truncateANSI(second, width)),
	}
}

func stableImpIndex(sessionID string) int {
	if len(impArts) == 1 {
		return 0
	}
	var hash uint32 = 2166136261
	for i := 0; i < len(sessionID); i++ {
		hash ^= uint32(sessionID[i])
		hash *= 16777619
	}
	return int(hash % uint32(len(impArts)))
}

func (m Model) renderTaskInfo() string {
	tasks := m.taskManager.ListTasks()
	if len(tasks) == 0 {
		return ""
	}

	var lines []string
	for i, t := range tasks {
		marker := " "
		if m.focus == focusTasks && i == m.selectedTask {
			marker = ">"
		}
		attention := ""
		for _, sid := range t.SessionIDs {
			if s, ok := m.manager.SessionByID(sid); ok && s.NeedsAttention {
				attention = " !"
				break
			}
		}
		status := renderTaskStatus(t.Status)
		lines = append(lines, fmt.Sprintf("%s %s %s%s", marker, t.ID, status, attention))
	}

	return titleStyle.Render("Tasks") + "\n" + strings.Join(lines, "\n") + "\n"
}

func renderTaskStatus(status task.TaskStatus) string {
	switch status {
	case task.TaskRunning:
		return runningStyle.Render("running")
	case task.TaskStopped:
		return stoppedStyle.Render("stopped")
	case task.TaskFailed:
		return failedStyle.Render("failed")
	case task.TaskDone:
		return "done"
	default:
		return string(status)
	}
}

func (m Model) renderLogPanel(width int, height int) string {
	s, ok := m.manager.Session(m.selectedSession)
	if !ok {
		return mutedStyle.Render("No session selected")
	}

	contentWidth := width - 4
	visible := height - 2
	if visible < 1 {
		visible = 1
	}

	headerLeft := titleStyle.Render(fmt.Sprintf(" %s ", s.Name))
	headerRight := renderStatus(s.Status)
	if s.NeedsAttention {
		headerRight += " !"
	}
	headerPad := contentWidth - lipgloss.Width(headerLeft) - lipgloss.Width(headerRight)
	if headerPad < 0 {
		headerPad = 0
	}
	header := headerLeft + strings.Repeat(" ", headerPad) + headerRight

	displayLogs := displayLogLines(s.Logs)
	if len(displayLogs) > visible {
		displayLogs = displayLogs[len(displayLogs)-visible:]
	}

	var lines []string
	for _, line := range displayLogs {
		lines = append(lines, truncateANSI(line, contentWidth))
	}
	for len(lines) < visible {
		lines = append(lines, "")
	}

	content := strings.Join(lines, "\n")
	return header + "\n" + content
}

func displayLogLines(logs []string) []string {
	if len(logs) == 0 {
		return []string{mutedStyle.Render("No logs yet")}
	}
	sanitized := make([]string, len(logs))
	for i, log := range logs {
		sanitized[i] = sanitizeLogText(log)
	}
	return sanitized
}

func sanitizeLogText(text string) string {
	text = strings.ReplaceAll(text, "\r", "")
	text = strings.ReplaceAll(text, "\t", "    ")
	var builder strings.Builder
	for _, r := range text {
		if r >= 32 && r != 127 {
			builder.WriteRune(r)
		} else if r == '\n' {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

// truncateANSI truncates a string containing ANSI escapes to a visible width.
func truncateANSI(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	visible := 0
	var out strings.Builder
	inEscape := false
	for _, r := range s {
		if inEscape {
			out.WriteRune(r)
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEscape = false
			}
			continue
		}
		if r == '\x1b' {
			inEscape = true
			out.WriteRune(r)
			continue
		}
		if visible >= maxWidth {
			break
		}
		out.WriteRune(r)
		visible++
	}
	result := out.String()
	if strings.Contains(result, "\x1b[") {
		result += "\x1b[0m"
	}
	return result
}

func (m Model) renderInputModal() string {
	switch m.mode {
	case UIModeRename:
		name := m.renameSessionID
		if s, ok := m.manager.SessionByID(m.renameSessionID); ok {
			name = s.Name
		}
		return renderModal("Rename session", "Current: "+name, "Name", m.renameInput, "", "enter save  esc cancel")
	case UIModeNewAgent:
		return renderModal("New OMP agent", "Create a stopped agent session.", "Working directory", m.newAgentWorkDirInput, m.newAgentCompletionHint, "tab complete  enter create  esc cancel")
	default:
		return ""
	}
}

func renderModal(title, subtitle, label, value, hint, help string) string {
	const width = 52
	input := value
	if input == "" {
		input = mutedStyle.Render("(default current directory)")
	}
	lines := []string{
		titleStyle.Render(title),
		mutedStyle.Render(subtitle),
		"",
		label,
		"› " + input,
	}
	if hint != "" {
		lines = append(lines, mutedStyle.Render(hint))
	}
	lines = append(lines, "", mutedStyle.Render(help))
	body := strings.Join(lines, "\n")
	return modalStyle.Width(width).Render(body)
}

func centerOverlay(base, overlay string, width, height int) string {
	baseLines := strings.Split(base, "\n")
	overlayLines := strings.Split(overlay, "\n")
	if len(baseLines) < height {
		for len(baseLines) < height {
			baseLines = append(baseLines, "")
		}
	}

	overlayWidth := 0
	for _, line := range overlayLines {
		if lineWidth := lipgloss.Width(line); lineWidth > overlayWidth {
			overlayWidth = lineWidth
		}
	}

	row := (height - len(overlayLines)) / 2
	if row < 0 {
		row = 0
	}
	col := (width - overlayWidth) / 2
	if col < 0 {
		col = 0
	}

	for i, line := range overlayLines {
		target := row + i
		if target >= len(baseLines) {
			break
		}
		baseLines[target] = strings.Repeat(" ", col) + line
	}
	return strings.Join(baseLines, "\n")
}

func (m Model) renderFooter() string {
	if m.mode == UIModeAttach {
		name := m.attachedSessionID
		if s, ok := m.manager.SessionByID(m.attachedSessionID); ok {
			name = s.Name
		}
		return fmt.Sprintf("ATTACHED to %s | esc detach", name)
	}
	if m.mode == UIModeRename {
		return "RENAME | enter save  esc cancel"
	}
	if m.mode == UIModeNewAgent {
		return "NEW AGENT | tab complete  enter create  esc cancel"
	}
	if s, ok := m.manager.Session(m.selectedSession); ok && s.NeedsAttention {
		return "selected session needs attention: press enter to attach"
	}
	return "MONITOR | j/k select  h/l focus  space start/stop  enter attach  n new agent  r rename  x remove  q quit"
}

func renderStatus(status session.Status) string {
	switch status {
	case session.StatusRunning:
		return runningStyle.Render("running")
	case session.StatusStopped:
		return stoppedStyle.Render("stopped")
	case session.StatusFailed:
		return failedStyle.Render("failed")
	default:
		return string(status)
	}
}
