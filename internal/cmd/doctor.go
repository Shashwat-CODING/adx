package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Shashwat-CODING/adx/internal/adb"
	"github.com/Shashwat-CODING/adx/internal/project"
	"github.com/Shashwat-CODING/adx/internal/ui"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check your Android development environment and toolchain",
	Long:  `Validates Java JDK, Android SDK, ADB, and Gradle wrapper configuration.`,
	Run: func(cmd *cobra.Command, args []string) {
		ui.Step("Checking Android Development Environment...")

		// 1. Check Java
		javaPath, err := exec.LookPath("java")
		if err == nil {
			out, _ := exec.Command("java", "-version").CombinedOutput()
			firstLine := strings.Split(string(out), "\n")[0]
			ui.Success("Java JDK found: %s (%s)", javaPath, firstLine)
		} else {
			ui.Error("Java JDK not found in PATH")
		}

		// 2. Check Android SDK
		androidHome := os.Getenv("ANDROID_HOME")
		androidSdkRoot := os.Getenv("ANDROID_SDK_ROOT")
		if androidHome != "" {
			ui.Success("ANDROID_HOME: %s", androidHome)
		} else if androidSdkRoot != "" {
			ui.Success("ANDROID_SDK_ROOT: %s", androidSdkRoot)
		} else {
			ui.Warn("Neither ANDROID_HOME nor ANDROID_SDK_ROOT is set in your environment")
		}

		// 3. Check ADB
		adbClient, err := adb.NewClient()
		if err == nil {
			ui.Success("ADB found: %s", adbClient.AdbPath)
			devices, dErr := adbClient.GetDevices()
			if dErr == nil {
				ui.Info("Connected devices: %d", len(devices))
				for _, d := range devices {
					ui.Dim("  - %s (%s, %s)", d.Model, d.Serial, d.State)
				}
			}
		} else {
			ui.Error("ADB: %v", err)
		}

		// 4. Check Project
		p, err := project.FindProject(".")
		if err == nil {
			ui.Success("Android project detected at: %s", p.RootDir)
			ui.Info("Gradle wrapper: %s", p.GradlewPath)
			if p.PackageName != "" {
				ui.Info("Application package ID: %s", p.PackageName)
			}
			if p.LauncherActivity != "" {
				ui.Info("Launcher activity: %s", p.LauncherActivity)
			}
		} else {
			ui.Dim("No Android project detected in current directory (run from inside an Android project)")
		}

		fmt.Println()
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
