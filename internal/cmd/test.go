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

var testCmd = &cobra.Command{
	Use:   "test [variant]",
	Short: "Run unit tests for the Android project",
	Long: `Runs the Gradle unit test task for the specified variant (defaults to debug).
Example:
  adx test
  adx test release
  adx test -v`,
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
		taskName := fmt.Sprintf("test%sUnitTest", capVariant)

		ui.Step("Running unit tests (%s)...", taskName)
		runner := gradle.NewRunner(p, IsVerbose())
		if err := runner.Run([]string{taskName}); err != nil {
			return err
		}

		ui.Success("All unit tests passed!")

		// Look for HTML report
		reportPath := filepath.Join(p.AppModuleDir, "build", "reports", "tests", taskName, "index.html")
		if _, err := os.Stat(reportPath); err == nil {
			ui.Info("Test report: %s", ui.ClickablePath(reportPath))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(testCmd)
}
