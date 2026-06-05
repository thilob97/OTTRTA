package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/handyfun97/ottrta/internal/event"
	"github.com/handyfun97/ottrta/internal/session"
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
	case event.SessionLogMsg:
		m.manager.AppendLog(msg.SessionID, msg.Line)
		return m, m.pollProcess(msg.SessionID)
	case event.SessionExitedMsg:
		m.manager.AppendExitLog(msg.SessionID, msg.Err)
		m.manager.MarkStopped(msg.SessionID)
		delete(m.processEvents, msg.SessionID)
		return m, nil
	default:
		return m, nil
	}
}

func (m Model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.manager.StopAllProcesses()
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
		return m.toggleSelected()
	}
	return m, nil
}

func (m Model) toggleSelected() (tea.Model, tea.Cmd) {
	s, ok := m.manager.Session(m.selected)
	if !ok {
		return m, nil
	}

	switch s.Kind {
	case session.SessionKindFake:
		m.manager.Toggle(m.selected)
		return m, nil
	case session.SessionKindProcess:
		if s.Status == session.StatusRunning {
			if err := m.manager.StopSession(s.ID); err != nil {
				m.manager.AppendLog(s.ID, fmt.Sprintf("[system] stop failed: %v", err))
			}
			return m, nil
		}

		events, err := m.manager.StartSession(processContext(), s.ID)
		if err != nil {
			m.manager.AppendLog(s.ID, fmt.Sprintf("[system] start failed: %v", err))
			return m, nil
		}
		if m.processEvents == nil {
			m.processEvents = make(map[string]<-chan event.ProcessMsg)
		}
		m.processEvents[s.ID] = events
		return m, pollProcessEvent(s.ID, events)
	default:
		return m, nil
	}
}

func (m Model) pollProcess(sessionID string) tea.Cmd {
	events, ok := m.processEvents[sessionID]
	if !ok {
		return nil
	}
	return pollProcessEvent(sessionID, events)
}
