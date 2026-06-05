package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/handyfun97/ottrta/internal/event"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.updateKey(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case event.FakeLogTick:
		m.manager.AppendLogsToRunning()
		return m, tick(m.tickInterval)
	default:
		return m, nil
	}
}

func (m Model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "j", "down":
		m.selected = m.manager.ClampIndex(m.selected + 1)
	case "k", "up":
		m.selected = m.manager.ClampIndex(m.selected - 1)
	case "h", "left":
		m.focus = focusSessions
	case "l", "right", "enter":
		m.focus = focusLogs
	case " ", "space":
		m.manager.Toggle(m.selected)
	}
	return m, nil
}
