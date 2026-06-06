package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const maxCompletionNames = 5

func completeDirectoryPath(input string) (string, string) {
	dir, prefix := filepath.Split(input)
	searchDir := dir
	if searchDir == "" {
		searchDir = "."
	}

	entries, err := os.ReadDir(searchDir)
	if err != nil {
		return input, "no readable directory match"
	}

	prefixKey := strings.ToLower(prefix)
	var matches []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(strings.ToLower(name), prefixKey) {
			matches = append(matches, name)
		}
	}
	if len(matches) == 0 {
		return input, "no directory match"
	}
	sort.Strings(matches)

	sep := pathSeparator(input)
	if len(matches) == 1 {
		return dir + matches[0] + sep, "completed " + matches[0]
	}

	common := commonPrefix(matches)
	completed := input
	if len(common) > len(prefix) {
		completed = dir + common
	}
	return completed, "matches: " + completionNames(matches)
}

func pathSeparator(input string) string {
	if strings.Contains(input, "/") {
		return "/"
	}
	if strings.Contains(input, "\\") {
		return "\\"
	}
	return string(os.PathSeparator)
}

func commonPrefix(values []string) string {
	if len(values) == 0 {
		return ""
	}
	prefix := []rune(values[0])
	for _, value := range values[1:] {
		for !strings.HasPrefix(value, string(prefix)) {
			if len(prefix) == 0 {
				return ""
			}
			prefix = prefix[:len(prefix)-1]
		}
	}
	return string(prefix)
}

func completionNames(matches []string) string {
	shown := matches
	if len(shown) > maxCompletionNames {
		shown = shown[:maxCompletionNames]
	}
	text := strings.Join(shown, ", ")
	if len(matches) > len(shown) {
		text += ", ..."
	}
	return text
}
