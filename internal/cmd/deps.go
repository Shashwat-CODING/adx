package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Shashwat-CODING/adx/internal/ui"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	depsFindFlag          string
	depsTreeFlag          bool
	depsConfigurationFlag string
)

type DependencyMatch struct {
	Configuration string `json:"configuration"`
	Coordinate    string `json:"coordinate"`
	Resolved      string `json:"resolved,omitempty"`
	TreePath      string `json:"tree_path,omitempty"`
}

type DepsResult struct {
	Query           string            `json:"query,omitempty"`
	Matches         []DependencyMatch `json:"matches"`
	DirectCount     int               `json:"direct_count"`
	TotalFoundCount int               `json:"total_found_count"`
}

var depsCmd = &cobra.Command{
	Use:     "deps",
	Aliases: []string{"dependencies"},
	Short:   "Lookup and analyze project dependency trees and version conflicts",
	Long: `Queries Gradle or scans project build files to find library dependency resolution paths,
transitively included versions, and potential version conflicts.

Examples:
  adx deps
  adx deps --find coil
  adx deps --find retrofit --json
  adx deps --tree
  adx deps --config releaseRuntimeClasspath`,
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := GetProject()
		if err != nil {
			return err
		}

		if !IsJSON() {
			if depsFindFlag != "" {
				ui.Step("Searching dependency tree for '%s' in %s...", depsFindFlag, p.AppModuleName)
			} else {
				ui.Step("Inspecting dependencies for %s...", p.AppModuleName)
			}
		}

		// If --find or --tree is requested, invoke Gradle dependencies task
		if depsFindFlag != "" || depsTreeFlag {
			gradleArgs := []string{fmt.Sprintf(":%s:dependencies", p.AppModuleName)}
			if depsConfigurationFlag != "" {
				gradleArgs = append(gradleArgs, "--configuration", depsConfigurationFlag)
			}

			spinner := ui.NewSpinner("Running Gradle dependency analysis...")
			c := exec.Command(p.GradlewPath, gradleArgs...)
			c.Dir = p.RootDir
			out, err := c.CombinedOutput()
			if err != nil && len(out) == 0 {
				spinner.StopFail("Failed to run Gradle dependencies: %v", err)
				return err
			}
			spinner.StopSuccess("Dependency tree resolved")

			matches, total := parseGradleDepsOutput(string(out), depsFindFlag)
			res := DepsResult{
				Query:           depsFindFlag,
				Matches:         matches,
				TotalFoundCount: total,
			}

			if IsJSON() {
				data, _ := json.MarshalIndent(res, "", "  ")
				fmt.Println(string(data))
				return nil
			}

			fmt.Println()
			if len(matches) == 0 {
				if depsFindFlag != "" {
					ui.Warn("No dependencies matching '%s' were found in the dependency graph.", depsFindFlag)
				} else {
					ui.Info("No dependencies found.")
				}
				return nil
			}

			ui.Success("Found %d occurrences of '%s' in dependency graph:", len(matches), depsFindFlag)
			for i, m := range matches {
				fmt.Printf("  [%d] %s: %s\n",
					i+1,
					color.New(color.Bold, color.FgCyan).Sprint(m.Configuration),
					ui.Orange().Sprint(m.Coordinate),
				)
				if m.TreePath != "" {
					fmt.Printf("      %s\n", color.New(color.Faint).Sprint(m.TreePath))
				}
			}
			fmt.Println()
			return nil
		}

		// Quick static view from build files & libs.versions.toml
		catalogDeps := readCatalogDependencies(p.RootDir)
		if IsJSON() {
			res := DepsResult{
				Matches:         catalogDeps,
				DirectCount:     len(catalogDeps),
				TotalFoundCount: len(catalogDeps),
			}
			data, _ := json.MarshalIndent(res, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Println()
		ui.Success("Version Catalog & Build Dependencies (%d declared):", len(catalogDeps))
		for i, d := range catalogDeps {
			fmt.Printf("  [%2d] %-32s  %s\n",
				i+1,
				color.New(color.Bold).Sprint(d.Configuration),
				ui.OrangeSoft().Sprint(d.Coordinate),
			)
		}
		fmt.Println()
		ui.Dim("Tip: Use 'adx deps --find <name>' to search the full transitive resolution tree.")
		fmt.Println()

		return nil
	},
}

func parseGradleDepsOutput(output string, query string) ([]DependencyMatch, int) {
	var matches []DependencyMatch
	scanner := bufio.NewScanner(strings.NewReader(output))
	var currentConfig string
	lowerQuery := strings.ToLower(query)

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Match configuration headers like "debugCompileClasspath - Resolved configuration..."
		if strings.Contains(line, " - ") && !strings.HasPrefix(line, "+---") && !strings.HasPrefix(line, "\\---") && !strings.HasPrefix(line, "|") {
			parts := strings.Split(trimmed, " - ")
			currentConfig = parts[0]
			continue
		}

		if (strings.HasPrefix(trimmed, "+---") || strings.HasPrefix(trimmed, "\\---") || strings.HasPrefix(trimmed, "|")) && (query == "" || strings.Contains(strings.ToLower(trimmed), lowerQuery)) {
			// Extract dependency coordinate
			cleanDep := strings.TrimPrefix(trimmed, "+--- ")
			cleanDep = strings.TrimPrefix(cleanDep, "\\--- ")
			cleanDep = strings.TrimPrefix(cleanDep, "|    ")
			cleanDep = strings.TrimPrefix(cleanDep, "|    ")
			cleanDep = strings.TrimSpace(cleanDep)

			cfg := currentConfig
			if cfg == "" {
				cfg = "default"
			}

			matches = append(matches, DependencyMatch{
				Configuration: cfg,
				Coordinate:    cleanDep,
				TreePath:      trimmed,
			})
		}
	}

	return matches, len(matches)
}

