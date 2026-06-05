package cli

import (
	"io"

	"github.com/handyfun97/ottrta/internal/session"
	"github.com/spf13/cobra"
)

type shellOptions struct {
	command string
	args    []string
	workDir string
}

func newShellCommand() *cobra.Command {
	opts := shellOptions{}
	cmd := &cobra.Command{
		Use:   "shell [--command SHELL] [--args ARG]...",
		Short: "Start an interactive PTY shell",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runForegroundShell(cmd, opts)
		},
	}

	cmd.Flags().StringVar(&opts.command, "command", "", "shell command to run")
	cmd.Flags().StringArrayVar(&opts.args, "args", nil, "shell argument; repeat for multiple arguments")
	cmd.Flags().StringVar(&opts.workDir, "workdir", "", "working directory")
	return cmd
}

func runForegroundShell(cmd *cobra.Command, opts shellOptions) error {
	command := opts.command
	if command == "" {
		command = session.DefaultShellCommand()
	}

	manager := session.NewManager([]session.Session{{
		ID:      "shell",
		Name:    "shell",
		Kind:    session.SessionKindPTY,
		Status:  session.StatusStopped,
		Command: command,
		Args:    opts.args,
		WorkDir: opts.workDir,
	}})

	events, err := manager.StartPTYSession(cmd.Context(), "shell", 120, 40)
	if err != nil {
		return err
	}
	stopOnReturn := true
	defer func() {
		if stopOnReturn {
			manager.StopAllProcesses()
		}
	}()

	go func() {
		_, _ = io.Copy(ptyWriter{manager: &manager, id: "shell"}, cmd.InOrStdin())
	}()

	for msg := range events {
		if len(msg.Data) > 0 {
			if err := writeShellOutput(cmd, msg.Data); err != nil {
				return err
			}
		}
		if msg.Err != nil {
			stopOnReturn = false
			manager.MarkStopped("shell")
			return msg.Err
		}
		if len(msg.Data) == 0 {
			stopOnReturn = false
			manager.MarkStopped("shell")
			return nil
		}
	}
	return nil
}

func writeShellOutput(cmd *cobra.Command, data []byte) error {
	_, err := cmd.OutOrStdout().Write(data)
	return err
}

type ptyWriter struct {
	manager *session.Manager
	id      string
}

func (w ptyWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if err := w.manager.WritePTYSession(w.id, p); err != nil {
		return 0, err
	}
	return len(p), nil
}
