package event

import "time"

// FakeLogTick is the Bubble Tea message used to append fake log output.
type FakeLogTick struct {
	At time.Time
}

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
