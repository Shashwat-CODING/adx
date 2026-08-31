package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Shashwat-CODING/adx/internal/ui"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var prefsCmd = &cobra.Command{
	Use:     "prefs",
	Aliases: []string{"preferences"},
	Short:   "Inspect and dump SharedPreferences XML files from app storage",
	Long: `Dumps SharedPreferences XML files stored in /data/data/<package>/shared_prefs/
directly from debuggable applications on connected devices.

Examples:
  adx prefs
  adx prefs dump`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return dumpSharedPreferences()
	},
}

var prefsDumpCmd = &cobra.Command{
	Use:   "dump [filename]",
	Short: "Dump content of SharedPreferences file(s)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return dumpSharedPreferences()
	},
}

var dbCmd = &cobra.Command{
	Use:     "db",
	Aliases: []string{"database"},
	Short:   "List and pull SQLite Room databases from device to host machine",
	Long: `Inspects and downloads SQLite/Room database files stored in
/data/data/<package>/databases/ from debuggable applications.

Examples:
  adx db list
  adx db pull
  adx db pull app_database.db`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return listDatabases()
	},
}

var dbListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all database files inside app storage",
	RunE: func(cmd *cobra.Command, args []string) error {
		return listDatabases()
	},
}

var dbPullCmd = &cobra.Command{
	Use:   "pull [dbname]",
	Short: "Pull SQLite database file to current directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := GetProject()
		if err != nil {
			return err
		}
		if p.PackageName == "" {
			return fmt.Errorf("package name could not be detected")
		}

		adbClient, err := GetADBClient()
		if err != nil {
			return err
		}

		devices, err := adbClient.GetDevices()
		if err != nil || len(devices) == 0 {
			return fmt.Errorf("no connected Android devices found")
		}
		dev := devices[0]

		ui.Step("Scanning databases for %s on %s...", p.PackageName, dev.Model)

		// List files in databases/
		listOut, err := exec.Command(adbClient.AdbPath, "-s", dev.Serial, "shell", "run-as", p.PackageName, "ls", "databases/").Output()
		if err != nil {
			return fmt.Errorf("failed to access app storage. Ensure app is built in debug mode (run-as permission required): %w", err)
		}

		rawFiles := strings.Fields(strings.TrimSpace(string(listOut)))
		var dbFiles []string
		for _, f := range rawFiles {
			if !strings.HasSuffix(f, "-journal") && !strings.HasSuffix(f, "-wal") && !strings.HasSuffix(f, "-shm") {
				dbFiles = append(dbFiles, f)
			}
		}

		if len(dbFiles) == 0 {
			ui.Warn("No database files found in databases/ directory.")
			return nil
		}

		targetDb := dbFiles[0]
		if len(args) > 0 {
			targetDb = args[0]
		}

		ui.Info("Pulling database: %s...", ui.Orange().Sprint(targetDb))

		outLocal := targetDb
		pullCmd := exec.Command(adbClient.AdbPath, "-s", dev.Serial, "exec-out", "run-as", p.PackageName, "cat", "databases/"+targetDb)
		data, err := pullCmd.Output()
		if err != nil || len(data) == 0 {
			return fmt.Errorf("failed to read database %s: %w", targetDb, err)
		}

		if err := os.WriteFile(outLocal, data, 0644); err != nil {
			return fmt.Errorf("failed to write local database file: %w", err)
		}

		absPath, _ := filepath.Abs(outLocal)
		ui.Success("Database pulled successfully to: %s (%d bytes)", ui.ClickablePath(absPath), len(data))
		ui.Dim("Open with DB Browser for SQLite or SQLite Studio to inspect tables.")

		return nil
	},
}

func dumpSharedPreferences() error {
	p, err := GetProject()
	if err != nil {
		return err
	}
	if p.PackageName == "" {
		return fmt.Errorf("package name could not be detected")
	}

	adbClient, err := GetADBClient()
	if err != nil {
		return err
	}

	devices, err := adbClient.GetDevices()
	if err != nil || len(devices) == 0 {
		return fmt.Errorf("no connected Android devices found")
	}
	dev := devices[0]

	ui.Step("Scanning SharedPreferences for %s on %s...", p.PackageName, dev.Model)

	listOut, err := exec.Command(adbClient.AdbPath, "-s", dev.Serial, "shell", "run-as", p.PackageName, "ls", "shared_prefs/").Output()
	if err != nil {
		return fmt.Errorf("failed to access shared_prefs. Ensure app is built in debug mode (run-as permission required): %w", err)
	}

	files := strings.Fields(strings.TrimSpace(string(listOut)))
	if len(files) == 0 {
		ui.Warn("No SharedPreferences files found in shared_prefs/ directory.")
		return nil
	}

	for _, file := range files {
		if !strings.HasSuffix(file, ".xml") {
			continue
		}
		catCmd := exec.Command(adbClient.AdbPath, "-s", dev.Serial, "shell", "run-as", p.PackageName, "cat", "shared_prefs/"+file)
		data, err := catCmd.Output()
		if err == nil {
			fmt.Println()
			fmt.Printf("📄 %s\n", color.New(color.Bold, color.FgCyan).Sprintf("shared_prefs/%s", file))
			fmt.Println(strings.TrimSpace(string(data)))
		}
	}
	fmt.Println()
	return nil
}

func listDatabases() error {
	p, err := GetProject()
	if err != nil {
		return err
	}
	if p.PackageName == "" {
		return fmt.Errorf("package name could not be detected")
	}

	adbClient, err := GetADBClient()
	if err != nil {
		return err
	}

	devices, err := adbClient.GetDevices()
	if err != nil || len(devices) == 0 {
		return fmt.Errorf("no connected Android devices found")
	}
	dev := devices[0]

	ui.Step("Databases for %s on %s:", p.PackageName, dev.Model)

	listOut, err := exec.Command(adbClient.AdbPath, "-s", dev.Serial, "shell", "run-as", p.PackageName, "ls", "-la", "databases/").Output()
	if err != nil {
		return fmt.Errorf("failed to access databases/. Ensure app is built in debug mode (run-as permission required): %w", err)
	}

	output := strings.TrimSpace(string(listOut))
	if output == "" {
		ui.Warn("No databases found.")
		return nil
	}

	fmt.Println(output)
	fmt.Println()
	ui.Dim("Use 'adx db pull <dbname>' to download a database to your computer.")
	return nil
}

func init() {
	prefsCmd.AddCommand(prefsDumpCmd)
	rootCmd.AddCommand(prefsCmd)

	dbCmd.AddCommand(dbListCmd)
	dbCmd.AddCommand(dbPullCmd)
	rootCmd.AddCommand(dbCmd)
}
