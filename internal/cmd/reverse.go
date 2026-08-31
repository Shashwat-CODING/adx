package cmd

import (
	"fmt"
	"os/exec"

	"github.com/Shashwat-CODING/adx/internal/ui"
	"github.com/spf13/cobra"
)

var reverseCmd = &cobra.Command{
	Use:   "reverse [port]",
	Short: "Reverse port forward to access host localhost from Android",
	Long: `Runs 'adb reverse tcp:<port> tcp:<port>' so your Android app can reach your local development backend (e.g. localhost:8080 or localhost:3000).
Defaults to 8080 if not specified.

Example:
  adx reverse 8080
  adx reverse 3000`,
	RunE: func(cmd *cobra.Command, args []string) error {
		port := "8080"
		if len(args) > 0 {
			port = args[0]
		}

		adbClient, err := GetADBClient()
		if err != nil {
			return err
		}

		targetDevices, err := ResolveTargetDevices(adbClient)
		if err != nil {
			return err
		}

		proto := fmt.Sprintf("tcp:%s", port)
		for _, dev := range targetDevices {
			cmd := exec.Command(adbClient.AdbPath, "-s", dev.Serial, "reverse", proto, proto)
			if out, err := cmd.CombinedOutput(); err != nil {
				ui.Error("Failed to reverse port %s on %s: %s (%v)", port, dev.Model, string(out), err)
			} else {
				ui.Success("Reversed %s on %s -> app can reach http://localhost:%s", proto, dev.Model, port)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(reverseCmd)
}
