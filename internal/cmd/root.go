package cmd

import (
	"fmt"
	"os"

	"github.com/Shashwat-CODING/adx/internal/adb"
	"github.com/Shashwat-CODING/adx/internal/project"
	"github.com/Shashwat-CODING/adx/internal/ui"
	"github.com/spf13/cobra"
)

var (
	projectDirFlag string
	deviceFlag     string
	verboseFlag    bool

	rootCmd = &cobra.Command{
		Use:   "adx",
		Short: "adx - Fast & simple Android CLI build and deployment tool",
		Long: `ADX is an ultra-fast, zero-config CLI tool for Android Kotlin/Java developers.
It simplifies Gradle and ADB workflows:
  adx build [debug|release]
  adx run [debug|release] [--open]
  adx clean
  adx logs [--clear]
  adx devices
  adx doctor`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if cmd.Name() != "completion" {
				ui.PrintBannerOnce()
			}
		},
	}
)

// Execute runs the root CLI command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		ui.Error("%v", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&projectDirFlag, "dir", "C", ".", "Android project root directory")
	rootCmd.PersistentFlags().StringVarP(&deviceFlag, "device", "d", "", "Target ADB device serial")
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "Print full verbose output")

	defaultHelp := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		ui.PrintBannerOnce()
		defaultHelp(cmd, args)
	})
}

// IsVerbose returns whether verbose mode is enabled
func IsVerbose() bool {
	return verboseFlag
}

// GetProject loads the Android project context from the current or specified directory
func GetProject() (*project.Project, error) {
	startDir := projectDirFlag
	if startDir == "" {
		startDir = "."
	}
	return project.FindProject(startDir)
}

// GetADBClient resolves the ADB client
func GetADBClient() (*adb.Client, error) {
	return adb.NewClient()
}

// ResolveTargetDevices gets the target device(s) based on flag or interactive selection
func ResolveTargetDevices(client *adb.Client) ([]adb.Device, error) {
	devices, err := client.GetDevices()
	if err != nil {
		return nil, err
	}

	if len(devices) == 0 {
		return nil, fmt.Errorf("no connected Android devices or emulators found. Connect a device via USB/Wi-Fi or start an emulator")
	}

	// If explicit device flag provided
	if deviceFlag != "" {
		if deviceFlag == "all" {
			return devices, nil
		}
		for _, d := range devices {
			if d.Serial == deviceFlag {
				return []adb.Device{d}, nil
			}
		}
		return nil, fmt.Errorf("specified device '%s' not found among connected devices", deviceFlag)
	}

	// If single device
	if len(devices) == 1 {
		ui.Info("Using connected device: %s (%s)", devices[0].Model, devices[0].Serial)
		return devices, nil
	}

	// Multiple devices -> interactive prompt
	opts := make([]ui.DeviceOption, len(devices))
	for i, d := range devices {
		opts[i] = ui.DeviceOption{
			Serial: d.Serial,
			Model:  d.Model,
		}
	}

	chosen, err := ui.SelectDeviceInteractive(opts)
	if err != nil {
		return nil, err
	}

	// Map back to adb.Device
	chosenSerials := make(map[string]bool)
	for _, c := range chosen {
		chosenSerials[c.Serial] = true
	}

	var result []adb.Device
	for _, d := range devices {
		if chosenSerials[d.Serial] {
			result = append(result, d)
		}
	}

	return result, nil
}
