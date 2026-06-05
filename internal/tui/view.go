package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/handyfun97/ottrta/internal/session"
	"unicode"
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

	leftWidth := width / 3
	if leftWidth < 24 {
		leftWidth = 24
	}
	rightWidth := width - leftWidth - 4
	if rightWidth < 30 {
		rightWidth = 30
	}
	panelHeight := height - 4
	if panelHeight < 8 {
		panelHeight = 8
	}

	left := panelStyle(m.focus == focusSessions).
		Width(leftWidth).
		Height(panelHeight).
		Render(m.renderSessionList(leftWidth))
	right := panelStyle(m.focus == focusLogs).
		Width(rightWidth).
		Height(panelHeight).
		Render(m.renderLogPanel(rightWidth, panelHeight))

	footer := mutedStyle.Render(m.renderFooter())
	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top, left, right),
		footer,
	)
}

func (m Model) renderSessionList(width int) string {
	sessions := m.manager.Sessions()
	lines := []string{titleStyle.Render("Sessions")}
	if len(sessions) == 0 {
		return strings.Join(append(lines, mutedStyle.Render("No sessions")), "\n")
	}

	for i := range sessions {
		marker := " "
		if i == m.selected {
			marker = ">"
		}
		status := renderStatus(sessions[i].Status)
		row := fmt.Sprintf("%s %-18s %s", marker, sessions[i].Name, status)
		if i == m.selected {
			row = selectedSessionStyle.Width(width - 2).Render(row)
		}
		lines = append(lines, row)
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderLogPanel(width int, height int) string {
	s, ok := m.manager.Session(m.selected)
	if !ok {
		return strings.Join([]string{titleStyle.Render("Logs"), mutedStyle.Render("No session selected")}, "\n")
	}

	lines := []string{titleStyle.Render(fmt.Sprintf("Logs: %s", s.Name))}
	if len(s.Logs) == 0 {
		return strings.Join(append(lines, mutedStyle.Render("No logs yet")), "\n")
	}

	visible := height - 3
	if visible < 1 {
		visible = 1
	}
	displayLogs := displayLogLines(s.Logs)
	if len(displayLogs) > visible {
		displayLogs = displayLogs[len(displayLogs)-visible:]
	}
	for _, line := range displayLogs {
		lines = append(lines, mutedStyle.Width(width-2).Render(line))
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
	stripped := ansi.Strip(text)
	stripped = strings.ReplaceAll(stripped, "\r\n", "\n")
	stripped = strings.ReplaceAll(stripped, "\r", "\n")

	out := strings.Builder{}
	out.Grow(len(stripped))
	for _, r := range stripped {
		switch {
		case r == '\n' || r == '\t':
			out.WriteRune(r)
		case unicode.IsControl(r):
			continue
		default:
			out.WriteRune(r)
		}
	}
	return out.String()
}

func (m Model) renderFooter() string {
	if m.mode == UIModeAttach {
		name := m.attachedSessionID
		if s, ok := m.manager.SessionByID(m.attachedSessionID); ok {
			name = s.Name
		}
		return fmt.Sprintf("ATTACHED to %s | esc detach", name)
	}
	return "MONITOR | j/k select  h/l focus  space toggle/start/stop  enter attach/open  q quit"
}

func renderStatus(status session.Status) string {
	switch status {
	case session.StatusRunning:
		return runningStyle.Render("running")
	case session.StatusStopped:
		return stoppedStyle.Render("stopped")
	default:
		return mutedStyle.Render(string(status))
	}
}
