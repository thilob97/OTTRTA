package session

import (
	"context"
	"os"
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
		{name: "shell-1", kind: SessionKindPTY, status: StatusStopped, command: DefaultShellCommand()},
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
	if manager.Toggle(4) {
		t.Fatal("toggle returned true for PTY session")
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
	if len(after[4].Logs) != len(before[4].Logs) {
		t.Fatalf("PTY logs = %d, want %d", len(after[4].Logs), len(before[4].Logs))
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

func TestAppendOutputCoalescesPTYChunks(t *testing.T) {
	manager := NewManager([]Session{{
		ID:     "pty",
		Name:   "pty",
		Kind:   SessionKindPTY,
		Status: StatusRunning,
		Logs:   []string{"[system] started PTY: sh"},
	}})

	if !manager.AppendOutput("pty", "a") || !manager.AppendOutput("pty", "b") || !manager.AppendOutput("pty", "c") {
		t.Fatal("AppendOutput returned false for existing session")
	}
	if manager.AppendOutput("missing", "x") {
		t.Fatal("AppendOutput returned true for missing session")
	}
	pty, _ := manager.SessionByID("pty")
	if got := strings.Join(pty.Logs, "|"); got != "[system] started PTY: sh|abc" {
		t.Fatalf("logs = %q, want coalesced abc line", got)
	}

	manager.AppendOutput("pty", "\b\nnext\r\n")
	if got := strings.Join(pty.Logs, "|"); got != "[system] started PTY: sh|ab|next|" {
		t.Fatalf("logs after control chars = %q", got)
	}
}

func TestAppendOutputHandlesPromptRedraw(t *testing.T) {
	manager := NewManager([]Session{{
		ID:     "pty",
		Name:   "pty",
		Kind:   SessionKindPTY,
		Status: StatusRunning,
	}})

	manager.AppendOutput("pty", "PS> a")
	manager.AppendOutput("pty", "\r\x1b[2KPS> ab")
	manager.AppendOutput("pty", "\r\x1b[2KPS> abc")
	pty, _ := manager.SessionByID("pty")
	if got := strings.Join(pty.Logs, "|"); got != "PS> abc" {
		t.Fatalf("redrawn prompt logs = %q, want final prompt only", got)
	}

	manager.AppendOutput("pty", "\rPS> x")
	if got := strings.Join(pty.Logs, "|"); got != "PS> xbc" {
		t.Fatalf("carriage overwrite logs = %q, want overwritten prefix", got)
	}

	manager.AppendOutput("pty", "\x1b[K")
	if got := strings.Join(pty.Logs, "|"); got != "PS> x" {
		t.Fatalf("erase-to-end logs = %q, want truncated suffix", got)
	}
}

func TestAppendOutputHandlesPowerShellCursorPositioning(t *testing.T) {
	manager := NewManager([]Session{{
		ID:     "pty",
		Name:   "pty",
		Kind:   SessionKindPTY,
		Status: StatusRunning,
	}})

	manager.AppendOutput("pty", "\x1b[2J\x1b[H\r\n\r\n\r\n\x1b[H")
	manager.AppendOutput("pty", "\r\n")
	manager.AppendOutput("pty", "\x1b[?25l\r\n    Verzeichnis: C:\\Users\\barth\\Developer\\ottrta\x1b[6;1HMode                 LastWriteTime         Length Name\x1b[65X\r\n----                 -------------         ------ ----\x1b[65X\r\nd-----        05.06.2026     20:59                cmd\x1b[66X\r\n\x1b[?25h")

	pty, _ := manager.SessionByID("pty")
	got := strings.Join(pty.Logs, "\n")
	if strings.Contains(got, "ottrtaMode") {
		t.Fatalf("PowerShell output merged cursor-positioned rows:\n%s", got)
	}
	if !strings.Contains(got, "Verzeichnis: C:\\Users\\barth\\Developer\\ottrta") {
		t.Fatalf("PowerShell output missing directory header:\n%s", got)
	}
	if !strings.Contains(got, "Mode                 LastWriteTime") {
		t.Fatalf("PowerShell output missing table header:\n%s", got)
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

func TestDefaultShellCommandResolution(t *testing.T) {
	lookupPowerShell := func(name string) (string, error) {
		if name == "powershell.exe" {
			return name, nil
		}
		return "", context.Canceled
	}
	lookupMissing := func(name string) (string, error) {
		return "", context.Canceled
	}
	envShell := func(name string) string {
		if name == "SHELL" {
			return "/bin/zsh"
		}
		return ""
	}
	emptyEnv := func(string) string { return "" }

	if got := defaultShellCommand("windows", emptyEnv, lookupPowerShell); got != "powershell.exe" {
		t.Fatalf("windows shell = %q, want powershell.exe", got)
	}
	if got := defaultShellCommand("windows", emptyEnv, lookupMissing); got != "cmd.exe" {
		t.Fatalf("windows fallback shell = %q, want cmd.exe", got)
	}
	if got := defaultShellCommand("linux", envShell, lookupMissing); got != "/bin/zsh" {
		t.Fatalf("unix shell = %q, want /bin/zsh", got)
	}
	if got := defaultShellCommand("darwin", emptyEnv, lookupMissing); got != "sh" {
		t.Fatalf("unix fallback shell = %q, want sh", got)
	}
}

func TestPTYSessionErrorPaths(t *testing.T) {
	manager := NewFakeManager()

	if _, err := manager.StartPTYSession(context.Background(), "missing", 80, 24); err == nil {
		t.Fatal("StartPTYSession missing session error = nil")
	}
	if _, err := manager.StartPTYSession(context.Background(), "fake-claude-1", 80, 24); err == nil {
		t.Fatal("StartPTYSession fake session error = nil")
	}
	if err := manager.WritePTYSession("shell-1", []byte("x")); err == nil {
		t.Fatal("WritePTYSession stopped PTY error = nil")
	}
	if err := manager.ResizePTYSession("shell-1", 80, 24); err == nil {
		t.Fatal("ResizePTYSession stopped PTY error = nil")
	}
	if err := manager.StopPTYSession("fake-claude-1"); err == nil {
		t.Fatal("StopPTYSession fake session error = nil")
	}
	if err := manager.StopPTYSession("shell-1"); err == nil {
		t.Fatal("StopPTYSession stopped PTY error = nil")
	}
}

func TestStartPTYSessionStreamsAndStops(t *testing.T) {
	manager := NewManager([]Session{{
		ID:      "pty-go-version",
		Name:    "pty-go-version",
		Kind:    SessionKindPTY,
		Status:  StatusStopped,
		Command: "go",
		Args:    []string{"version"},
	}})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	events, err := manager.StartPTYSession(ctx, "pty-go-version", 80, 24)
	if err != nil {
		t.Fatalf("StartPTYSession returned error: %v", err)
	}
	s, _ := manager.SessionByID("pty-go-version")
	if s.Status != StatusRunning {
		t.Fatalf("status after start = %s, want running", s.Status)
	}

	var output strings.Builder
	for {
		select {
		case msg, ok := <-events:
			if !ok {
				t.Fatal("events channel closed before PTY exit message")
			}
			if len(msg.Data) > 0 {
				text := string(msg.Data)
				output.WriteString(text)
				manager.AppendLog(msg.SessionID, text)
			}
			if msg.Err != nil || len(msg.Data) == 0 {
				manager.AppendExitLog(msg.SessionID, msg.Err)
				manager.MarkStopped(msg.SessionID)
				if !strings.Contains(output.String(), "go version") {
					t.Fatalf("PTY output = %q, want go version", output.String())
				}
				if s.Status != StatusStopped {
					t.Fatalf("status after PTY exit = %s, want stopped", s.Status)
				}
				return
			}
		case <-ctx.Done():
			t.Fatalf("timed out waiting for PTY events: %v", s.Logs)
		}
	}
}

func TestStopPTYSessionKeepsManagerAlive(t *testing.T) {
	t.Setenv("OTTRTA_TEST_HELPER_PROCESS", "1")
	manager := NewManager([]Session{{
		ID:      "pty-helper",
		Name:    "pty-helper",
		Kind:    SessionKindPTY,
		Status:  StatusStopped,
		Command: os.Args[0],
		Args:    []string{"-test.run=TestPTYHelperProcess"},
	}})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	events, err := manager.StartPTYSession(ctx, "pty-helper", 80, 24)
	if err != nil {
		t.Fatalf("StartPTYSession returned error: %v", err)
	}
	if err := manager.StopPTYSession("pty-helper"); err != nil {
		t.Fatalf("StopPTYSession returned error: %v", err)
	}
	s, _ := manager.SessionByID("pty-helper")
	if s.Status != StatusStopped {
		t.Fatalf("status after stop = %s, want stopped", s.Status)
	}

	select {
	case <-events:
	case <-ctx.Done():
		t.Fatal("timed out waiting for stopped PTY event")
	}
}

func TestPTYHelperProcess(t *testing.T) {
	if os.Getenv("OTTRTA_TEST_HELPER_PROCESS") != "1" {
		return
	}
	time.Sleep(time.Minute)
}
