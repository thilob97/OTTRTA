package cli

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
)

type runOptions struct {
	name    string
	command string
	args    []string
	workDir string
}

func newRunCommand() *cobra.Command {
	opts := runOptions{}
	cmd := &cobra.Command{
		Use:   "run --name NAME --command COMMAND [--args ARG]...",
		Short: "Run a foreground process session",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runForegroundProcess(cmd, opts)
		},
	}

	cmd.Flags().StringVar(&opts.name, "name", "", "session name")
	cmd.Flags().StringVar(&opts.command, "command", "", "command to run")
	cmd.Flags().StringArrayVar(&opts.args, "args", nil, "command argument; repeat for multiple arguments")
	cmd.Flags().StringVar(&opts.workDir, "workdir", "", "working directory")
	return cmd
}

func runForegroundProcess(cmd *cobra.Command, opts runOptions) error {
	if opts.name == "" {
		return fmt.Errorf("--name is required")
	}
	if opts.command == "" {
		return fmt.Errorf("--command is required")
	}

	process := exec.CommandContext(cmd.Context(), opts.command, opts.args...)
	if opts.workDir != "" {
		process.Dir = opts.workDir
	}
	process.Stdin = cmd.InOrStdin()
	process.Stdout = cmd.OutOrStdout()
	process.Stderr = cmd.ErrOrStderr()
	return process.Run()
}
