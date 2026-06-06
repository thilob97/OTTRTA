package agent

import (
	"os/exec"
	"strings"
)

var AttentionHints = []string{
	"Do you want to proceed",
	"Allow",
	"approve",
	"Yes",
	"No",
}

func OmpExists() bool {
	_, err := exec.LookPath("omp")
	return err == nil
}

func CheckAttention(output string) bool {
	for _, hint := range AttentionHints {
		if strings.Contains(output, hint) {
			return true
		}
	}
	return false
}
