package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

func newUpdateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Show explicit update instructions for OTTRTA/RTA",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprint(cmd.OutOrStdout(), updateInstructions(runtime.GOOS))
			return err
		},
	}
}

const (
	installScriptURL        = "https://raw.githubusercontent.com/thilob97/ottrta/main/install.sh"
	installPowerShellURL    = "https://raw.githubusercontent.com/thilob97/ottrta/main/install.ps1"
	updateManualNotice      = "OTTRTA does not run remote installer scripts automatically.\nReview the installer first, then run the command explicitly if you trust it.\n\n"
	updateUnixCommand       = "curl -fsSL " + installScriptURL + " | sh"
	updatePowerShellCommand = "irm " + installPowerShellURL + " | iex"
)

func updateInstructions(goos string) string {
	if goos == "windows" {
		return updateManualNotice +
			"Installer URL: " + installPowerShellURL + "\n" +
			"Command:\n  " + updatePowerShellCommand + "\n"
	}

	return updateManualNotice +
		"Installer URL: " + installScriptURL + "\n" +
		"Command:\n  " + updateUnixCommand + "\n"
}
