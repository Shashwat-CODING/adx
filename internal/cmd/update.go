package cmd

import (
	"fmt"

	"github.com/Shashwat-CODING/adx/internal/deps"
	"github.com/Shashwat-CODING/adx/internal/gradle"
	"github.com/Shashwat-CODING/adx/internal/ui"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	updateCheckOnlyFlag bool
	updateNoSyncFlag    bool

	updateCmd = &cobra.Command{
		Use:   "update",
		Short: "Check and update project dependencies to latest versions",
		Long: `Scans your gradle/libs.versions.toml and build.gradle[.kts] files,
queries Maven Central for newer stable releases, updates the version strings,
and downloads the updated dependencies with Gradle.

Examples:
  adx update
  adx update --check
  adx update --no-sync`,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := GetProject()
			if err != nil {
				return err
			}

			spinner := ui.NewSpinner("Checking Maven Central for dependency updates...")
			results, err := deps.UpdateDependencies(p.RootDir, p.AppModuleDir, updateCheckOnlyFlag)
			if err != nil {
				spinner.StopFail("Failed checking dependencies: %v", err)
				return err
			}

			if len(results) == 0 {
				spinner.StopSuccess("All project dependencies are already up-to-date!")
				return nil
			}

			spinner.StopSuccess("Found %d dependency update(s)", len(results))

			ui.Step("Dependency Updates:")
			for _, r := range results {
				fmt.Printf("  • %-35s %s ➜ %s  %s\n",
					color.New(color.Bold).Sprint(r.Coordinate),
					color.New(color.FgRed).Sprint(r.OldVersion),
					color.New(color.FgGreen, color.Bold).Sprint(r.NewVersion),
					color.New(color.Faint).Sprintf("(%s)", r.File),
				)
			}

			if updateCheckOnlyFlag {
				ui.Info("Run 'adx update' without --check to apply these updates.")
				return nil
			}

			ui.Success("Updated %d dependencies in project build configuration!", len(results))

			if !updateNoSyncFlag {
				ui.Step("Syncing updated dependencies with Gradle...")
				runner := gradle.NewRunner(p, IsVerbose())
				if err := runner.Run([]string{"dependencies", "--refresh-dependencies"}); err != nil {
					ui.Warn("Gradle sync had warnings: %v", err)
				} else {
					ui.Success("All updated dependencies synced and cached!")
				}
			}

			return nil
		},
	}
)

func init() {
	updateCmd.Flags().BoolVarP(&updateCheckOnlyFlag, "check", "c", false, "Check for available updates without modifying files")
	updateCmd.Flags().BoolVar(&updateNoSyncFlag, "no-sync", false, "Update build files without triggering Gradle download")
	rootCmd.AddCommand(updateCmd)
}
