package session

import (
	"strings"
	"testing"
)

func TestRandomImpNamesHaveLargeInventory(t *testing.T) {
	if got := len(impNameAdjectives) * len(impNameNouns); got < 800 {
		t.Fatalf("imp name inventory = %d, want at least 800", got)
	}
	name := RandomImpName()
	if !strings.Contains(name, " ") {
		t.Fatalf("RandomImpName() = %q, want multi-word imp name", name)
	}
	if len(name) > 24 {
		t.Fatalf("RandomImpName() = %q, want short card-friendly name", name)
	}
}

func TestImpNamesSuggestAISlop(t *testing.T) {
	combined := strings.Join(append(impNameAdjectives[:], impNameNouns[:]...), " ")
	for _, want := range []string{"Slop", "Prompt", "Token", "Vibe", "Goblin"} {
		if !strings.Contains(combined, want) {
			t.Fatalf("imp name inventory does not contain slop theme %q", want)
		}
	}
}
