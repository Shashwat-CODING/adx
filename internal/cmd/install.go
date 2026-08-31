package cmd

import (
	"fmt"
	"os"

	"github.com/Shashwat-CODING/adx/internal/ui"
	"github.com/spf13/cobra"
)

var (
	installCustomApkFlag string
	installOpenFlag      bool

	installCmd = &cobra.Command{
		Use:   "install [variant]",
		Short: "Install APK onto target device(s) without rebuilding",
		Long: `Installs the built APK onto connected Android device(s).
Variant defaults to debug. Alternatively specify --apk to install a specific APK file.
Use --open to automatically launch the application after installation.

Examples:
  adx install
  adx install release --open
  adx install app --open
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

			p, err := GetProject()
			if err != nil {
				return err
			}

			var apkPath string
			if installCustomApkFlag != "" {
				apkPath = installCustomApkFlag
			} else {
				if len(args) == 0 || args[0] == "app" || args[0] == "apk" {
					var detectedVariant string
					apkPath, detectedVariant, err = p.FindBestAvailableApk()
					if err != nil {
						return err
					}
					ui.Info("Found %s APK: %s", detectedVariant, apkPath)
				} else {
					variant := args[0]
					apkPath, err = p.FindApk(variant)
					if err != nil {
						return err
					}
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

			if installOpenFlag {
				ui.Step("Launching application...")
				if p.PackageName == "" {
					ui.Warn("Package name could not be automatically detected. App installed but skipping auto-launch.")
					return nil
				}

				for _, dev := range targetDevices {
					ui.Info("Opening %s on %s...", p.PackageName, dev.Model)
					if err := adbClient.LaunchApp(dev.Serial, p.PackageName, p.LauncherActivity); err != nil {
						ui.Warn("Could not launch on %s: %v", dev.Model, err)
					} else {
						ui.Success("Application started on %s", dev.Model)
					}
				}
			}

			return nil
		},
	}
)

func init() {
	installCmd.Flags().StringVar(&installCustomApkFlag, "apk", "", "Direct path to an APK file")
	installCmd.Flags().BoolVar(&installOpenFlag, "open", false, "Open the application on device(s) after installation")
	rootCmd.AddCommand(installCmd)
}
