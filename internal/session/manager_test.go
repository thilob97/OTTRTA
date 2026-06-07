package session

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/thilob97/ottrta/internal/event"
)

func TestAppendLogTargetsSessionAndBoundsLogs(t *testing.T) {
	manager := NewManager([]Session{
		{ID: "one", Name: "one", Kind: SessionKindProcess, Status: StatusRunning},
		{ID: "two", Name: "two", Kind: SessionKindProcess, Status: StatusRunning},
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

func TestRenameSessionTrimsAndKeepsIdentity(t *testing.T) {
	manager := NewManager([]Session{{
		ID:      "one",
		Name:    "old",
		Kind:    SessionKindAgent,
		Status:  StatusRunning,
		Command: "omp",
		Logs:    []string{"existing log"},
		WorkDir: "work",
		Args:    []string{"--flag"},
	}})

	if !manager.RenameSession("one", "  New Name  ") {
		t.Fatal("RenameSession returned false for existing non-empty name")
	}
	s, _ := manager.SessionByID("one")
	if s.ID != "one" {
		t.Fatalf("ID = %q, want one", s.ID)
	}
	if s.Name != "New Name" {
		t.Fatalf("Name = %q, want trimmed New Name", s.Name)
	}
	if s.Status != StatusRunning || len(s.Logs) != 1 {
		t.Fatalf("rename mutated runtime fields: %+v", *s)
	}

	if manager.RenameSession("one", " \t ") {
		t.Fatal("RenameSession returned true for empty trimmed name")
	}
	if s.Name != "New Name" {
		t.Fatalf("Name after empty rename = %q, want unchanged", s.Name)
	}
	if manager.RenameSession("missing", "name") {
		t.Fatal("RenameSession returned true for missing session")
	}
}

func TestSessionStoreRoundTripsDefinitionsOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.json")
	want := []Session{{
		ID:             "agent-1",
		Name:           "Agent One",
		Kind:           SessionKindAgent,
		Status:         StatusRunning,
		Command:        "omp",
		Args:           []string{"--model", "default"},
		WorkDir:        "work",
		AgentKind:      AgentKindOmp,
		NeedsAttention: true,
		Logs:           []string{"runtime log"},
		Cells:          [][]Cell{{{Char: 'x', SGR: "31"}}},
		CurrentSGR:     "31",
	}}

	if err := SaveSessions(path, want); err != nil {
		t.Fatalf("SaveSessions returned error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	raw := string(data)
	for _, forbidden := range []string{"runtime log", "needsAttention", "\"status\"", "\"cells\"", "\"currentSGR\""} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("stored JSON contains runtime-only field %q:\n%s", forbidden, raw)
		}
	}

	got, err := LoadSessions(path)
	if err != nil {
		t.Fatalf("LoadSessions returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("loaded %d sessions, want 1", len(got))
	}
	s := got[0]
	if s.ID != "agent-1" || s.Name != "Agent One" || s.Kind != SessionKindAgent || s.Command != "omp" ||
		s.WorkDir != "work" || s.AgentKind != AgentKindOmp {
		t.Fatalf("loaded definition mismatch: %+v", s)
	}
	if strings.Join(s.Args, ",") != "--model,default" {
		t.Fatalf("Args = %v, want copied args", s.Args)
	}
	if s.Status != StatusStopped {
		t.Fatalf("Status = %s, want stopped", s.Status)
	}
	if s.NeedsAttention || len(s.Logs) != 0 || len(s.Cells) != 0 || s.CurrentSGR != "" {
		t.Fatalf("loaded runtime fields were not reset: %+v", s)
	}
}

func TestLoadSessionsRejectsMalformedDefinitions(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{name: "empty id", json: `[{"id":"","name":"Agent","kind":"agent","command":"omp","agentKind":"omp"}]`},
		{name: "empty command", json: `[{"id":"agent","name":"Agent","kind":"agent","command":"","agentKind":"omp"}]`},
		{name: "unsupported kind", json: `[{"id":"x","name":"X","kind":"unknown","command":"go"}]`},
		{name: "missing agent kind", json: `[{"id":"agent","name":"Agent","kind":"agent","command":"omp"}]`},
		{name: "agent kind on process", json: `[{"id":"proc","name":"Proc","kind":"process","command":"go","agentKind":"omp"}]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "sessions.json")
			if err := os.WriteFile(path, []byte(tt.json), 0o644); err != nil {
				t.Fatalf("WriteFile returned error: %v", err)
			}
			if got, err := LoadSessions(path); err == nil {
				t.Fatalf("LoadSessions returned (%+v, nil), want error", got)
			}
		})
	}
}

func TestSaveSessionsRejectsInvalidDefinitionsBeforeReplacingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.json")
	original := []byte("[]\n")
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	err := SaveSessions(path, []Session{{
		ID:      "bad",
		Name:    "Bad",
		Kind:    SessionKindProcess,
		Command: "",
	}})
	if err == nil {
		t.Fatal("SaveSessions returned nil error for invalid session")
	}

	got, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("ReadFile returned error: %v", readErr)
	}
	if string(got) != string(original) {
		t.Fatalf("file content changed after failed save: %q", got)
	}
}

func TestManagerCopiesCellsOnClonePaths(t *testing.T) {
	original := Session{
		ID:      "pty",
		Name:    "pty",
		Kind:    SessionKindPTY,
		Command: "sh",
		Args:    []string{"-l"},
		Logs:    []string{"log"},
		Cells:   [][]Cell{{{Char: 'a', SGR: "31"}}},
	}
	manager := NewManager([]Session{original})

	original.Args[0] = "mutated"
	original.Logs[0] = "mutated"
	original.Cells[0][0].Char = 'z'
	stored, _ := manager.SessionByID("pty")
	if stored.Args[0] != "-l" || stored.Logs[0] != "log" || stored.Cells[0][0].Char != 'a' {
		t.Fatalf("NewManager retained caller-owned slices: %+v", *stored)
	}

	snapshots := manager.Sessions()
	snapshots[0].Args[0] = "snapshot"
	snapshots[0].Logs[0] = "snapshot"
	snapshots[0].Cells[0][0].Char = 'x'
	stored, _ = manager.SessionByID("pty")
	if stored.Args[0] != "-l" || stored.Logs[0] != "log" || stored.Cells[0][0].Char != 'a' {
		t.Fatalf("Sessions exposed mutable slices: %+v", *stored)
	}

	added := Session{
		ID:      "added",
		Name:    "added",
		Kind:    SessionKindPTY,
		Command: "sh",
		Cells:   [][]Cell{{{Char: 'b'}}},
	}
	manager.AddSession(added)
	added.Cells[0][0].Char = 'y'
	stored, _ = manager.SessionByID("added")
	if stored.Cells[0][0].Char != 'b' {
		t.Fatalf("AddSession retained caller-owned cells: %+v", *stored)
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

func TestAppendOutputCursorHomeDoesNotClearLogs(t *testing.T) {
	manager := NewManager([]Session{{
		ID:     "pty",
		Name:   "pty",
		Kind:   SessionKindPTY,
		Status: StatusRunning,
	}})

	manager.AppendOutput("pty", "first\r\nsecond")
	manager.AppendOutput("pty", "\x1b[Htop")

	pty, _ := manager.SessionByID("pty")
	got := strings.Join(pty.Logs, "|")
	if !strings.Contains(got, "topst") {
		t.Fatalf("cursor-home output = %q, want first row overwritten", got)
	}
	if !strings.Contains(got, "second") {
		t.Fatalf("cursor-home output = %q, want existing later rows preserved", got)
	}
}

func TestAppendOutputBackspaceAtColumnZeroDoesNotDelete(t *testing.T) {
	manager := NewManager([]Session{{
		ID:     "pty",
		Name:   "pty",
		Kind:   SessionKindPTY,
		Status: StatusRunning,
	}})

	manager.AppendOutput("pty", "abc\b")
	pty, _ := manager.SessionByID("pty")
	if got := strings.Join(pty.Logs, "|"); got != "ab" {
		t.Fatalf("normal backspace logs = %q, want ab", got)
	}

	manager = NewManager([]Session{{
		ID:     "pty",
		Name:   "pty",
		Kind:   SessionKindPTY,
		Status: StatusRunning,
	}})
	manager.AppendOutput("pty", "abc\r\b")
	pty, _ = manager.SessionByID("pty")
	if got := strings.Join(pty.Logs, "|"); got != "abc" {
		t.Fatalf("column-zero backspace logs = %q, want abc", got)
	}
}

func TestCommandLineQuotesDisplayArgs(t *testing.T) {
	got := commandLine("cmd path", []string{"plain", "two words", "semi;colon", ""})
	want := `"cmd path" plain "two words" "semi;colon" ""`
	if got != want {
		t.Fatalf("commandLine = %q, want %q", got, want)
	}
}

func TestStartGoVersionProcessStreamsAndStops(t *testing.T) {
	manager := NewManager([]Session{
		{ID: "real-go-version", Name: "real-go-version", Kind: SessionKindProcess, Status: StatusStopped, Command: "go", Args: []string{"version"}},
	})
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

func TestProcessStreamsLongOutputLine(t *testing.T) {
	t.Setenv("OTTRTA_TEST_LONG_LINE_PROCESS", "1")
	wantLen := 128 * 1024
	manager := NewManager([]Session{{
		ID:      "long-line",
		Name:    "long-line",
		Kind:    SessionKindProcess,
		Status:  StatusStopped,
		Command: os.Args[0],
		Args:    []string{"-test.run=TestLongLineHelperProcess", "-test.v=false"},
	}})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	events, err := manager.StartSession(ctx, "long-line")
	if err != nil {
		t.Fatalf("StartSession returned error: %v", err)
	}

	var sawLongLine bool
	for {
		select {
		case msg, ok := <-events:
			if !ok {
				t.Fatal("events channel closed before exit message")
			}
			switch msg := msg.(type) {
			case event.SessionLogMsg:
				if strings.Contains(msg.Line, "log stream error") {
					t.Fatalf("scanner reported error for long line: %q", msg.Line)
				}
				if len(msg.Line) == wantLen {
					sawLongLine = true
				}
			case event.SessionExitedMsg:
				if msg.Err != nil {
					t.Fatalf("helper process exited with error: %v", msg.Err)
				}
				if !sawLongLine {
					t.Fatalf("long line of length %d was not streamed", wantLen)
				}
				return
			}
		case <-ctx.Done():
			t.Fatal("timed out waiting for long-line process")
		}
	}
}

func TestRemoveSessionStopsOnlyMatchingRuntimeKind(t *testing.T) {
	stopErr := errors.New("stop failed")
	manager := NewManager([]Session{{
		ID:      "pty",
		Name:    "pty",
		Kind:    SessionKindPTY,
		Status:  StatusRunning,
		Command: "sh",
	}})
	manager.ptys["pty"] = fakePTYSession{stopErr: stopErr}

	if err := manager.RemoveSession(0); !errors.Is(err, stopErr) {
		t.Fatalf("RemoveSession error = %v, want %v", err, stopErr)
	}
	if manager.Count() != 1 {
		t.Fatalf("Count after failed remove = %d, want 1", manager.Count())
	}

	manager.ptys["pty"] = fakePTYSession{}
	if err := manager.RemoveSession(0); err != nil {
		t.Fatalf("RemoveSession returned error: %v", err)
	}
	if manager.Count() != 0 {
		t.Fatalf("Count after successful remove = %d, want 0", manager.Count())
	}
}

func TestStopSessionErrorPaths(t *testing.T) {
	manager := NewManager([]Session{
		{ID: "missing", Name: "missing", Kind: SessionKindProcess, Status: StatusStopped, Command: "missing"},
	})
	if err := manager.StopSession("missing"); err == nil {
		t.Fatal("StopSession missing session error = nil")
	}
	if err := manager.StopSession("nonexistent"); err == nil {
		t.Fatal("StopSession nonexistent session error = nil")
	}
	if err := manager.StopSession("missing"); err == nil {
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
	manager := NewManager([]Session{
		{ID: "missing", Name: "missing", Kind: SessionKindPTY, Status: StatusStopped, Command: "missing"},
	})
	if _, err := manager.StartPTYSession(context.Background(), "missing", 80, 24); err == nil {
		t.Fatal("StartPTYSession missing session error = nil")
	}
	if _, err := manager.StartPTYSession(context.Background(), "nonexistent", 80, 24); err == nil {
		t.Fatal("StartPTYSession nonexistent session error = nil")
	}
	if err := manager.WritePTYSession("missing", []byte("x")); err == nil {
		t.Fatal("WritePTYSession not-running PTY error = nil")
	}
	if err := manager.ResizePTYSession("missing", 80, 24); err == nil {
		t.Fatal("ResizePTYSession not-running PTY error = nil")
	}
	if err := manager.StopPTYSession("nonexistent"); err == nil {
		t.Fatal("StopPTYSession nonexistent session error = nil")
	}
	if err := manager.StopPTYSession("missing"); err == nil {
		t.Fatal("StopPTYSession not-running PTY error = nil")
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

func TestLongLineHelperProcess(t *testing.T) {
	if os.Getenv("OTTRTA_TEST_LONG_LINE_PROCESS") != "1" {
		return
	}
	fmt.Println(strings.Repeat("x", 128*1024))
	os.Exit(0)
}
func TestPTYHelperProcess(t *testing.T) {
	if os.Getenv("OTTRTA_TEST_HELPER_PROCESS") != "1" {
		return
	}
	time.Sleep(time.Minute)
}

type fakePTYSession struct {
	stopErr error
}

func (f fakePTYSession) Start(context.Context, PTYSpec) error { return nil }
func (f fakePTYSession) Write([]byte) error                   { return nil }
func (f fakePTYSession) Resize(int, int) error                { return nil }
func (f fakePTYSession) Stop() error                          { return f.stopErr }

func (f fakePTYSession) Events() <-chan PTYEvent {
	ch := make(chan PTYEvent)
	close(ch)
	return ch
}
