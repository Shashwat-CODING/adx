package cmd

import (
	"fmt"

	"github.com/Shashwat-CODING/adx/internal/deps"
	"github.com/Shashwat-CODING/adx/internal/gradle"
	"github.com/Shashwat-CODING/adx/internal/ui"
	"github.com/spf13/cobra"
)

var (
	addConfigFlag string
	addNoSyncFlag bool

	addCmd = &cobra.Command{
		Use:   "add <dependency>",
		Short: "Add a library dependency to the project and sync/download it",
		Long: `Searches Maven Central, adds the dependency to your build files (or libs.versions.toml),
and triggers Gradle to download and cache it locally.

Examples:
  adx add retrofit
  adx add coil-compose
  adx add androidx.room:room-runtime
  adx add com.squareup.moshi:moshi:1.15.1
  adx add ksp --config ksp
  adx add junit --config testImplementation`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]

			p, err := GetProject()
			if err != nil {
				return err
			}

			spinner := ui.NewSpinner(fmt.Sprintf("Searching Maven Central for '%s'...", query))
			dep, err := deps.ResolveDependency(query)
			if err != nil {
				spinner.StopFail("Failed to resolve dependency: %v", err)
				return err
			}
			spinner.StopSuccess("Resolved %s", dep.Coordinate())

			ui.Info("Adding %s (%s) to %s...", dep.Coordinate(), addConfigFlag, p.AppModuleName)
			msg, err := deps.AddDependency(p.RootDir, p.AppModuleDir, dep, addConfigFlag)
			if err != nil {
				return err
			}
			ui.Success("%s", msg)

			if !addNoSyncFlag {
				ui.Step("Syncing and downloading dependency with Gradle...")
				runner := gradle.NewRunner(p, IsVerbose())
				// Running 'dependencies --refresh-dependencies' downloads and resolves without full build
				if err := runner.Run([]string{"dependencies", "--refresh-dependencies"}); err != nil {
					ui.Warn("Gradle sync had warnings/errors: %v", err)
				} else {
					ui.Success("Dependency %s downloaded and installed successfully!", dep.Coordinate())
				}
			}

			return nil
		},
	}
)

func init() {
	addCmd.Flags().StringVarP(&addConfigFlag, "config", "c", "implementation", "Dependency configuration (implementation, api, ksp, testImplementation)")
	addCmd.Flags().BoolVar(&addNoSyncFlag, "no-sync", false, "Add to build files without triggering Gradle download")
	rootCmd.AddCommand(addCmd)
}
