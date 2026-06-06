package cli

import (
	"github.com/thilob97/ottrta/internal/tui"
	"github.com/spf13/cobra"
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
	return tui.Run()
}
