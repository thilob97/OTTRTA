package session

import (
	"os"
	"os/exec"
	"runtime"
)

// DefaultShellCommand returns the best interactive shell command for the host.
func DefaultShellCommand() string {
	return defaultShellCommand(runtime.GOOS, os.Getenv, exec.LookPath)
}

func defaultShellCommand(goos string, getenv func(string) string, lookPath func(string) (string, error)) string {
	if goos == "windows" {
		for _, name := range []string{"pwsh", "powershell.exe", "cmd.exe"} {
			if _, err := lookPath(name); err == nil {
				return name
			}
		}
		return "cmd.exe"
	}

	if shell := getenv("SHELL"); shell != "" {
		return shell
	}
	return "sh"
}