func readCatalogDependencies(rootDir string) []DependencyMatch {
	var matches []DependencyMatch
	catalogPath := filepath.Join(rootDir, "gradle", "libs.versions.toml")
	if data, err := os.ReadFile(catalogPath); err == nil {
		content := string(data)
		reLibrary := regexp.MustCompile(`(?m)^([a-zA-Z0-9_-]+)\s*=\s*(?:\{[^}]*module\s*=\s*["']([^"']+)["'][^}]*\}|["']([^"']+)["'])`)
		for _, m := range reLibrary.FindAllStringSubmatch(content, -1) {
			alias := m[1]
			coord := m[2]
			if coord == "" {
				coord = m[3]
			}
			matches = append(matches, DependencyMatch{
				Configuration: alias,
				Coordinate:    coord,
			})
		}
	}
	return matches
}

// Analyze Build Command
type BuildDiagnostics struct {
	GradleVersion string            `json:"gradle_version"`
	Daemons       []GradleDaemon    `json:"daemons"`
	JVMArgs       string            `json:"jvm_args,omitempty"`
	BuildCache    bool              `json:"build_cache_enabled"`
	ConfigurationAvoidance bool     `json:"configuration_cache_enabled"`
	Parallel      bool              `json:"parallel_enabled"`
	Tips          []string          `json:"tips,omitempty"`
}

type GradleDaemon struct {
	PID    string `json:"pid"`
	Status string `json:"status"`
	Info   string `json:"info"`
}

