package cli

import "github.com/spf13/cobra"

func Execute() error {
	return NewRootCommand().Execute()
}

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "ottrta",
		Short:        "One Terminal To Rule Them All",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI()
		},
	}

	cmd.AddCommand(newTUICommand())
	cmd.AddCommand(newVersionCommand())
	cmd.AddCommand(newRunCommand())
	cmd.AddCommand(newShellCommand())
	cmd.AddCommand(newAgentCommand())
	cmd.AddCommand(newTaskCommand())

	return cmd
}
