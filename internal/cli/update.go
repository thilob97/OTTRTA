package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

func newUpdateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update OTTRTA/RTA to the latest version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Updating OTTRTA/RTA...")
			if runtime.GOOS == "windows" {
				psCmd := exec.Command("powershell", "-c", "irm https://raw.githubusercontent.com/thilob97/ottrta/main/install.ps1 | iex")
				psCmd.Stdout = os.Stdout
				psCmd.Stderr = os.Stderr
				return psCmd.Run()
			} else {
				shCmd := exec.Command("sh", "-c", "curl -fsSL https://raw.githubusercontent.com/thilob97/ottrta/main/install.sh | sh")
				shCmd.Stdout = os.Stdout
				shCmd.Stderr = os.Stderr
				return shCmd.Run()
			}
		},
	}
}
