package event

import "time"

// FakeLogTick is the Bubble Tea message used to append fake log output.
type FakeLogTick struct {
	At time.Time
}
