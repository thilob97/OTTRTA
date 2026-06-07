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
	for _, name := range []string{"tui", "version", "run", "shell"} {
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
