package agent

import "github.com/handyfun97/ottrta/internal/session"

type AgentSpec struct {
	ID      string
	Name    string
	Kind    session.AgentKind
	Command string
	Args    []string
	WorkDir string
}
