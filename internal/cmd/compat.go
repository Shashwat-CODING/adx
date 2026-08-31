package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Shashwat-CODING/adx/internal/ui"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var compatCmd = &cobra.Command{
	Use:     "check-compat",
	Aliases: []string{"compat", "matrix"},
	Short:   "Check Kotlin, Compose Compiler, AGP, and Gradle version compatibility",
	Long: `Inspects your project build files and Version Catalog to validate
version compatibility between Kotlin, Compose Compiler, Android Gradle Plugin (AGP),
and Gradle wrapper.

Examples:
  adx check-compat
  adx compat`,
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := GetProject()
		if err != nil {
			return err
		}

		ui.Step("Analyzing Toolchain & Version Compatibility...")

		var gradleVer, agpVer, kotlinVer, composeCompilerVer string
		var usesKotlin2ComposePlugin bool

		// 1. Read gradle-wrapper.properties
		wrapperProps := filepath.Join(p.RootDir, "gradle", "wrapper", "gradle-wrapper.properties")
		if data, err := os.ReadFile(wrapperProps); err == nil {
			reGradle := regexp.MustCompile(`gradle-([0-9]+\.[0-9]+(\.[0-9]+)?)-`)
			if m := reGradle.FindStringSubmatch(string(data)); len(m) > 1 {
				gradleVer = m[1]
			}
		}

		// 2. Read Version Catalog (libs.versions.toml) if present
		catalogPath := filepath.Join(p.RootDir, "gradle", "libs.versions.toml")
		if data, err := os.ReadFile(catalogPath); err == nil {
			content := string(data)
			reAGP := regexp.MustCompile(`(?i)(?:agp|androidGradlePlugin)\s*=\s*["']([^"']+)["']`)
			reKotlin := regexp.MustCompile(`(?i)kotlin\s*=\s*["']([^"']+)["']`)
			reComposeComp := regexp.MustCompile(`(?i)(?:composeCompiler|compose-compiler)\s*=\s*["']([^"']+)["']`)

			if m := reAGP.FindStringSubmatch(content); len(m) > 1 {
				agpVer = m[1]
			}
			if m := reKotlin.FindStringSubmatch(content); len(m) > 1 {
				kotlinVer = m[1]
			}
			if m := reComposeComp.FindStringSubmatch(content); len(m) > 1 {
				composeCompilerVer = m[1]
			}
			if strings.Contains(content, "kotlin.compose") || strings.Contains(content, "kotlin-compose") || strings.Contains(content, "org.jetbrains.kotlin.plugin.compose") {
				usesKotlin2ComposePlugin = true
			}
		}

		// 3. Scan build.gradle.kts / build.gradle files
		buildFiles := []string{
			filepath.Join(p.AppModuleDir, "build.gradle.kts"),
			filepath.Join(p.AppModuleDir, "build.gradle"),
			filepath.Join(p.RootDir, "build.gradle.kts"),
			filepath.Join(p.RootDir, "build.gradle"),
		}

		for _, bf := range buildFiles {
			data, err := os.ReadFile(bf)
			if err != nil {
				continue
			}
			content := string(data)

			if strings.Contains(content, "kotlin.compose") || strings.Contains(content, "org.jetbrains.kotlin.plugin.compose") {
				usesKotlin2ComposePlugin = true
			}
			if agpVer == "" {
				reAGPDirect := regexp.MustCompile(`com\.android\.(?:application|library)\s+version\s+["']([^"']+)["']`)
				if m := reAGPDirect.FindStringSubmatch(content); len(m) > 1 {
					agpVer = m[1]
				}
			}
			if kotlinVer == "" {
				reKotDirect := regexp.MustCompile(`org\.jetbrains\.kotlin\.android\s+version\s+["']([^"']+)["']`)
				if m := reKotDirect.FindStringSubmatch(content); len(m) > 1 {
					kotlinVer = m[1]
				}
			}
		}

		bold := color.New(color.Bold)
		orange := ui.Orange()

		fmt.Println()
		fmt.Printf("  %-25s %s\n", bold.Sprint("Gradle Version:"), valueOrUnknown(gradleVer))
		fmt.Printf("  %-25s %s\n", bold.Sprint("AGP Version:"), valueOrUnknown(agpVer))
		fmt.Printf("  %-25s %s\n", bold.Sprint("Kotlin Version:"), valueOrUnknown(kotlinVer))
		if usesKotlin2ComposePlugin {
			fmt.Printf("  %-25s %s\n", bold.Sprint("Compose Compiler:"), orange.Sprint("Kotlin 2.0+ Compiler Plugin (Bundled)"))
		} else if composeCompilerVer != "" {
			fmt.Printf("  %-25s %s\n", bold.Sprint("Compose Compiler:"), composeCompilerVer)
		} else {
			fmt.Printf("  %-25s %s\n", bold.Sprint("Compose Compiler:"), color.New(color.Faint).Sprint("Not specified / using defaults"))
		}
		fmt.Println()

		// Compatibility Rules Evaluation
		ui.Step("Compatibility Verdict:")

		var issuesFound bool

		// Rule 1: Kotlin 2.0+ and Compose Compiler
		if isVersionGTE(kotlinVer, "2.0.0") {
			if usesKotlin2ComposePlugin {
				ui.Success("Kotlin 2.0+ Compose Compiler: Configured with official JetBrains Compose plugin.")
			} else if composeCompilerVer != "" {
				ui.Warn("Kotlin 2.0+ detected, but legacy compose-compiler version (%s) is declared.", composeCompilerVer)
				ui.Dim("  Recommendation: Migrate to 'org.jetbrains.kotlin.plugin.compose' plugin.")
				issuesFound = true
			}
		} else if kotlinVer != "" && composeCompilerVer != "" {
			ui.Info("Legacy Kotlin 1.x (<2.0) detected with Compose Compiler extension %s.", composeCompilerVer)
		}

		// Rule 2: AGP vs Gradle compatibility
		if agpVer != "" && gradleVer != "" {
			minGradle := getMinimumGradleForAGP(agpVer)
			if minGradle != "" {
				if isVersionGTE(gradleVer, minGradle) {
					ui.Success("AGP %s is compatible with Gradle %s (requires >= %s).", agpVer, gradleVer, minGradle)
				} else {
					ui.Warn("AGP %s requires Gradle %s or higher, but project uses %s.", agpVer, minGradle, gradleVer)
					issuesFound = true
				}
			}
		}

		// Rule 3: Java 17+ requirement for AGP 8.0+
		if isVersionGTE(agpVer, "8.0.0") {
			ui.Success("AGP 8.x+ requires JDK 17+ (Run 'adx doctor' to verify local JDK).")
		}

		if !issuesFound {
			fmt.Println()
			ui.Success("All detected toolchain versions are mutually compatible! 🚀")
		}

		return nil
	},
}

