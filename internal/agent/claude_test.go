package agent

import "testing"

func TestCheckAttention(t *testing.T) {
	tests := []struct {
		output string
		want   bool
	}{
		{"Hello there", false},
		{"Do you want to proceed?", true},
		{"Please Allow this action", true},
		{"I need to approve this.", true},
		{"Yes", true},
		{"No", true},
		{"Random text", false},
	}

	for _, tt := range tests {
		got := CheckAttention(tt.output)
		if got != tt.want {
			t.Errorf("CheckAttention(%q) = %v, want %v", tt.output, got, tt.want)
		}
	}
}

func TestOmpExists(t *testing.T) {
	// this might return true or false depending on the runner environment,
	// but it shouldn't panic.
	_ = OmpExists()
}
