package tui

import tea "github.com/charmbracelet/bubbletea"

func keyToBytes(msg tea.KeyMsg) []byte {
	switch msg.Type {
	case tea.KeyRunes:
		return []byte(string(msg.Runes))
	case tea.KeySpace:
		return []byte{' '}
	case tea.KeyEnter:
		return []byte{'\r'}
	case tea.KeyBackspace, tea.KeyCtrlH:
		return []byte{0x7f}
	case tea.KeyTab:
		return []byte{'\t'}
	case tea.KeyUp:
		return []byte("\x1b[A")
	case tea.KeyDown:
		return []byte("\x1b[B")
	case tea.KeyRight:
		return []byte("\x1b[C")
	case tea.KeyLeft:
		return []byte("\x1b[D")
	case tea.KeyCtrlC:
		return []byte{0x03}
	case tea.KeyCtrlD:
		return []byte{0x04}
	default:
		return nil
	}
}
