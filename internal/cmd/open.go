package cmd

import (
	"github.com/Shashwat-CODING/adx/internal/ui"
	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:   "open <url>",
	Short: "Open a deep link, app link, or URL on the connected device",
	Long: `Dispatches an android.intent.action.VIEW intent with the specified URI or deep link.
Ideal for testing deep link routing, OAuth redirects, and web links.

Examples:
  adx open "myapp://checkout?id=123"
  adx open "https://example.com/products/42"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		deepLink := args[0]

		adbClient, err := GetADBClient()
		if err != nil {
			return err
		}

		targetDevices, err := ResolveTargetDevices(adbClient)
		if err != nil {
			return err
		}

		for _, dev := range targetDevices {
			ui.Info("Opening deep link on %s: %s", dev.Model, deepLink)
			if err := adbClient.OpenDeepLink(dev.Serial, deepLink); err != nil {
				ui.Error("Failed to open on %s: %v", dev.Model, err)
			} else {
				ui.Success("Dispatched deep link on %s", dev.Model)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(openCmd)
}
