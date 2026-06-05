package session

import "fmt"

const DefaultMaxLogs = 80

// Manager owns all fake sessions for the TUI.
type Manager struct {
	sessions []Session
	maxLogs  int
}

func NewManager(sessions []Session) Manager {
	items := make([]Session, len(sessions))
	for i := range sessions {
		items[i] = sessions[i]
		if len(sessions[i].Logs) > 0 {
			items[i].Logs = append([]string(nil), sessions[i].Logs...)
		}
	}

	return Manager{sessions: items, maxLogs: DefaultMaxLogs}
}

func (m *Manager) Count() int {
	return len(m.sessions)
}

func (m *Manager) Sessions() []Session {
	items := make([]Session, len(m.sessions))
	for i := range m.sessions {
		items[i] = m.sessions[i]
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
	if !ok {
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
		if !m.sessions[i].Running() {
			continue
		}
		m.sessions[i].logCounter++
		line := fmt.Sprintf("[%03d] %s produced fake output", m.sessions[i].logCounter, m.sessions[i].Name)
		m.appendLog(&m.sessions[i], line)
		appended++
	}
	return appended
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
