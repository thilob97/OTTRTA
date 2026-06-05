package session

import (
	"strings"
	"testing"
)

func TestNewFakeManagerCreatesRequiredSessions(t *testing.T) {
	manager := NewFakeManager()
	sessions := manager.Sessions()

	want := []string{"fake-claude-1", "fake-claude-2", "fake-pi"}
	if len(sessions) != len(want) {
		t.Fatalf("got %d sessions, want %d", len(sessions), len(want))
	}
	for i, name := range want {
		if sessions[i].Name != name || sessions[i].ID != name {
			t.Fatalf("session %d = %q/%q, want %q", i, sessions[i].ID, sessions[i].Name, name)
		}
	}
}

func TestToggleSelectedSession(t *testing.T) {
	manager := NewFakeManager()

	if !manager.Toggle(0) {
		t.Fatal("toggle returned false for valid index")
	}
	s, _ := manager.Session(0)
	if s.Status != StatusStopped {
		t.Fatalf("status = %s, want stopped", s.Status)
	}

	if !manager.Toggle(0) {
		t.Fatal("second toggle returned false for valid index")
	}
	if s.Status != StatusRunning {
		t.Fatalf("status = %s, want running", s.Status)
	}

	if manager.Toggle(99) {
		t.Fatal("toggle returned true for invalid index")
	}
}

func TestAppendLogsToRunningOnly(t *testing.T) {
	manager := NewFakeManager()
	before := manager.Sessions()

	appended := manager.AppendLogsToRunning()
	if appended != 2 {
		t.Fatalf("appended = %d, want 2", appended)
	}
	after := manager.Sessions()

	if len(after[0].Logs) != len(before[0].Logs)+1 {
		t.Fatalf("running session logs = %d, want %d", len(after[0].Logs), len(before[0].Logs)+1)
	}
	if len(after[1].Logs) != len(before[1].Logs) {
		t.Fatalf("stopped session logs = %d, want %d", len(after[1].Logs), len(before[1].Logs))
	}
	if len(after[2].Logs) != len(before[2].Logs)+1 {
		t.Fatalf("running session logs = %d, want %d", len(after[2].Logs), len(before[2].Logs)+1)
	}
	if !strings.Contains(after[0].Logs[len(after[0].Logs)-1], "fake-claude-1") {
		t.Fatalf("new log %q does not identify session", after[0].Logs[len(after[0].Logs)-1])
	}
}

func TestLogsAreBounded(t *testing.T) {
	manager := NewManager([]Session{{ID: "one", Name: "one", Status: StatusRunning}})
	manager.maxLogs = 2

	manager.AppendLogsToRunning()
	manager.AppendLogsToRunning()
	manager.AppendLogsToRunning()

	s, _ := manager.Session(0)
	if len(s.Logs) != 2 {
		t.Fatalf("logs = %d, want 2", len(s.Logs))
	}
	if strings.Contains(s.Logs[0], "001") {
		t.Fatalf("oldest log was not evicted: %v", s.Logs)
	}
}
