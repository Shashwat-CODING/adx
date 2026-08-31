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

var analyseCmd = &cobra.Command{
	Use:     "analyse [module]",
	Aliases: []string{"analyze"},
	Short:   "Run static code analysis across the project or a specific module",
	Long: `Runs the Gradle 'check' task, which includes lint, unit tests, and all
static analysis checks configured for your project.

Use 'adx lint' for just Android Lint.
Use 'adx analyse' for a full quality check: lint + tests + all code checks.

Examples:
  adx analyse
  adx analyze          # US spelling alias
  adx analyse :feature:login`,
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := GetProject()
		if err != nil {
			return err
		}

		var tasks []string
		if len(args) > 0 {
			module := strings.TrimPrefix(args[0], ":")
			tasks = []string{fmt.Sprintf(":%s:check", module)}
		} else {
			tasks = []string{"check"}
		}

		ui.Step("Running full project analysis (%s)...", strings.Join(tasks, ", "))
		runner := gradle.NewRunner(p, IsVerbose())
		if err := runner.Run(tasks); err != nil {
			return err
		}

		ui.Success("Analysis completed successfully!")

		// Print lint report path if present
		lintReport := filepath.Join(p.AppModuleDir, "build", "reports", "lint-results-debug.html")
		if _, err := os.Stat(lintReport); err == nil {
			ui.Info("Lint report:  %s", ui.ClickablePath(lintReport))
		}

		// Print test report path if present
		testReport := filepath.Join(p.AppModuleDir, "build", "reports", "tests", "testDebugUnitTest", "index.html")
		if _, err := os.Stat(testReport); err == nil {
			ui.Info("Test report:  %s", ui.ClickablePath(testReport))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(analyseCmd)
}
