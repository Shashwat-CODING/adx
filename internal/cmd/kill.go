package cmd

import (
	"os/exec"

	"github.com/Shashwat-CODING/adx/internal/ui"
	"github.com/spf13/cobra"
)

var killCmd = &cobra.Command{
	Use:   "kill",
	Short: "Kill stuck Gradle/Kotlin daemons and restart ADB server",
	Long: `Kills rogue Gradle and Kotlin daemons consuming RAM/CPU and restarts the ADB server.
Useful when ADB hangs or Gradle locks files.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := GetProject()
		if err == nil {
			ui.Info("Stopping Gradle daemons...")
			_ = exec.Command(p.GradlewPath, "--stop").Run()
		}

		adbClient, err := GetADBClient()
		if err == nil {
			ui.Info("Restarting ADB server...")
			_ = exec.Command(adbClient.AdbPath, "kill-server").Run()
			_ = exec.Command(adbClient.AdbPath, "start-server").Run()
			ui.Success("ADB server restarted")
		}

		ui.Success("Daemons stopped and toolchain reset.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(killCmd)
}
