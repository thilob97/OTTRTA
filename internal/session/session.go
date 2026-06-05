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
	SessionKindPTY     SessionKind = "pty"
)

// Session is the user-visible state for fake, process, or PTY-backed sessions.
// Runtime handles live in Manager.
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
	outputCol  int
	outputRow  int
}

func (s Session) Running() bool {
	return s.Status == StatusRunning
}
