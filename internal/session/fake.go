package session

func NewFakeManager() Manager {
	return NewManager([]Session{
		{
			ID:     "fake-claude-1",
			Name:   "fake-claude-1",
			Status: StatusRunning,
			Logs:   []string{"fake-claude-1 ready"},
		},
		{
			ID:     "fake-claude-2",
			Name:   "fake-claude-2",
			Status: StatusStopped,
			Logs:   []string{"fake-claude-2 waiting"},
		},
		{
			ID:     "fake-pi",
			Name:   "fake-pi",
			Status: StatusRunning,
			Logs:   []string{"fake-pi ready"},
		},
	})
}
