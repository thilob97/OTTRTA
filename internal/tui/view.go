package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/handyfun97/ottrta/internal/session"
	"github.com/handyfun97/ottrta/internal/task"
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

	panelHeight := height - 3
	if panelHeight < 8 {
		panelHeight = 8
	}

	if m.mode == UIModeAttach {
		return m.renderAttachView(width, height, panelHeight)
	}

	leftWidth := width / 3
	if leftWidth < 24 {
		leftWidth = 24
	}
	rightWidth := width - leftWidth - 4
	if rightWidth < 30 {
		rightWidth = 30
	}

	taskInfo := m.renderTaskInfo()
	left := panelStyle(m.focus == focusSessions).
		Width(leftWidth).
		Height(panelHeight).
		Render(taskInfo + "\n" + m.renderSessionList(leftWidth))
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

func (m Model) renderSessionList(width int) string {
	sessions := m.manager.Sessions()
	lines := []string{titleStyle.Render("Sessions")}
	if len(sessions) == 0 {
		return strings.Join(append(lines, mutedStyle.Render("No sessions")), "\n")
	}

	for i := range sessions {
		marker := " "
		if i == m.selectedSession {
			marker = ">"
		}
		status := renderStatus(sessions[i].Status)
		if sessions[i].NeedsAttention {
			status += "  !"
		}
		row := fmt.Sprintf("%s %-18s %s", marker, sessions[i].Name, status)
		if i == m.selectedSession {
			row = selectedSessionStyle.Width(width - 2).Render(row)
		}
		lines = append(lines, row)
	}
	return strings.Join(lines, "\n")
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

	header := titleStyle.Render(s.Name)
	statusStr := renderStatus(s.Status)
	header += " " + statusStr
	if s.WorkDir != "" {
		header += " " + mutedStyle.Render("cwd: "+s.WorkDir)
	}

	if len(s.Logs) == 0 {
		return header + "\n" + mutedStyle.Render("No output yet")
	}

	visible := height - 3
	if visible < 1 {
		visible = 1
	}
	displayLogs := displayLogLines(s.Logs)
	if len(displayLogs) > visible {
		displayLogs = displayLogs[len(displayLogs)-visible:]
	}

	contentWidth := width - 4 // border(2) + padding(2)
	if contentWidth < 10 {
		contentWidth = 10
	}
	var lines []string
	lines = append(lines, header)
	for _, line := range displayLogs {
		if strings.HasPrefix(line, "[system]") {
			lines = append(lines, mutedStyle.Render(line))
		} else {
			lines = append(lines, truncateANSI(line, contentWidth))
		}
	}
	return strings.Join(lines, "\n")
}

func displayLogLines(logs []string) []string {
	lines := make([]string, 0, len(logs))
	for _, log := range logs {
		cleaned := sanitizeLogText(log)
		if cleaned == "" {
			lines = append(lines, "")
			continue
		}
		lines = append(lines, strings.Split(cleaned, "\n")...)
	}
	return lines
}

func sanitizeLogText(text string) string {
	stripped := strings.ReplaceAll(text, "\r\n", "\n")
	stripped = strings.ReplaceAll(stripped, "\r", "\n")

	out := strings.Builder{}
	out.Grow(len(stripped))
	for i := 0; i < len(stripped); {
		b := stripped[i]
		switch {
		case b == '\n' || b == '\t':
			out.WriteByte(b)
			i++
		case b == '\x1b':
			// preserve entire ANSI sequence
			j := i + 1
			if j < len(stripped) && stripped[j] == '[' {
				j++
				for j < len(stripped) && (stripped[j] < 0x40 || stripped[j] > 0x7e) {
					j++
				}
				if j < len(stripped) {
					j++ // include final byte
				}
			}
			out.WriteString(stripped[i:j])
			i = j
		case b < 0x20 && b != '\n' && b != '\t':
			i++ // skip other control chars
		default:
			out.WriteByte(b)
			i++
		}
	}
	return out.String()
}

// truncateANSI truncates a string containing ANSI escapes to a visible width.
func truncateANSI(s string, maxWidth int) string {
	var out strings.Builder
	visWidth := 0
	i := 0
	for i < len(s) && visWidth < maxWidth {
		if s[i] == '\x1b' {
			// copy entire escape sequence
			j := i + 1
			if j < len(s) && s[j] == '[' {
				j++
				for j < len(s) && (s[j] < 0x40 || s[j] > 0x7e) {
					j++
				}
				if j < len(s) {
					j++
				}
			}
			out.WriteString(s[i:j])
			i = j
		} else {
			out.WriteByte(s[i])
			if s[i] >= 0x20 {
				visWidth++
			}
			i++
		}
	}
	// always reset at the end to avoid color bleed
	if strings.Contains(s, "\x1b[") {
		out.WriteString("\x1b[0m")
	}
	return out.String()
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
		return mutedStyle.Render(string(status))
	}
}