var analyzeBuildCmd = &cobra.Command{
	Use:     "analyze-build",
	Aliases: []string{"build-diag", "diagnose-build"},
	Short:   "Instant build-cache and Gradle daemon diagnostics for build performance",
	Long: `Inspects Gradle daemon processes, build-cache settings, JVM memory parameters,
and Gradle optimization flags to speed up Android build times.

Examples:
  adx analyze-build
  adx analyze-build --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := GetProject()
		if err != nil {
			return err
		}

		if !IsJSON() {
			ui.Step("Inspecting Gradle Daemons & Build Performance Settings...")
		}

		diag := BuildDiagnostics{}

		// 1. Read gradle.properties
		gradlePropsPath := filepath.Join(p.RootDir, "gradle.properties")
		if data, err := os.ReadFile(gradlePropsPath); err == nil {
			content := string(data)
			reJVM := regexp.MustCompile(`(?m)^org\.gradle\.jvmargs\s*=\s*(.*)$`)
			if m := reJVM.FindStringSubmatch(content); len(m) > 1 {
				diag.JVMArgs = strings.TrimSpace(m[1])
			}
			if strings.Contains(content, "org.gradle.caching=true") {
				diag.BuildCache = true
			}
			if strings.Contains(content, "org.gradle.parallel=true") {
				diag.Parallel = true
			}
			if strings.Contains(content, "org.gradle.configuration-cache=true") {
				diag.ConfigurationAvoidance = true
			}
		}

		// 2. Query Gradle Daemons
		c := exec.Command(p.GradlewPath, "--status")
		c.Dir = p.RootDir
		if out, err := c.Output(); err == nil {
			diag.Daemons = parseDaemonStatus(string(out))
		}

		// 3. Generate tips
		if !diag.BuildCache {
			diag.Tips = append(diag.Tips, "Enable build cache: Add 'org.gradle.caching=true' in gradle.properties")
		}
		if !diag.Parallel {
			diag.Tips = append(diag.Tips, "Enable parallel builds: Add 'org.gradle.parallel=true' in gradle.properties")
		}
		if !diag.ConfigurationAvoidance {
			diag.Tips = append(diag.Tips, "Enable configuration cache: Add 'org.gradle.configuration-cache=true' in gradle.properties")
		}
		if diag.JVMArgs == "" || !strings.Contains(diag.JVMArgs, "-Xmx") {
			diag.Tips = append(diag.Tips, "Allocate sufficient Gradle RAM: Add 'org.gradle.jvmargs=-Xmx4096m -XX:+UseParallelGC' in gradle.properties")
		}

		if IsJSON() {
			data, _ := json.MarshalIndent(diag, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Println()
		fmt.Printf("  %-28s %s\n", color.New(color.Bold).Sprint("Gradle JVM Args:"), valueOrUnknown(diag.JVMArgs))
		fmt.Printf("  %-28s %s\n", color.New(color.Bold).Sprint("Build Cache:"), boolStatus(diag.BuildCache))
		fmt.Printf("  %-28s %s\n", color.New(color.Bold).Sprint("Parallel Execution:"), boolStatus(diag.Parallel))
		fmt.Printf("  %-28s %s\n", color.New(color.Bold).Sprint("Configuration Cache:"), boolStatus(diag.ConfigurationAvoidance))

		fmt.Println()
		ui.Step("Running Gradle Daemons (%d):", len(diag.Daemons))
		if len(diag.Daemons) == 0 {
			ui.Dim("  No active Gradle daemons found.")
		} else {
			for _, d := range diag.Daemons {
				statusCol := color.FgGreen
				if d.Status != "IDLE" && d.Status != "BUSY" {
					statusCol = color.FgYellow
				}
				fmt.Printf("  PID %-8s  %-10s  %s\n",
					color.New(color.Bold).Sprint(d.PID),
					color.New(statusCol, color.Bold).Sprint(d.Status),
					color.New(color.Faint).Sprint(d.Info),
				)
			}
		}

		if len(diag.Tips) > 0 {
			fmt.Println()
			ui.Step("Optimization Recommendations:")
			for _, tip := range diag.Tips {
				ui.Info("%s", tip)
			}
		}
		fmt.Println()

		return nil
	},
}

func boolStatus(b bool) string {
	if b {
		return color.New(color.FgGreen, color.Bold).Sprint("Enabled ✔")
	}
	return color.New(color.FgYellow).Sprint("Disabled (Can be optimized)")
}

func parseDaemonStatus(output string) []GradleDaemon {
	var daemons []GradleDaemon
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "PID") || strings.HasPrefix(line, "---") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			d := GradleDaemon{
				PID:    fields[0],
				Status: fields[1],
			}
			if len(fields) > 2 {
				d.Info = strings.Join(fields[2:], " ")
			}
			daemons = append(daemons, d)
		}
	}
	return daemons
}

func init() {
	depsCmd.Flags().StringVar(&depsFindFlag, "find", "", "Filter and search dependency tree for a specific library name")
	depsCmd.Flags().BoolVar(&depsTreeFlag, "tree", false, "Print full resolution tree via Gradle dependencies")
	depsCmd.Flags().StringVarP(&depsConfigurationFlag, "config", "c", "", "Filter by specific Gradle configuration (e.g. implementation, releaseRuntimeClasspath)")
	rootCmd.AddCommand(depsCmd)
	rootCmd.AddCommand(analyzeBuildCmd)
}
