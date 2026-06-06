package session

import "crypto/rand"

var impNameAdjectives = [...]string{
	"Slop",
	"Token",
	"Prompt",
	"Vibe",
	"RAG",
	"Cache",
	"Diff",
	"Patch",
	"Merge",
	"Eval",
	"JSON",
	"YAML",
	"Regex",
	"Vector",
	"Context",
	"Model",
	"Retry",
	"Tool",
	"Embed",
	"Halluc",
	"Boiler",
	"Auto",
	"Fuzzy",
	"Overfit",
	"Stale",
	"Flaky",
	"LoRA",
	"ZeroShot",
	"Buggy",
	"Noisy",
}

var impNameNouns = [...]string{
	"Goblin",
	"Gremlin",
	"Burper",
	"Muncher",
	"Gobbler",
	"Pixie",
	"Imp",
	"Troll",
	"Kobold",
	"Rat",
	"Sprite",
	"Ferret",
	"Mole",
	"Mite",
	"Warlock",
	"Phantom",
	"Dustling",
	"Licker",
	"Dodger",
	"Fuzzer",
	"Paster",
	"Drifter",
	"Squinter",
	"Spammer",
	"Looper",
	"Leaker",
	"Botling",
	"Boggler",
	"Nibbler",
	"Slurper",
}

func RandomImpName() string {
	return impNameAdjectives[randomIndex(len(impNameAdjectives))] + " " + impNameNouns[randomIndex(len(impNameNouns))]
}

func randomIndex(max int) int {
	if max <= 1 {
		return 0
	}
	var b [2]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0
	}
	return ((int(b[0]) << 8) | int(b[1])) % max
}
