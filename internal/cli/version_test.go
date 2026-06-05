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
	if got, want := out.String(), "ottrta v0.1.0\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestRootCommandIncludesRequiredCommands(t *testing.T) {
	cmd := NewRootCommand()
	for _, name := range []string{"tui", "version"} {
		if child, _, err := cmd.Find([]string{name}); err != nil || child == nil || child.Name() != name {
			t.Fatalf("command %q not found: child=%v err=%v", name, child, err)
		}
	}
}
