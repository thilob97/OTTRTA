package session

func NewFakeManager() Manager {
	return NewManager([]Session{
		{
			ID:     "fake-claude-1",
			Name:   "fake-claude-1",
			Kind:   SessionKindFake,
			Status: StatusRunning,
			Logs:   []string{"fake-claude-1 ready"},
		},
		{
			ID:     "fake-claude-2",
			Name:   "fake-claude-2",
			Kind:   SessionKindFake,
			Status: StatusStopped,
			Logs:   []string{"fake-claude-2 waiting"},
		},
		{
			ID:     "fake-pi",
			Name:   "fake-pi",
			Kind:   SessionKindFake,
			Status: StatusRunning,
			Logs:   []string{"fake-pi ready"},
		},
		{
			ID:      "real-go-version",
			Name:    "real-go-version",
			Kind:    SessionKindProcess,
			Status:  StatusStopped,
			Command: "go",
			Args:    []string{"version"},
			Logs:    []string{"real-go-version ready"},
		},
		{
			ID:      "shell-1",
			Name:    "shell-1",
			Kind:    SessionKindPTY,
			Status:  StatusStopped,
			Command: DefaultShellCommand(),
			Logs:    []string{"shell-1 ready"},
		},
	})
}
