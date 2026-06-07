package tui

import (
	"bytes"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestKeyToBytes(t *testing.T) {
	tests := []struct {
		name string
		msg  tea.KeyMsg
		want []byte
	}{
		{name: "enter", msg: tea.KeyMsg{Type: tea.KeyEnter}, want: []byte{'\r'}},
		{name: "backspace", msg: tea.KeyMsg{Type: tea.KeyBackspace}, want: []byte{0x7f}},
		{name: "space", msg: tea.KeyMsg{Type: tea.KeySpace}, want: []byte{' '}},
		{name: "tab", msg: tea.KeyMsg{Type: tea.KeyTab}, want: []byte{'\t'}},
		{name: "up", msg: tea.KeyMsg{Type: tea.KeyUp}, want: []byte("\x1b[A")},
		{name: "down", msg: tea.KeyMsg{Type: tea.KeyDown}, want: []byte("\x1b[B")},
		{name: "right", msg: tea.KeyMsg{Type: tea.KeyRight}, want: []byte("\x1b[C")},
		{name: "left", msg: tea.KeyMsg{Type: tea.KeyLeft}, want: []byte("\x1b[D")},
		{name: "ctrl-a", msg: tea.KeyMsg{Type: tea.KeyCtrlA}, want: []byte{0x01}},
		{name: "ctrl-c", msg: tea.KeyMsg{Type: tea.KeyCtrlC}, want: []byte{0x03}},
		{name: "ctrl-d", msg: tea.KeyMsg{Type: tea.KeyCtrlD}, want: []byte{0x04}},
		{name: "ctrl-e", msg: tea.KeyMsg{Type: tea.KeyCtrlE}, want: []byte{0x05}},
		{name: "ctrl-k", msg: tea.KeyMsg{Type: tea.KeyCtrlK}, want: []byte{0x0b}},
		{name: "ctrl-l", msg: tea.KeyMsg{Type: tea.KeyCtrlL}, want: []byte{0x0c}},
		{name: "ctrl-r", msg: tea.KeyMsg{Type: tea.KeyCtrlR}, want: []byte{0x12}},
		{name: "ctrl-u", msg: tea.KeyMsg{Type: tea.KeyCtrlU}, want: []byte{0x15}},
		{name: "ctrl-w", msg: tea.KeyMsg{Type: tea.KeyCtrlW}, want: []byte{0x17}},
		{name: "ctrl-z", msg: tea.KeyMsg{Type: tea.KeyCtrlZ}, want: []byte{0x1a}},
		{name: "runes", msg: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("go version")}, want: []byte("go version")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := keyToBytes(tt.msg); !bytes.Equal(got, tt.want) {
				t.Fatalf("keyToBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}
