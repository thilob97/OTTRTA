package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/handyfun97/ottrta/internal/event"
	"github.com/handyfun97/ottrta/internal/session"
)

func TestKeyNavigationIsClamped(t *testing.T) {
	m := NewModel()

	m = updateForTest(t, m, keyRune('k'))
	if m.selected != 0 {
		t.Fatalf("selected = %d, want 0", m.selected)
	}

	m = updateForTest(t, m, keyRune('j'))
	m = updateForTest(t, m, keyRune('j'))
	m = updateForTest(t, m, keyRune('j'))
	m = updateForTest(t, m, keyRune('j'))
	if m.selected != 4 {
		t.Fatalf("selected = %d, want 4", m.selected)
	}
}

func TestSpaceTogglesSelectedFakeSession(t *testing.T) {
	m := NewModel()
	s, _ := m.manager.Session(0)
	if s.Status != session.StatusRunning {
		t.Fatalf("initial status = %s, want running", s.Status)
	}

	m = updateForTest(t, m, keyRune(' '))
	s, _ = m.manager.Session(0)
	if s.Status != session.StatusStopped {
		t.Fatalf("status = %s, want stopped", s.Status)
	}
}

func TestSpaceStartsProcessAndHandlesEvents(t *testing.T) {
	m := NewModel()
	m.selected = 3

	updated, cmd := m.Update(keyRune(' '))
	if cmd == nil {
		t.Fatal("space on process session did not return a process poll command")
	}
	m = modelFromUpdate(t, updated)
	s, _ := m.manager.SessionByID("real-go-version")
	if s.Status != session.StatusRunning {
		t.Fatalf("status after start = %s, want running", s.Status)
	}

	deadline := time.After(5 * time.Second)
	for s.Status != session.StatusStopped {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for process exit: %v", s.Logs)
		default:
		}
		msg := cmd()
		updated, cmd = m.Update(msg)
		m = modelFromUpdate(t, updated)
		s, _ = m.manager.SessionByID("real-go-version")
		if cmd == nil && s.Status != session.StatusStopped {
			t.Fatalf("poll command stopped before process exit: %v", s.Logs)
		}
	}

	logs := strings.Join(s.Logs, "\n")
	if !strings.Contains(logs, "go version") {
		t.Fatalf("logs do not contain go version output: %v", s.Logs)
	}
	if !strings.Contains(logs, "[system] exited with code 0") {
		t.Fatalf("logs do not contain exit code: %v", s.Logs)
	}
}

func TestSpaceStopsRunningProcessSession(t *testing.T) {
	m := NewModel()
	m.selected = 3

	updated, _ := m.Update(keyRune(' '))
	m = modelFromUpdate(t, updated)
	updated, _ = m.Update(keyRune(' '))
	m = modelFromUpdate(t, updated)

	s, _ := m.manager.SessionByID("real-go-version")
	if s.Status != session.StatusStopped {
		t.Fatalf("status after stop = %s, want stopped", s.Status)
	}
	if got := strings.Join(s.Logs, "\n"); !strings.Contains(got, "[system] stopped") {
		t.Fatalf("logs do not contain stopped message: %v", s.Logs)
	}
}

