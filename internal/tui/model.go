package tui

import (
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

type Model struct {
	manager      session.Manager
	selected     int
	focus        focusPanel
	width        int
	height       int
	tickInterval time.Duration
}

func NewModel() Model {
	return Model{
		manager:      session.NewFakeManager(),
		focus:        focusSessions,
		tickInterval: defaultTickInterval,
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
