package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/handyfun97/ottrta/internal/session"
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

	footer := mutedStyle.Render("j/k select  h/l focus  space toggle  enter open  q quit")
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
	logs := s.Logs
	if len(logs) > visible {
		logs = logs[len(logs)-visible:]
	}
	for _, line := range logs {
		lines = append(lines, mutedStyle.Width(width-2).Render(line))
	}
	return strings.Join(lines, "\n")
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
