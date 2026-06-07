package session

// Status is the execution state of a session.
type Status string

const (
	StatusRunning Status = "running"
	StatusStopped Status = "stopped"
	StatusFailed  Status = "failed"
)

type AgentKind string

const (
	AgentKindOmp AgentKind = "omp"
)

// SessionKind identifies how a session produces logs.
type SessionKind string

const (
	SessionKindProcess SessionKind = "process"
	SessionKindPTY     SessionKind = "pty"
	SessionKindAgent   SessionKind = "agent"
)

// Session is the user-visible state for fake, process, or PTY-backed sessions.
// Runtime handles live in Manager.
type Cell struct {
	Char rune
	SGR  string
}

type Session struct {
	ID         string
	Name       string
	Kind       SessionKind
	Status     Status
	Command    string
	Args       []string
	WorkDir    string
	AgentKind  AgentKind
	Logs       []string
	Cells      [][]Cell
	CurrentSGR string
	outputCol  int
	outputRow  int
	savedCol   int
	savedRow   int
}

func (s Session) Running() bool {
	return s.Status == StatusRunning
}
