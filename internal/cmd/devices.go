package cmd

import (
	"fmt"

	"github.com/Shashwat-CODING/adx/internal/ui"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var devicesCmd = &cobra.Command{
	Use:   "devices",
	Short: "List all connected Android devices and emulators",
	Long:  `Queries ADB and displays all attached physical devices and running emulators.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := GetADBClient()
		if err != nil {
			return err
		}

		devices, err := client.GetDevices()
		if err != nil {
			return err
		}

		if len(devices) == 0 {
			ui.Warn("No connected Android devices found.")
			ui.Dim("Make sure USB debugging is enabled on your device or start an Android emulator.")
			return nil
		}

		ui.Step("Connected Android Devices (%d):", len(devices))
		for i, d := range devices {
			stateColor := color.FgGreen
			if d.State != "device" {
				stateColor = color.FgYellow
			}
			fmt.Printf("  [%d] %-24s  %-20s  %s\n",
				i+1,
				color.New(color.Bold).Sprint(d.Model),
				ui.OrangeSoft().Sprint(d.Serial),
				color.New(stateColor).Sprint(d.State),
			)
		}
		fmt.Println()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(devicesCmd)
}
