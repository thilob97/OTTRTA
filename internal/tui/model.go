package tui

import (
	"context"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/handyfun97/ottrta/internal/event"
	"github.com/handyfun97/ottrta/internal/session"
	"github.com/handyfun97/ottrta/internal/task"
	"time"
)

type focusPanel int

const (
	focusTasks focusPanel = iota
	focusSessions
	focusLogs
)

type UIMode string

const (
	UIModeMonitor  UIMode = "monitor"
	UIModeAttach   UIMode = "attach"
	UIModeRename   UIMode = "rename"
	UIModeNewAgent UIMode = "new-agent"
)

const animationInterval = 450 * time.Millisecond

type animationTickMsg struct{}

type Model struct {
	manager                session.Manager
	taskManager            *task.Manager
	selectedSession        int
	selectedTask           int
	focus                  focusPanel
	width                  int
	height                 int
	mode                   UIMode
	attachedSessionID      string
	renameSessionID        string
	renameInput            string
	newAgentWorkDirInput   string
	newAgentCompletionHint string
	animationFrame         int
	storePath              string
	processEvents          map[string]<-chan event.ProcessMsg
	ptyEvents              map[string]<-chan event.PTYMsg
}

func NewModel() Model {
	storePath, err := session.DefaultStorePath()
	if err != nil {
		return newDefaultModel("", err)
	}
	return newModelWithStorePath(storePath)
}

func newModelWithStorePath(storePath string) Model {
	if storePath != "" {
		sessions, err := session.LoadSessions(storePath)
		if err == nil && len(sessions) > 0 {
			sessionMgr := session.NewManager(sessions)
			taskMgr := task.NewManager(&sessionMgr)
			return newBaseModel(sessionMgr, taskMgr, storePath)
		}
		if err != nil {
			return newDefaultModel(storePath, err)
		}
	}
	return newDefaultModel(storePath, nil)
}

func newDefaultModel(storePath string, loadErr error) Model {
	sessionMgr := session.NewManager(nil)
	taskMgr := task.NewManager(&sessionMgr)

	// Create default task with 3 omp sessions
	agentReq := task.AgentRequest{Kind: session.AgentKindOmp, Count: 3}
	taskMgr.CreateTask("Demo race task", task.TaskModeRace, agentReq, "")

	m := newBaseModel(sessionMgr, taskMgr, storePath)
	if loadErr != nil {
		sessions := m.manager.Sessions()
		if len(sessions) > 0 {
			m.manager.AppendLog(sessions[0].ID, "[system] failed to load sessions: "+loadErr.Error())
		}
	}
	return m
}

func newBaseModel(sessionMgr session.Manager, taskMgr *task.Manager, storePath string) Model {
	return Model{
		manager:       sessionMgr,
		taskManager:   taskMgr,
		selectedTask:  0,
		focus:         focusSessions,
		mode:          UIModeMonitor,
		storePath:     storePath,
		processEvents: make(map[string]<-chan event.ProcessMsg),
		ptyEvents:     make(map[string]<-chan event.PTYMsg),
	}
}

func Run() error {
	finalModel, err := tea.NewProgram(NewModel(), tea.WithAltScreen()).Run()
	if err != nil {
		return err
	}
	m, ok := finalModel.(Model)
	if !ok || m.storePath == "" {
		return nil
	}
	return session.SaveSessions(m.storePath, m.manager.Sessions())
}

func (m Model) Init() tea.Cmd {
	return tickAnimation()
}

func tickAnimation() tea.Cmd {
	return tea.Tick(animationInterval, func(time.Time) tea.Msg {
		return animationTickMsg{}
	})
}

func processContext() context.Context {
	return context.Background()
}

func pollProcessEvent(sessionID string, events <-chan event.ProcessMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-events
		if !ok {
			return event.SessionExitedMsg{SessionID: sessionID}
		}
		return msg
	}
}

func pollPTYEvent(sessionID string, events <-chan event.PTYMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-events
		if !ok {
			return event.SessionPTYExitedMsg{SessionID: sessionID}
		}
		return msg
	}
}

func adaptPTYEvents(events <-chan session.PTYEvent) <-chan event.PTYMsg {
	out := make(chan event.PTYMsg, 128)
	go func() {
		defer close(out)
		for msg := range events {
			if msg.Err != nil || len(msg.Data) == 0 {
				out <- event.SessionPTYExitedMsg{SessionID: msg.SessionID, Err: msg.Err}
				continue
			}
			out <- event.SessionPTYOutputMsg{SessionID: msg.SessionID, Data: msg.Data}
		}
	}()
	return out
}
