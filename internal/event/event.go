package event

// ProcessMsg is a Bubble Tea-safe process event. Goroutines send these through
// channels; the TUI update loop owns all model mutation.
type ProcessMsg interface {
	processMsg()
}

// SessionLogMsg appends one output line to a process session.
type SessionLogMsg struct {
	SessionID string
	Line      string
}

func (SessionLogMsg) processMsg() {}

// SessionExitedMsg reports that a process session has exited.
type SessionExitedMsg struct {
	SessionID string
	Err       error
}

func (SessionExitedMsg) processMsg() {}

// PTYMsg is a Bubble Tea-safe PTY event. Goroutines send these through
// channels; the TUI update loop owns all model mutation.
type PTYMsg interface {
	ptyMsg()
}

// SessionPTYOutputMsg appends raw PTY output to a PTY session.
type SessionPTYOutputMsg struct {
	SessionID string
	Data      []byte
}

func (SessionPTYOutputMsg) ptyMsg() {}

// SessionPTYExitedMsg reports that a PTY-backed session has exited.
type SessionPTYExitedMsg struct {
	SessionID string
	Err       error
}

func (SessionPTYExitedMsg) ptyMsg() {}
