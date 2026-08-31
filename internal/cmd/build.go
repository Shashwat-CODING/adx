package cmd

import (
	"os"

	"github.com/Shashwat-CODING/adx/internal/gradle"
	"github.com/Shashwat-CODING/adx/internal/ui"
	"github.com/spf13/cobra"
)

var (
	buildCleanFlag      bool
	buildOfflineFlag    bool
	buildStacktraceFlag bool

	buildCmd = &cobra.Command{
		Use:   "build [variant]",
		Short: "Build Android APK (debug or release)",
		Long: `Builds the Android project for the given build variant (defaults to debug).
Examples:
  adx build
  adx build debug
  adx build release
  adx build stagingDebug --clean`,
		RunE: func(cmd *cobra.Command, args []string) error {
			variant := "debug"
			if len(args) > 0 {
				variant = args[0]
			}

			if variant == "aab" || variant == "abb" || variant == "bundle" {
				return bundleCmd.RunE(cmd, []string{"release"})
			}

			p, err := GetProject()
			if err != nil {
				return err
			}

			ui.Info("Project root: %s", p.RootDir)
			if p.PackageName != "" {
				ui.Info("Package name: %s", p.PackageName)
			}

			runner := gradle.NewRunner(p, IsVerbose())

			var extraArgs []string
			if buildOfflineFlag {
				extraArgs = append(extraArgs, "--offline")
			}
			if buildStacktraceFlag {
				extraArgs = append(extraArgs, "--stacktrace")
			}

			if buildCleanFlag {
				if err := runner.Clean(); err != nil {
					return err
				}
			}

			if err := runner.BuildVariant(variant, extraArgs...); err != nil {
				return err
			}

			apkPath, err := p.FindApk(variant)
			if err == nil {
				clickable := ui.ClickablePath(apkPath)
				if fi, statErr := os.Stat(apkPath); statErr == nil {
					sizeMB := float64(fi.Size()) / (1024 * 1024)
					ui.Success("APK built successfully: %s (%.2f MB)", clickable, sizeMB)
				} else {
					ui.Success("APK built successfully: %s", clickable)
				}
			} else {
				ui.Success("Build completed successfully for %s", variant)
			}

			return nil
		},
	}
)

func init() {
	buildCmd.Flags().BoolVarP(&buildCleanFlag, "clean", "c", false, "Clean project before building")
	buildCmd.Flags().BoolVar(&buildOfflineFlag, "offline", false, "Run Gradle in offline mode")
	buildCmd.Flags().BoolVar(&buildStacktraceFlag, "stacktrace", false, "Print stacktrace on Gradle error")
	rootCmd.AddCommand(buildCmd)
}
