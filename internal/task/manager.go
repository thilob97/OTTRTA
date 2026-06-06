package task

import (
	"context"
	"fmt"
	"sync"

	"github.com/thilob97/ottrta/internal/session"
)

type Manager struct {
	mu       sync.RWMutex
	tasks    map[string]*Task
	sessions *session.Manager
}

func NewManager(sessionMgr *session.Manager) *Manager {
	return &Manager{
		tasks:    make(map[string]*Task),
		sessions: sessionMgr,
	}
}

func (m *Manager) CreateTask(title string, mode TaskMode, agentReq AgentRequest, workdir string) (*Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	taskID := fmt.Sprintf("task-%03d", len(m.tasks)+1)
	task := &Task{
		ID:         taskID,
		Title:      title,
		Mode:       mode,
		Status:     TaskStopped,
		SessionIDs: make([]string, 0, agentReq.Count),
	}

	for i := 1; i <= agentReq.Count; i++ {
		sessionID := fmt.Sprintf("%s-%s-%d", taskID, agentReq.Kind, i)
		s := session.Session{
			ID:        sessionID,
			Name:      session.RandomImpName(),
			Kind:      session.SessionKindAgent,
			Status:    session.StatusStopped,
			Command:   string(agentReq.Kind),
			WorkDir:   workdir,
			AgentKind: agentReq.Kind,
			TaskID:    taskID,
		}
		m.sessions.AddSession(s)
		task.SessionIDs = append(task.SessionIDs, sessionID)
	}

	m.tasks[taskID] = task
	return task, nil
}

func (m *Manager) StartTask(ctx context.Context, taskID string, cols, rows int) error {
	m.mu.Lock()
	task, ok := m.tasks[taskID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("task not found: %s", taskID)
	}
	task.Status = TaskRunning
	m.mu.Unlock()

	for _, sid := range task.SessionIDs {
		_, err := m.sessions.StartPTYSession(ctx, sid, cols, rows)
		if err != nil {
			return fmt.Errorf("failed to start session %s: %w", sid, err)
		}
	}

	return nil
}

func (m *Manager) StopTask(taskID string) error {
	m.mu.Lock()
	task, ok := m.tasks[taskID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("task not found: %s", taskID)
	}
	task.Status = TaskStopped
	m.mu.Unlock()

	for _, sid := range task.SessionIDs {
		m.sessions.StopPTYSession(sid)
	}

	return nil
}

func (m *Manager) ListTasks() []*Task {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tasks := make([]*Task, 0, len(m.tasks))
	for _, t := range m.tasks {
		tasks = append(tasks, t)
	}
	return tasks
}

func (m *Manager) TaskByID(id string) (*Task, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.tasks[id]
	return t, ok
}

func (m *Manager) UpdateTaskStatus(taskID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.tasks[taskID]
	if !ok {
		return
	}

	anyRunning := false
	allFailed := true
	anyFailed := false

	for _, sid := range task.SessionIDs {
		if s, ok := m.sessions.SessionByID(sid); ok {
			if s.Status == session.StatusRunning {
				anyRunning = true
				allFailed = false
			} else if s.Status == session.StatusFailed {
				anyFailed = true
			} else {
				allFailed = false
			}
		}
	}

	if anyRunning {
		task.Status = TaskRunning
	} else if allFailed && anyFailed {
		task.Status = TaskFailed
	} else {
		task.Status = TaskStopped
	}
}
