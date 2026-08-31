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

var bundleCmd = &cobra.Command{
	Use:     "bundle [variant]",
	Aliases: []string{"aab", "abb"},
	Short:   "Build Android App Bundle (.aab) for Google Play",
	Long: `Builds an Android App Bundle (.aab) for publishing (defaults to release).
Examples:
  adx bundle
  adx aab
  adx abb
  adx bundle release
  adx bundle debug`,
	RunE: func(cmd *cobra.Command, args []string) error {
		variant := "release"
		if len(args) > 0 {
			variant = args[0]
		}

		p, err := GetProject()
		if err != nil {
			return err
		}

		capVariant := strings.ToUpper(variant[:1]) + strings.ToLower(variant[1:])
		taskName := fmt.Sprintf("bundle%s", capVariant)

		ui.Step("Building App Bundle (%s)...", taskName)
		runner := gradle.NewRunner(p, IsVerbose())
		if err := runner.Run([]string{taskName}); err != nil {
			return err
		}

		// Look for generated .aab
		cleanVariant := strings.ToLower(variant)
		bundleDirs := []string{
			filepath.Join(p.AppModuleDir, "build", "outputs", "bundle", cleanVariant),
			filepath.Join(p.AppModuleDir, "build", "outputs", "bundle"),
			filepath.Join(p.RootDir, "build", "outputs", "bundle", cleanVariant),
		}

		var foundAab string
		for _, dir := range bundleDirs {
			if _, err := os.Stat(dir); err != nil {
				continue
			}
			_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				if err == nil && !info.IsDir() && strings.HasSuffix(path, ".aab") {
					foundAab = path
				}
				return nil
			})
			if foundAab != "" {
				break
			}
		}

		if foundAab != "" {
			if fi, err := os.Stat(foundAab); err == nil {
				sizeMB := float64(fi.Size()) / (1024 * 1024)
				ui.Success("App Bundle generated: %s (%.2f MB)", ui.ClickablePath(foundAab), sizeMB)
			} else {
				ui.Success("App Bundle generated: %s", ui.ClickablePath(foundAab))
			}
		} else {
			ui.Success("Bundle build finished for %s", variant)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(bundleCmd)
}
