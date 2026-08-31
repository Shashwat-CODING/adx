package cmd

import (
	"fmt"
	"time"

	"github.com/Shashwat-CODING/adx/internal/ui"
	"github.com/spf13/cobra"
)

var screenshotCmd = &cobra.Command{
	Use:   "screenshot [output.png]",
	Short: "Take a screenshot of the connected device display",
	Long: `Captures a high-resolution PNG screenshot of the connected Android device or emulator.
If no filename is specified, saves to screenshot-<timestamp>.png in the current directory.

Examples:
  adx screenshot
  adx screenshot bug-report.png
  adx screenshot -d emulator-5554`,
	RunE: func(cmd *cobra.Command, args []string) error {
		adbClient, err := GetADBClient()
		if err != nil {
			return err
		}

		targetDevices, err := ResolveTargetDevices(adbClient)
		if err != nil {
			return err
		}

		device := targetDevices[0]

		destFile := fmt.Sprintf("screenshot-%s.png", time.Now().Format("20060102-150405"))
		if len(args) > 0 {
			destFile = args[0]
		}

		spinner := ui.NewSpinner(fmt.Sprintf("Capturing screenshot from %s...", device.Model))
		if err := adbClient.TakeScreenshot(device.Serial, destFile); err != nil {
			spinner.StopFail("Screenshot failed: %v", err)
			return err
		}

		spinner.StopSuccess("Screenshot saved: %s", ui.ClickablePath(destFile))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(screenshotCmd)
}
