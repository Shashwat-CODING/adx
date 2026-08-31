package cmd

import (
	"fmt"

	"github.com/Shashwat-CODING/adx/internal/ui"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall [package_name]",
	Short: "Uninstall the application from connected device(s)",
	Long: `Uninstalls the detected project application (or specified package) from target device(s).

Examples:
  adx uninstall
  adx uninstall com.example.myapp`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var pkgName string
		if len(args) > 0 {
			pkgName = args[0]
		} else {
			p, err := GetProject()
			if err != nil {
				return err
			}
			pkgName = p.PackageName
		}

		if pkgName == "" {
			return fmt.Errorf("could not detect package name. Please provide it explicitly: adx uninstall <pkg>")
		}

		adbClient, err := GetADBClient()
		if err != nil {
			return err
		}

		targetDevices, err := ResolveTargetDevices(adbClient)
		if err != nil {
			return err
		}

		for _, dev := range targetDevices {
			ui.Info("Uninstalling %s from %s...", pkgName, dev.Model)
			if err := adbClient.Uninstall(dev.Serial, pkgName); err != nil {
				ui.Error("Failed to uninstall from %s: %v", dev.Model, err)
			} else {
				ui.Success("Uninstalled %s from %s", pkgName, dev.Model)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
}
