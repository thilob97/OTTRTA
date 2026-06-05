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
	if m.selected != 3 {
		t.Fatalf("selected = %d, want 3", m.selected)
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
