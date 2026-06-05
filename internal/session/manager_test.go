package session

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/handyfun97/ottrta/internal/event"
)

func TestNewFakeManagerCreatesRequiredSessions(t *testing.T) {
	manager := NewFakeManager()
	sessions := manager.Sessions()

	want := []struct {
		name    string
		kind    SessionKind
		status  Status
		command string
		args    []string
	}{
		{name: "fake-claude-1", kind: SessionKindFake, status: StatusRunning},
		{name: "fake-claude-2", kind: SessionKindFake, status: StatusStopped},
		{name: "fake-pi", kind: SessionKindFake, status: StatusRunning},
		{name: "real-go-version", kind: SessionKindProcess, status: StatusStopped, command: "go", args: []string{"version"}},
	}
	if len(sessions) != len(want) {
		t.Fatalf("got %d sessions, want %d", len(sessions), len(want))
	}
	for i, want := range want {
		got := sessions[i]
		if got.Name != want.name || got.ID != want.name {
			t.Fatalf("session %d = %q/%q, want %q", i, got.ID, got.Name, want.name)
		}
		if got.Kind != want.kind || got.Status != want.status {
			t.Fatalf("session %q kind/status = %s/%s, want %s/%s", got.ID, got.Kind, got.Status, want.kind, want.status)
		}
		if got.Command != want.command {
			t.Fatalf("session %q command = %q, want %q", got.ID, got.Command, want.command)
		}
		if strings.Join(got.Args, "\x00") != strings.Join(want.args, "\x00") {
			t.Fatalf("session %q args = %v, want %v", got.ID, got.Args, want.args)
		}
	}
}

func TestToggleSelectedFakeSession(t *testing.T) {
	manager := NewFakeManager()

	if !manager.Toggle(0) {
		t.Fatal("toggle returned false for valid fake index")
	}
	s, _ := manager.Session(0)
	if s.Status != StatusStopped {
		t.Fatalf("status = %s, want stopped", s.Status)
	}

	if !manager.Toggle(0) {
		t.Fatal("second toggle returned false for valid fake index")
	}
	if s.Status != StatusRunning {
		t.Fatalf("status = %s, want running", s.Status)
	}

	if manager.Toggle(3) {
		t.Fatal("toggle returned true for process session")
	}
	if manager.Toggle(99) {
		t.Fatal("toggle returned true for invalid index")
	}
}

func TestAppendLogsToRunningFakeOnly(t *testing.T) {
	manager := NewFakeManager()
	process, _ := manager.SessionByID("real-go-version")
	process.Status = StatusRunning
	before := manager.Sessions()

	appended := manager.AppendLogsToRunning()
	if appended != 2 {
		t.Fatalf("appended = %d, want 2", appended)
	}
	after := manager.Sessions()

	if len(after[0].Logs) != len(before[0].Logs)+1 {
		t.Fatalf("running fake logs = %d, want %d", len(after[0].Logs), len(before[0].Logs)+1)
	}
	if len(after[1].Logs) != len(before[1].Logs) {
		t.Fatalf("stopped fake logs = %d, want %d", len(after[1].Logs), len(before[1].Logs))
	}
	if len(after[2].Logs) != len(before[2].Logs)+1 {
		t.Fatalf("running fake logs = %d, want %d", len(after[2].Logs), len(before[2].Logs)+1)
	}
	if len(after[3].Logs) != len(before[3].Logs) {
		t.Fatalf("process logs = %d, want %d", len(after[3].Logs), len(before[3].Logs))
	}
	if !strings.Contains(after[0].Logs[len(after[0].Logs)-1], "fake-claude-1") {
		t.Fatalf("new log %q does not identify session", after[0].Logs[len(after[0].Logs)-1])
	}
}

func TestAppendLogTargetsSessionAndBoundsLogs(t *testing.T) {
	manager := NewManager([]Session{
		{ID: "one", Name: "one", Kind: SessionKindFake, Status: StatusRunning},
		{ID: "two", Name: "two", Kind: SessionKindFake, Status: StatusRunning},
	})
	manager.maxLogs = 2

	if !manager.AppendLog("one", "first") || !manager.AppendLog("one", "second") || !manager.AppendLog("one", "third") {
		t.Fatal("AppendLog returned false for existing session")
	}
	if manager.AppendLog("missing", "line") {
		t.Fatal("AppendLog returned true for missing session")
	}

	one, _ := manager.SessionByID("one")
	two, _ := manager.SessionByID("two")
	if got := strings.Join(one.Logs, ","); got != "second,third" {
		t.Fatalf("one logs = %q, want second,third", got)
	}
	if len(two.Logs) != 0 {
		t.Fatalf("two logs = %v, want none", two.Logs)
	}
}

func TestStartGoVersionProcessStreamsAndStops(t *testing.T) {
	manager := NewFakeManager()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	events, err := manager.StartSession(ctx, "real-go-version")
	if err != nil {
		t.Fatalf("StartSession returned error: %v", err)
	}
	s, _ := manager.SessionByID("real-go-version")
	if s.Status != StatusRunning {
		t.Fatalf("status after start = %s, want running", s.Status)
	}

	var sawVersion bool
	for {
		select {
		case msg, ok := <-events:
			if !ok {
				t.Fatal("events channel closed before exit message")
			}
			switch msg := msg.(type) {
			case event.SessionLogMsg:
				manager.AppendLog(msg.SessionID, msg.Line)
				if strings.Contains(msg.Line, "go version") {
					sawVersion = true
				}
			case event.SessionExitedMsg:
				manager.AppendExitLog(msg.SessionID, msg.Err)
				manager.MarkStopped(msg.SessionID)
				if !sawVersion {
					t.Fatalf("go version output was not streamed: %v", s.Logs)
				}
				if s.Status != StatusStopped {
					t.Fatalf("status after exit = %s, want stopped", s.Status)
				}
				if got := strings.Join(s.Logs, "\n"); !strings.Contains(got, "[system] exited with code 0") {
					t.Fatalf("logs do not include exit code: %v", s.Logs)
				}
				return
			}
		case <-ctx.Done():
			t.Fatalf("timed out waiting for process events: %v", s.Logs)
		}
	}
}

func TestStopSessionErrorPaths(t *testing.T) {
	manager := NewFakeManager()

	if err := manager.StopSession("missing"); err == nil {
		t.Fatal("StopSession missing session error = nil")
	}
	if err := manager.StopSession("fake-claude-1"); err == nil {
		t.Fatal("StopSession fake session error = nil")
	}
	if err := manager.StopSession("real-go-version"); err == nil {
		t.Fatal("StopSession stopped process error = nil")
	}
}
