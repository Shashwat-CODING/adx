package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Shashwat-CODING/adx/internal/ui"
	"github.com/spf13/cobra"
)

var (
	logsClearFlag   bool
	logsPackageFlag string

	logsCmd = &cobra.Command{
		Use:   "logs",
		Short: "Stream logcat logs for the current Android application",
		Long: `Streams real-time filtered logcat logs for the project application.
Automatically filters logs by application package ID and active process ID.

Examples:
  adx logs
  adx logs --clear
  adx logs -p com.example.myapp
  adx logs -d emulator-5554`,
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
			if len(targetDevices) > 1 {
				ui.Info("Logging will stream from first selected device: %s (%s)", device.Model, device.Serial)
			}

			pkgName := logsPackageFlag
			if pkgName == "" {
				p, err := GetProject()
				if err == nil && p.PackageName != "" {
					pkgName = p.PackageName
				}
			}

			if pkgName == "" {
				return fmt.Errorf("could not detect package name. Specify manually using --package or -p")
			}

			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			return adbClient.StreamLogs(ctx, device.Serial, pkgName, logsClearFlag)
		},
	}
)

func init() {
	logsCmd.Flags().BoolVarP(&logsClearFlag, "clear", "c", false, "Clear previous logcat logs before streaming")
	logsCmd.Flags().StringVarP(&logsPackageFlag, "package", "p", "", "Explicit application package name to filter")
	rootCmd.AddCommand(logsCmd)
}
