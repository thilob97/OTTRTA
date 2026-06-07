package cli

import (
	"bytes"
	"testing"
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
		{name: "unix default", goos: "linux", want: installScriptURL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := updateInstructions(tt.goos)
			if !bytes.Contains([]byte(got), []byte(tt.want)) {
				t.Fatalf("updateInstructions(%q) = %q, want %q", tt.goos, got, tt.want)
			}
			if bytes.Contains([]byte(got), []byte("Updating OTTRTA/RTA")) {
				t.Fatalf("updateInstructions(%q) retained old execution banner: %q", tt.goos, got)
			}
		})
	}
}
