package session

// Status is the execution state of a fake session.
type Status string

const (
	StatusRunning Status = "running"
	StatusStopped Status = "stopped"
)

// Session is a small fake agent session. It deliberately does not model a real
// terminal or process; the MVP only needs names, state, and bounded logs.
type Session struct {
	ID         string
	Name       string
	Status     Status
	Logs       []string
	logCounter int
}

func (s Session) Running() bool {
	return s.Status == StatusRunning
}
