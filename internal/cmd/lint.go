package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shashwat-CODING/adx/internal/gradle"
	"github.com/Shashwat-CODING/adx/internal/ui"
	"github.com/spf13/cobra"
)

var lintCmd = &cobra.Command{
	Use:   "lint [variant]",
	Short: "Run Android Lint code analysis",
	Long: `Runs the Gradle lint task on your Kotlin/Java codebase (defaults to debug).
Example:
  adx lint
  adx lint release`,
	RunE: func(cmd *cobra.Command, args []string) error {
		variant := "debug"
		if len(args) > 0 {
			variant = args[0]
		}

		p, err := GetProject()
		if err != nil {
			return err
		}

		capVariant := strings.ToUpper(variant[:1]) + strings.ToLower(variant[1:])
		taskName := fmt.Sprintf("lint%s", capVariant)

		ui.Step("Running Android Lint (%s)...", taskName)
		runner := gradle.NewRunner(p, IsVerbose())
		if err := runner.Run([]string{taskName}); err != nil {
			return err
		}

		ui.Success("Lint analysis completed!")

		// Look for HTML lint report
		reportPath := filepath.Join(p.AppModuleDir, "build", "reports", fmt.Sprintf("lint-results-%s.html", strings.ToLower(variant)))
		if _, err := os.Stat(reportPath); err == nil {
			ui.Info("Lint report: %s", ui.ClickablePath(reportPath))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(lintCmd)
}
