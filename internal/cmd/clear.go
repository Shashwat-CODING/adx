package cmd

import (
	"fmt"
	"os/exec"

	"github.com/Shashwat-CODING/adx/internal/ui"
	"github.com/spf13/cobra"
)

var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear app data and cache on connected device(s)",
	Long: `Executes 'adb shell pm clear <package>' on target device(s).
Resets Room DB, SharedPreferences, Datastore, and local cache.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := GetProject()
		if err != nil {
			return err
		}

		if p.PackageName == "" {
			return fmt.Errorf("could not detect application package name")
		}

		adbClient, err := GetADBClient()
		if err != nil {
			return err
		}

		targetDevices, err := ResolveTargetDevices(adbClient)
		if err != nil {
			return err
		}

		for _, dev := range targetDevices {
			ui.Info("Clearing data for %s on %s...", p.PackageName, dev.Model)
			cmd := exec.Command(adbClient.AdbPath, "-s", dev.Serial, "shell", "pm", "clear", p.PackageName)
			out, err := cmd.CombinedOutput()
			if err != nil {
				ui.Error("Failed to clear data on %s: %v", dev.Model, err)
			} else {
				ui.Success("App data cleared on %s (%s)", dev.Model, string(out))
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(clearCmd)
}