func TestFocusKeys(t *testing.T) {
	m := NewModel()

	m = updateForTest(t, m, keyRune('l'))
	if m.focus != focusLogs {
		t.Fatalf("focus = %d, want logs", m.focus)
	}

	m = updateForTest(t, m, keyRune('h'))
	if m.focus != focusSessions {
		t.Fatalf("focus = %d, want sessions", m.focus)
	}

	m = updateForTest(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.focus != focusLogs {
		t.Fatalf("focus = %d, want logs after enter", m.focus)
	}
}

func TestQuitKeyReturnsCommand(t *testing.T) {
	m := NewModel()

	_, cmd := m.Update(keyRune('q'))
	if cmd == nil {
		t.Fatal("q did not return a quit command")
	}
}

func TestFakeLogTickAppendsOnlyRunningLogs(t *testing.T) {
	m := NewModel()
	before := m.manager.Sessions()

	m = updateForTest(t, m, event.FakeLogTick{At: time.Unix(1, 0)})
	after := m.manager.Sessions()

	if len(after[0].Logs) != len(before[0].Logs)+1 {
		t.Fatalf("selected running session logs = %d, want %d", len(after[0].Logs), len(before[0].Logs)+1)
	}
	if len(after[1].Logs) != len(before[1].Logs) {
		t.Fatalf("stopped session logs = %d, want %d", len(after[1].Logs), len(before[1].Logs))
	}
	if len(after[3].Logs) != len(before[3].Logs) {
		t.Fatalf("process session logs = %d, want %d", len(after[3].Logs), len(before[3].Logs))
	}
	if len(after[4].Logs) != len(before[4].Logs) {
		t.Fatalf("PTY session logs = %d, want %d", len(after[4].Logs), len(before[4].Logs))
	}
}

func TestViewShowsSessionListAndSelectedLogsOnly(t *testing.T) {
	m := NewModel()
	m.width = 100
	m.height = 24
	m.manager = session.NewManager([]session.Session{
		{ID: "left", Name: "left", Kind: session.SessionKindFake, Status: session.StatusRunning, Logs: []string{"left-only"}},
		{ID: "right", Name: "right", Kind: session.SessionKindFake, Status: session.StatusRunning, Logs: []string{"right-only"}},
	})

	view := m.View()
	for _, want := range []string{"Sessions", "left", "right", "left-only"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view does not contain %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "right-only") {
		t.Fatalf("view shows non-selected session log:\n%s", view)
	}
}

func TestDisplayLogLinesSanitizesPTYControlSequences(t *testing.T) {
	lines := displayLogLines([]string{"\x1b[2J\x1b[H\r\nprompt> go version\r\n\x07"})
	got := strings.Join(lines, "\n")
	if strings.Contains(got, "\x1b") || strings.Contains(got, "\x07") || strings.Contains(got, "\r") {
		t.Fatalf("display log still contains terminal controls: %q", got)
	}
	if !strings.Contains(got, "prompt> go version") {
		t.Fatalf("display log lost printable output: %q", got)
	}
}

func TestEnterAttachesOnlyRunningPTYSession(t *testing.T) {
	m := NewModel()
	m.selected = 4

	m = updateForTest(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != UIModeMonitor || m.attachedSessionID != "" {
		t.Fatalf("stopped PTY attach mode = %s/%q, want monitor/empty", m.mode, m.attachedSessionID)
	}
	if m.focus != focusLogs {
		t.Fatalf("stopped PTY enter focus = %d, want logs", m.focus)
	}

	s, _ := m.manager.SessionByID("shell-1")
	s.Status = session.StatusRunning
	m.focus = focusSessions
	m = updateForTest(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != UIModeAttach || m.attachedSessionID != "shell-1" {
		t.Fatalf("running PTY attach mode = %s/%q, want attach/shell-1", m.mode, m.attachedSessionID)
	}
}

func TestAttachModeForwardsKeysAndEscDetaches(t *testing.T) {
	m := NewModel()
	s, _ := m.manager.SessionByID("shell-1")
	s.Status = session.StatusRunning
	m.mode = UIModeAttach
	m.attachedSessionID = "shell-1"

	updated, cmd := m.Update(keyRune('q'))
	if cmd != nil {
		t.Fatal("q in attach mode returned a command; want forwarded input")
	}
	m = modelFromUpdate(t, updated)
	s, _ = m.manager.SessionByID("shell-1")
	if got := strings.Join(s.Logs, "\n"); !strings.Contains(got, "[system] write failed:") {
		t.Fatalf("attach key did not attempt PTY write: %v", s.Logs)
	}

	m = updateForTest(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.mode != UIModeMonitor || m.attachedSessionID != "" {
		t.Fatalf("esc detach mode = %s/%q, want monitor/empty", m.mode, m.attachedSessionID)
	}
}

func TestSpaceStoppingPTYDoesNotQuitTUI(t *testing.T) {
	m := NewModel()
	m.selected = 4
	s, _ := m.manager.SessionByID("shell-1")
	s.Status = session.StatusRunning

	updated, cmd := m.Update(keyRune(' '))
	if cmd != nil {
		t.Fatal("space on running PTY returned a command; want no quit command")
	}
	m = modelFromUpdate(t, updated)
	if m.mode != UIModeMonitor {
		t.Fatalf("mode after PTY stop = %s, want monitor", m.mode)
	}
}

func TestPTYOutputMessagesCoalesceTypedCharacters(t *testing.T) {
	m := NewModel()
	s, _ := m.manager.SessionByID("shell-1")
	s.Status = session.StatusRunning
	s.Logs = []string{"[system] started PTY: shell"}

	m = updateForTest(t, m, event.SessionPTYOutputMsg{SessionID: "shell-1", Data: []byte("a")})
	m = updateForTest(t, m, event.SessionPTYOutputMsg{SessionID: "shell-1", Data: []byte("b")})
	m = updateForTest(t, m, event.SessionPTYOutputMsg{SessionID: "shell-1", Data: []byte("c")})

	s, _ = m.manager.SessionByID("shell-1")
	if got := strings.Join(s.Logs, "|"); got != "[system] started PTY: shell|abc" {
		t.Fatalf("PTY typed output logs = %q, want one coalesced line", got)
	}
}

func TestPTYMessagesAppendAndExit(t *testing.T) {
	m := NewModel()
	s, _ := m.manager.SessionByID("shell-1")
	s.Status = session.StatusRunning
	m.mode = UIModeAttach
	m.attachedSessionID = "shell-1"

	m = updateForTest(t, m, event.SessionPTYOutputMsg{SessionID: "shell-1", Data: []byte("hello\r\n")})
	s, _ = m.manager.SessionByID("shell-1")
	if got := strings.Join(s.Logs, "\n"); !strings.Contains(got, "hello") {
		t.Fatalf("PTY output not appended: %v", s.Logs)
	}

	m = updateForTest(t, m, event.SessionPTYExitedMsg{SessionID: "shell-1"})
	s, _ = m.manager.SessionByID("shell-1")
	if s.Status != session.StatusStopped {
		t.Fatalf("status after PTY exit = %s, want stopped", s.Status)
	}
	if m.mode != UIModeMonitor || m.attachedSessionID != "" {
		t.Fatalf("mode after attached PTY exit = %s/%q, want monitor/empty", m.mode, m.attachedSessionID)
	}
}

func TestFooterShowsMonitorAndAttachModes(t *testing.T) {
	m := NewModel()
	if footer := m.renderFooter(); !strings.Contains(footer, "MONITOR") {
		t.Fatalf("monitor footer = %q", footer)
	}
	m.mode = UIModeAttach
	m.attachedSessionID = "shell-1"
	if footer := m.renderFooter(); !strings.Contains(footer, "ATTACHED to shell-1 | esc detach") {
		t.Fatalf("attach footer = %q", footer)
	}
}

func updateForTest(t *testing.T, m Model, msg tea.Msg) Model {
	t.Helper()
	updated, _ := m.Update(msg)
	return modelFromUpdate(t, updated)
}

func modelFromUpdate(t *testing.T, updated tea.Model) Model {
	t.Helper()
	next, ok := updated.(Model)
	if !ok {
		t.Fatalf("updated model has type %T", updated)
	}
	return next
}

func keyRune(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}
