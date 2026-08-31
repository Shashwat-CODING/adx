package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Shashwat-CODING/adx/internal/adb"
	"github.com/Shashwat-CODING/adx/internal/ui"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var emulatorCmd = &cobra.Command{
	Use:   "emulator [avd_name]",
	Short: "Launch an Android Virtual Device (AVD)",
	Long: `Lists available Android Virtual Devices and launches an emulator.
If only one AVD exists, it starts automatically.
If multiple AVDs exist, an interactive selection menu is displayed.

Examples:
  adx emulator
  adx run emulator
  adx emulator Pixel_8_API_34`,
	RunE: func(cmd *cobra.Command, args []string) error {
		avds, err := adb.ListAVDs()
		if err != nil {
			return err
		}

		if len(avds) == 0 {
			ui.Warn("No Android Virtual Devices (AVDs) found.")
			ui.Dim("You can create an AVD using Android Studio Device Manager.")
			return nil
		}

		var targetAVD string

		// If user specified an explicit AVD name
		if len(args) > 0 {
			targetAVD = args[0]
		} else if len(avds) == 1 {
			targetAVD = avds[0]
			ui.Info("Auto-launching detected AVD: %s", ui.Orange().Sprint(targetAVD))
		} else {
			// Multiple AVDs -> interactive selection
			fmt.Println()
			ui.Orange().Println("Multiple AVDs detected. Please select one to launch:")
			for i, avd := range avds {
				fmt.Printf("  [%s] 📱 %s\n",
					ui.Orange().Sprintf("%d", i+1),
					color.New(color.Bold).Sprint(avd),
				)
			}

			reader := bufio.NewReader(os.Stdin)
			for {
				fmt.Print("\nEnter choice [1-" + strconv.Itoa(len(avds)) + "]: ")
				input, err := reader.ReadString('\n')
				if err != nil {
					return err
				}

				num, err := strconv.Atoi(strings.TrimSpace(input))
				if err == nil && num >= 1 && num <= len(avds) {
					targetAVD = avds[num-1]
					break
				}
				ui.Warn("Invalid selection. Please enter a number between 1 and %d.", len(avds))
			}
		}

		ui.Step("Launching emulator: %s...", targetAVD)
		if err := adb.StartEmulator(targetAVD); err != nil {
			return err
		}

		ui.Success("Emulator '%s' started in the background!", targetAVD)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(emulatorCmd)
	runCmd.AddCommand(emulatorCmd)
}
