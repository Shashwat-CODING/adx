package cmd

import (
	"github.com/Shashwat-CODING/adx/internal/gradle"
	"github.com/Shashwat-CODING/adx/internal/ui"
	"github.com/spf13/cobra"
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean Gradle build cache and output directories",
	Long: `Runs the Gradle clean task on the Android project.
Equivalent to ./gradlew clean`,
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := GetProject()
		if err != nil {
			return err
		}

		runner := gradle.NewRunner(p, IsVerbose())
		if err := runner.Clean(); err != nil {
			return err
		}

		ui.Success("Project cleaned successfully!")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(cleanCmd)
}