func valueOrUnknown(v string) string {
	if v == "" {
		return color.New(color.Faint).Sprint("Unknown")
	}
	return color.New(color.Bold, color.FgCyan).Sprint(v)
}

func isVersionGTE(ver string, min string) bool {
	if ver == "" || min == "" {
		return false
	}
	return compareVersions(ver, min) >= 0
}

func compareVersions(v1, v2 string) int {
	clean := func(v string) []int {
		parts := strings.Split(strings.Split(v, "-")[0], ".")
		res := make([]int, len(parts))
		for i, p := range parts {
			var num int
			fmt.Sscanf(p, "%d", &num)
			res[i] = num
		}
		return res
	}
	p1 := clean(v1)
	p2 := clean(v2)
	maxLen := len(p1)
	if len(p2) > maxLen {
		maxLen = len(p2)
	}
	for i := 0; i < maxLen; i++ {
		var n1, n2 int
		if i < len(p1) {
			n1 = p1[i]
		}
		if i < len(p2) {
			n2 = p2[i]
		}
		if n1 > n2 {
			return 1
		}
		if n1 < n2 {
			return -1
		}
	}
	return 0
}

func getMinimumGradleForAGP(agp string) string {
	if strings.HasPrefix(agp, "8.9") || strings.HasPrefix(agp, "8.8") {
		return "8.10"
	}
	if strings.HasPrefix(agp, "8.7") {
		return "8.9"
	}
	if strings.HasPrefix(agp, "8.6") || strings.HasPrefix(agp, "8.5") {
		return "8.7"
	}
	if strings.HasPrefix(agp, "8.4") {
		return "8.6"
	}
	if strings.HasPrefix(agp, "8.3") {
		return "8.4"
	}
	if strings.HasPrefix(agp, "8.2") {
		return "8.2"
	}
	if strings.HasPrefix(agp, "8.1") {
		return "8.0"
	}
	if strings.HasPrefix(agp, "8.0") {
		return "8.0"
	}
	return ""
}

func init() {
	rootCmd.AddCommand(compatCmd)
}
