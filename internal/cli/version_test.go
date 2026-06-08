package cli

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/thilob97/ottrta/internal/session"
)

func TestVersionCommand(t *testing.T) {
	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if got, want := out.String(), "ottrta "+Version+"\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestRootCommandIncludesRequiredCommands(t *testing.T) {
	cmd := NewRootCommand()
	for _, name := range []string{"tui", "version", "run", "shell", "agent", "update"} {
		if child, _, err := cmd.Find([]string{name}); err != nil || child == nil || child.Name() != name {
			t.Fatalf("command %q not found: child=%v err=%v", name, child, err)
		}
	}
	if child, _, err := cmd.Find([]string{"task"}); err == nil && child != nil && child.Name() == "task" {
		t.Fatalf("unexpected task command found: %+v", child)
	}
}

func TestRunCommandStreamsGoVersion(t *testing.T) {
	cmd := NewRootCommand()
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs([]string{"run", "--name", "test", "--command", "go", "--args", "version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v; stderr=%q", err, errOut.String())
	}
	if got := out.String(); !bytes.Contains([]byte(got), []byte("go version")) {
		t.Fatalf("output = %q, want go version", got)
	}
}

func TestRunCommandRequiresNameAndCommand(t *testing.T) {
	cmd := NewRootCommand()
	cmd.SetArgs([]string{"run", "--name", "test"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("Execute returned nil error without --command")
	}
}

func TestUpdateCommandPrintsManualInstructions(t *testing.T) {
	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"update"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	got := out.String()
	for _, want := range []string{
		"does not run remote installer scripts automatically",
		"Review the installer first",
		"Installer URL:",
		"Command:",
	} {
		if !bytes.Contains([]byte(got), []byte(want)) {
			t.Fatalf("output = %q, want substring %q", got, want)
		}
	}
}

func TestUpdateInstructionsSelectInstallerByPlatform(t *testing.T) {
	tests := []struct {
		name string
		goos string
		want string
	}{
		{name: "windows", goos: "windows", want: installPowerShellURL},
		{name: "unix default", goos: "linux", want: "curl -fsSL " + installScriptURL + " | sh"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := updateInstructions(tt.goos)
			if !bytes.Contains([]byte(got), []byte(tt.want)) {
				t.Fatalf("updateInstructions(%q) = %q, want %q", tt.goos, got, tt.want)
			}
			if tt.goos == "windows" && bytes.Contains([]byte(got), []byte("powershell -NoProfile")) {
				t.Fatalf("updateInstructions(%q) should not wrap command with powershell.exe: %q", tt.goos, got)
			}
			if bytes.Contains([]byte(got), []byte("Updating OTTRTA/RTA")) {
				t.Fatalf("updateInstructions(%q) retained old execution banner: %q", tt.goos, got)
			}
		})
	}
}

func TestAgentEventLoopExitsAfterAllSessionsComplete(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	first := make(chan session.PTYEvent, 2)
	first <- session.PTYEvent{SessionID: "omp-1", Data: []byte("first output\n")}
	first <- session.PTYEvent{SessionID: "omp-1"}
	close(first)

	second := make(chan session.PTYEvent, 1)
	second <- session.PTYEvent{SessionID: "omp-2"}
	close(second)

	sessions := []session.Session{
		{ID: "omp-1"},
		{ID: "omp-2"},
	}

	var out bytes.Buffer
	err := runAgentEventLoop(
		ctx,
		cancel,
		&out,
		sessions,
		[]<-chan session.PTYEvent{first, second},
		make(chan os.Signal),
		func(string) error { return nil },
	)
	if err != nil {
		t.Fatalf("runAgentEventLoop returned error: %v", err)
	}

	got := out.String()
	for _, want := range []string{"first output", "[omp-1 exited]", "[omp-2 exited]"} {
		if !strings.Contains(got, want) {
			t.Fatalf("output = %q, want substring %q", got, want)
		}
	}
}
