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
		m.resizeRunningPTYs()
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
	case event.SessionPTYOutputMsg:
		m.manager.AppendOutput(msg.SessionID, string(msg.Data))
		return m, m.pollPTY(msg.SessionID)
	case event.SessionPTYExitedMsg:
		m.manager.AppendExitLog(msg.SessionID, msg.Err)
		m.manager.MarkStopped(msg.SessionID)
		delete(m.ptyEvents, msg.SessionID)
		if m.attachedSessionID == msg.SessionID {
			m.mode = UIModeMonitor
			m.attachedSessionID = ""
		}
		return m, nil
	default:
		return m, nil
	}
}

func (m Model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.mode == UIModeAttach {
		return m.updateAttachKey(msg)
	}
	return m.updateMonitorKey(msg)
}

func (m Model) updateMonitorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
	case "l", "right":
		m.focus = focusLogs
	case "enter":
		return m.attachSelected()
	case " ", "space":
		return m.toggleSelected()
	}
	return m, nil
}

func (m Model) updateAttachKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyEsc {
		m.mode = UIModeMonitor
		m.attachedSessionID = ""
		return m, nil
	}

	data := keyToBytes(msg)
	if len(data) == 0 {
		return m, nil
	}
	if err := m.manager.WritePTYSession(m.attachedSessionID, data); err != nil {
		m.manager.AppendLog(m.attachedSessionID, fmt.Sprintf("[system] write failed: %v", err))
	}
	return m, nil
}

func (m Model) attachSelected() (tea.Model, tea.Cmd) {
	s, ok := m.manager.Session(m.selected)
	if !ok {
		return m, nil
	}
	if s.Kind == session.SessionKindPTY && s.Status == session.StatusRunning {
		m.mode = UIModeAttach
		m.attachedSessionID = s.ID
		return m, nil
	}
	m.focus = focusLogs
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
	case session.SessionKindPTY:
		if s.Status == session.StatusRunning {
			if err := m.manager.StopPTYSession(s.ID); err != nil {
				m.manager.AppendLog(s.ID, fmt.Sprintf("[system] stop failed: %v", err))
			}
			if m.attachedSessionID == s.ID {
				m.mode = UIModeMonitor
				m.attachedSessionID = ""
			}
			delete(m.ptyEvents, s.ID)
			return m, nil
		}

		events, err := m.manager.StartPTYSession(processContext(), s.ID, m.ptyCols(), m.ptyRows())
		if err != nil {
			m.manager.AppendLog(s.ID, fmt.Sprintf("[system] start failed: %v", err))
			return m, nil
		}
		if m.ptyEvents == nil {
			m.ptyEvents = make(map[string]<-chan event.PTYMsg)
		}
		ptyEvents := adaptPTYEvents(events)
		m.ptyEvents[s.ID] = ptyEvents
		return m, pollPTYEvent(s.ID, ptyEvents)
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

func (m Model) pollPTY(sessionID string) tea.Cmd {
	events, ok := m.ptyEvents[sessionID]
	if !ok {
		return nil
	}
	return pollPTYEvent(sessionID, events)
}

func (m Model) resizeRunningPTYs() {
	cols := m.ptyCols()
	rows := m.ptyRows()
	for _, s := range m.manager.Sessions() {
		if s.Kind != session.SessionKindPTY || s.Status != session.StatusRunning {
			continue
		}
		if err := m.manager.ResizePTYSession(s.ID, cols, rows); err != nil {
			m.manager.AppendLog(s.ID, fmt.Sprintf("[system] resize failed: %v", err))
		}
	}
}

func (m Model) ptyCols() int {
	width := m.width
	if width < 60 {
		width = 90
	}
	leftWidth := width / 3
	if leftWidth < 24 {
		leftWidth = 24
	}
	cols := width - leftWidth - 6
	if cols < 20 {
		return 20
	}
	return cols
}

func (m Model) ptyRows() int {
	height := m.height
	if height < 12 {
		height = 24
	}
	rows := height - 7
	if rows < 5 {
		return 5
	}
	return rows
}
