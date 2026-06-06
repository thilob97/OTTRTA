package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

const Version = "v0.4.8"

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the OTTRTA version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "ottrta %s\n", Version)
		},
	}
}
