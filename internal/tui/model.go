package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/handyfun97/ottrta/internal/event"
	"github.com/handyfun97/ottrta/internal/session"
)

const defaultTickInterval = time.Second

type focusPanel int

const (
	focusSessions focusPanel = iota
	focusLogs
)

type UIMode string

const (
	UIModeMonitor UIMode = "monitor"
	UIModeAttach  UIMode = "attach"
)

type Model struct {
	manager           session.Manager
	selected          int
	focus             focusPanel
	width             int
	height            int
	tickInterval      time.Duration
	mode              UIMode
	attachedSessionID string
	processEvents     map[string]<-chan event.ProcessMsg
	ptyEvents         map[string]<-chan event.PTYMsg
}

func NewModel() Model {
	return Model{
		manager:       session.NewFakeManager(),
		focus:         focusSessions,
		tickInterval:  defaultTickInterval,
		mode:          UIModeMonitor,
		processEvents: make(map[string]<-chan event.ProcessMsg),
		ptyEvents:     make(map[string]<-chan event.PTYMsg),
	}
}

func Run() error {
	_, err := tea.NewProgram(NewModel()).Run()
	return err
}

func (m Model) Init() tea.Cmd {
	return tick(m.tickInterval)
}

func tick(interval time.Duration) tea.Cmd {
	if interval <= 0 {
		interval = defaultTickInterval
	}
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return event.FakeLogTick{At: t}
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
