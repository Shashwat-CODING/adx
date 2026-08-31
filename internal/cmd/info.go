package cmd

import (
	"fmt"

	"github.com/Shashwat-CODING/adx/internal/ui"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Inspect detected project and device configuration",
	Long:  `Displays detected Android project root, module structure, package ID, and attached devices.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := GetProject()
		if err != nil {
			return err
		}

		bold := color.New(color.Bold)
		orange := ui.OrangeSoft()

		ui.Step("Android Project Information")
		fmt.Printf("  %-20s %s\n", bold.Sprint("Root Directory:"), p.RootDir)
		fmt.Printf("  %-20s %s\n", bold.Sprint("App Module:"), p.AppModuleName)
		fmt.Printf("  %-20s %s\n", bold.Sprint("Gradle Wrapper:"), p.GradlewPath)
		if p.PackageName != "" {
			fmt.Printf("  %-20s %s\n", bold.Sprint("Package ID:"), orange.Sprint(p.PackageName))
		}
		if p.LauncherActivity != "" {
			fmt.Printf("  %-20s %s\n", bold.Sprint("Launcher Activity:"), orange.Sprint(p.LauncherActivity))
		}

		adbClient, err := GetADBClient()
		if err == nil {
			devices, dErr := adbClient.GetDevices()
			if dErr == nil {
				fmt.Println()
				ui.Step("Connected Devices (%d)", len(devices))
				for _, d := range devices {
					fmt.Printf("  - %s (%s, %s)\n", bold.Sprint(d.Model), d.Serial, d.State)
				}
			}
		}

		fmt.Println()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
