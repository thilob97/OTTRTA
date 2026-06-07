package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thilob97/ottrta/internal/event"
	"github.com/thilob97/ottrta/internal/session"
)

type focusPanel int

const (
	focusSessions focusPanel = iota
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
	selectedSession        int
	focus                  focusPanel
	width                  int
	height                 int
	mode                   UIMode
	attachedSessionID      string
	renameSessionID        string
	renameInput            string
	newAgentCommandInput   string
	newAgentWorkDirInput   string
	newAgentCompletionHint string
	newAgentStep           int
	animationFrame         int
	storePath              string
	processEvents          map[string]<-chan event.ProcessMsg
	ptyEvents              map[string]<-chan event.PTYMsg
	version                string
	updateAvailable        string
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
			return newBaseModel(sessionMgr, storePath)
		}
		if err != nil {
			return newDefaultModel(storePath, err)
		}
	}
	return newDefaultModel(storePath, nil)
}

func newDefaultModel(storePath string, loadErr error) Model {
	sessionMgr := session.NewManager(defaultSessions())
	m := newBaseModel(sessionMgr, storePath)
	if loadErr != nil {
		sessions := m.manager.Sessions()
		if len(sessions) > 0 {
			m.manager.AppendLog(sessions[0].ID, "[system] failed to load sessions: "+loadErr.Error())
		}
	}
	return m
}

func defaultSessions() []session.Session {
	return []session.Session{
		{
			ID:      "proc-1",
			Name:    "proc-1",
			Kind:    session.SessionKindProcess,
			Status:  session.StatusStopped,
			Command: "go",
			Args:    []string{"version"},
		},
		{
			ID:      "shell-1",
			Name:    "shell-1",
			Kind:    session.SessionKindPTY,
			Status:  session.StatusStopped,
			Command: session.DefaultShellCommand(),
		},
		{
			ID:        "omp-1",
			Name:      "omp-1",
			Kind:      session.SessionKindAgent,
			Status:    session.StatusStopped,
			Command:   "omp",
			AgentKind: session.AgentKindOmp,
		},
	}
}

func newBaseModel(sessionMgr session.Manager, storePath string) Model {
	return Model{
		manager:       sessionMgr,
		focus:         focusSessions,
		mode:          UIModeMonitor,
		storePath:     storePath,
		processEvents: make(map[string]<-chan event.ProcessMsg),
		ptyEvents:     make(map[string]<-chan event.PTYMsg),
	}
}

func Run(version string) error {
	m := NewModel()
	m.version = version
	finalModel, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	if err != nil {
		return err
	}
	m = finalModel.(Model)
	if m.storePath == "" {
		return nil
	}
	return session.SaveSessions(m.storePath, m.manager.Sessions())
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(tickAnimation(), m.checkForUpdateCmd())
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
