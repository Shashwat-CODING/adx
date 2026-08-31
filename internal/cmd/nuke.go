package cmd

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Shashwat-CODING/adx/internal/ui"
	"github.com/spf13/cobra"
)

var nukeCmd = &cobra.Command{
	Use:   "nuke",
	Short: "Deep clean: delete .gradle, build folders, and stop daemons",
	Long: `Resolves corrupted Kotlin/KSP/KAPT caches and stuck builds by:
1. Stopping all running Gradle and Kotlin daemons (./gradlew --stop)
2. Recursively deleting .gradle/ in the project root
3. Recursively deleting all build/ directories across all modules`,
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := GetProject()
		if err != nil {
			return err
		}

		ui.Step("Stopping Gradle and Kotlin daemons...")
		stopCmd := exec.Command(p.GradlewPath, "--stop")
		stopCmd.Dir = p.RootDir
		_ = stopCmd.Run()

		spinner := ui.NewSpinner("Nuking build caches and artifacts...")

		// 1. Delete root .gradle
		dotGradle := filepath.Join(p.RootDir, ".gradle")
		if err := os.RemoveAll(dotGradle); err == nil {
			ui.Info("Removed: %s", dotGradle)
		}

		// 2. Delete all 'build' directories
		deletedCount := 0
		_ = filepath.Walk(p.RootDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || !info.IsDir() {
				return nil
			}
			if info.Name() == "build" {
				if removeErr := os.RemoveAll(path); removeErr == nil {
					deletedCount++
					ui.Info("Removed: %s", path)
				}
				return filepath.SkipDir
			}
			return nil
		})

		spinner.StopSuccess("Nuke completed! Removed %d build directories and .gradle cache", deletedCount)
		ui.Success("Your project is clean. Next build will perform a 100%% fresh compile.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(nukeCmd)
}
