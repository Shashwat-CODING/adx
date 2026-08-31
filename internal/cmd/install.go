package cmd

import (
	"fmt"
	"os"

	"github.com/Shashwat-CODING/adx/internal/ui"
	"github.com/spf13/cobra"
)

var (
	installCustomApkFlag string

	installCmd = &cobra.Command{
		Use:   "install [variant]",
		Short: "Install APK onto target device(s) without rebuilding",
		Long: `Installs the built APK onto connected Android device(s).
Variant defaults to debug. Alternatively specify --apk to install a specific APK file.

Examples:
  adx install
  adx install release
  adx install --apk /path/to/my-app.apk`,
		RunE: func(cmd *cobra.Command, args []string) error {
			adbClient, err := GetADBClient()
			if err != nil {
				return err
			}

			targetDevices, err := ResolveTargetDevices(adbClient)
			if err != nil {
				return err
			}

			var apkPath string
			if installCustomApkFlag != "" {
				apkPath = installCustomApkFlag
			} else {
				variant := "debug"
				if len(args) > 0 {
					variant = args[0]
				}

				p, err := GetProject()
				if err != nil {
					return err
				}

				apkPath, err = p.FindApk(variant)
				if err != nil {
					return err
				}
			}

			if fi, err := os.Stat(apkPath); err != nil {
				return fmt.Errorf("APK file not found: %s", apkPath)
			} else {
				sizeMB := float64(fi.Size()) / (1024 * 1024)
				ui.Info("Installing %s (%.2f MB)", apkPath, sizeMB)
			}

			if len(targetDevices) == 1 {
				dev := targetDevices[0]
				ui.Info("Installing on %s...", dev.Model)
				if err := adbClient.InstallAPK(dev.Serial, apkPath); err != nil {
					return err
				}
				ui.Success("Installation succeeded on %s", dev.Model)
			} else {
				errs := adbClient.InstallParallel(targetDevices, apkPath)
				for i, e := range errs {
					if e != nil {
						ui.Error("Installation failed on %s: %v", targetDevices[i].Model, e)
					}
				}
			}

			return nil
		},
	}
)

func init() {
	installCmd.Flags().StringVar(&installCustomApkFlag, "apk", "", "Direct path to an APK file")
	rootCmd.AddCommand(installCmd)
}
