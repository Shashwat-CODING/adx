package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Shashwat-CODING/adx/internal/ui"
	"github.com/spf13/cobra"
)

var recordCmd = &cobra.Command{
	Use:   "record [output.mp4]",
	Short: "Record the device screen to an MP4 video file",
	Long: `Starts recording the connected Android screen to an MP4 video file.
Press Ctrl+C to stop recording and pull the video file locally.

Examples:
  adx record
  adx record demo.mp4
  adx record -d emulator-5554`,
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

		destFile := fmt.Sprintf("recording-%s.mp4", time.Now().Format("20060102-150405"))
		if len(args) > 0 {
			destFile = args[0]
		}

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		ui.Step("Recording screen on %s...", device.Model)
		ui.Info("Recording in progress. Press Ctrl+C to finish and save video...")

		if err := adbClient.RecordScreen(ctx, device.Serial, destFile); err != nil {
			ui.Error("Recording failed: %v", err)
			return err
		}

		ui.Success("Screen recording saved: %s", ui.ClickablePath(destFile))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(recordCmd)
}
