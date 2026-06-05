package session

// Status is the execution state of a session.
type Status string

const (
	StatusRunning Status = "running"
	StatusStopped Status = "stopped"
)

// SessionKind identifies how a session produces logs.
type SessionKind string

const (
	SessionKindFake    SessionKind = "fake"
	SessionKindProcess SessionKind = "process"
)

// Session is the user-visible state for either a fake session or a real
// foreground process session. Runtime process handles live in Manager.
type Session struct {
	ID         string
	Name       string
	Kind       SessionKind
	Status     Status
	Command    string
	Args       []string
	WorkDir    string
	Logs       []string
	logCounter int
}

func (s Session) Running() bool {
	return s.Status == StatusRunning
}
