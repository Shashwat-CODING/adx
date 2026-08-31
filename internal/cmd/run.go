package cmd

import (
	"fmt"
	"os"

	"github.com/Shashwat-CODING/adx/internal/gradle"
	"github.com/Shashwat-CODING/adx/internal/ui"
	"github.com/spf13/cobra"
)

var (
	runOpenFlag    bool
	runNoBuildFlag bool
	runCleanFlag   bool

	runCmd = &cobra.Command{
		Use:   "run [variant]",
		Short: "Build, install, and run on connected Android device(s)",
		Long: `Builds the Android app, installs it to connected device(s), and optionally launches it.
If multiple devices are connected, an interactive menu allows targeting one or all devices.

Examples:
  adx run
  adx run debug --open
  adx run release
  adx run debug --no-build
  adx run debug -d emulator-5554`,
		RunE: func(cmd *cobra.Command, args []string) error {
			variant := "debug"
			if len(args) > 0 {
				variant = args[0]
			}

			// 1. Resolve Project & Package Info
			p, err := GetProject()
			if err != nil {
				return err
			}

			// 2. Resolve ADB & Devices early before building
			adbClient, err := GetADBClient()
			if err != nil {
				return err
			}

			targetDevices, err := ResolveTargetDevices(adbClient)
			if err != nil {
				return err
			}

			ui.Info("Targeting %d device(s)", len(targetDevices))

			// 3. Build APK unless --no-build specified
			runner := gradle.NewRunner(p, IsVerbose())
			if !runNoBuildFlag {
				if runCleanFlag {
					if err := runner.Clean(); err != nil {
						return err
					}
				}

				if err := runner.BuildVariant(variant); err != nil {
					return err
				}
			}

			// 4. Locate APK
			apkPath, err := p.FindApk(variant)
			if err != nil {
				return fmt.Errorf("could not find %s APK: %w", variant, err)
			}

			if fi, statErr := os.Stat(apkPath); statErr == nil {
				sizeMB := float64(fi.Size()) / (1024 * 1024)
				ui.Info("Found APK: %s (%.2f MB)", ui.ClickablePath(apkPath), sizeMB)
			}

			// 5. Install on target devices
			ui.Step("Installing APK on target device(s)...")
			if len(targetDevices) == 1 {
				dev := targetDevices[0]
				ui.Info("Installing on %s (%s)...", dev.Model, dev.Serial)
				if err := adbClient.InstallAPK(dev.Serial, apkPath); err != nil {
					return err
				}
				ui.Success("Installation succeeded on %s", dev.Model)
			} else {
				errs := adbClient.InstallParallel(targetDevices, apkPath)
				var failedCount int
				for i, e := range errs {
					if e != nil {
						ui.Error("Installation failed on %s: %v", targetDevices[i].Model, e)
						failedCount++
					}
				}
				if failedCount == len(targetDevices) {
					return fmt.Errorf("failed to install APK on all devices")
				}
			}

			// 6. Launch application if --open is set
			if runOpenFlag {
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

			ui.Success("Run command completed!")
			return nil
		},
	}
)

func init() {
	runCmd.Flags().BoolVar(&runOpenFlag, "open", true, "Open the application on device(s) after installation")
	runCmd.Flags().BoolVar(&runNoBuildFlag, "no-build", false, "Skip build step and install existing APK")
	runCmd.Flags().BoolVarP(&runCleanFlag, "clean", "c", false, "Clean project before building")
	rootCmd.AddCommand(runCmd)
}
