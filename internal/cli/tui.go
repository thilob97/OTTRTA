package cli

import (
	"github.com/spf13/cobra"
	"github.com/thilob97/ottrta/internal/tui"
)

func newTUICommand() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Start the OTTRTA TUI",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI()
		},
	}
}

func runTUI() error {
	return tui.Run(Version)
}
