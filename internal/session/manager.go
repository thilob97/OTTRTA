package session

import (
	"context"
	"errors"
	"fmt"
	"os/exec"

	"github.com/handyfun97/ottrta/internal/event"
)

const DefaultMaxLogs = 80

// Manager owns session state and process runtimes for the TUI.
type Manager struct {
	sessions []Session
	maxLogs  int
	runtimes map[string]*ProcessRuntime
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

func (m *Manager) MarkStopped(id string) bool {
	s, ok := m.SessionByID(id)
	if !ok {
		return false
	}
	s.Status = StatusStopped
	delete(m.runtimes, id)
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

func (m *Manager) StopAllProcesses() {
	for id, runtime := range m.runtimes {
		runtime.Stop()
		if s, ok := m.SessionByID(id); ok {
			s.Status = StatusStopped
			m.appendLog(s, "[system] stopped")
		}
		delete(m.runtimes, id)
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
