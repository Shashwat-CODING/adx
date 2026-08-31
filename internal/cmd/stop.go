package cmd

import (
	"fmt"

	"github.com/Shashwat-CODING/adx/internal/ui"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Force-stop the application process on target device(s)",
	Long:  `Sends am force-stop for the detected project application package.`,
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
			if err := adbClient.ForceStop(dev.Serial, p.PackageName); err != nil {
				ui.Error("Failed to stop on %s: %v", dev.Model, err)
			} else {
				ui.Success("Stopped %s on %s", p.PackageName, dev.Model)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
