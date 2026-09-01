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
	crashPackageFlag string
	crashLinesFlag   int
)

type CrashSummary struct {
	Device         string   `json:"device"`
	Package        string   `json:"package"`
	CrashType      string   `json:"crash_type"` // FATAL EXCEPTION, ANR, SIGSEGV, UNCAUGHT EXCEPTION
	ExceptionClass string   `json:"exception_class"`
	Message        string   `json:"message"`
	RootCauseFile  string   `json:"root_cause_file,omitempty"`
	RootCauseLine  int      `json:"root_cause_line,omitempty"`
	StackTrace     []string `json:"stack_trace"`
	RawLog         string   `json:"raw_log,omitempty"`
}

var crashCmd = &cobra.Command{
	Use:     "crash",
	Aliases: []string{"trace", "anr", "exception"},
	Short:   "Extract, summarize, and de-obfuscate recent app crashes and ANRs",
	Long: `Fetches the latest crash, fatal exception, or ANR stack trace from logcat
for your application package, strips logcat noise, extracts root cause, and links to source files.

Examples:
  adx crash
  adx trace
  adx crash --json
  adx crash -p com.example.myapp
  adx crash --lines 500`,
	RunE: func(cmd *cobra.Command, args []string) error {
		adbClient, err := GetADBClient()
		if err != nil {
			return err
		}

		targetDevices, err := ResolveTargetDevices(adbClient)
		if err != nil {
			return err
		}

		dev := targetDevices[0]

		pkgName := crashPackageFlag
		var pRootDir, pAppDir string
		if pkgName == "" {
			p, err := GetProject()
			if err == nil {
				pkgName = p.PackageName
				pRootDir = p.RootDir
				pAppDir = p.AppModuleDir
			}
		}

		if pkgName == "" {
			return fmt.Errorf("could not detect package name. Specify manually using --package or -p")
		}

		if !IsJSON() {
			ui.Step("Scanning logcat for crashes/exceptions in %s on %s...", pkgName, dev.Model)
		}

		logcatCmd := exec.Command(adbClient.AdbPath, "-s", dev.Serial, "logcat", "-d", "-t", fmt.Sprintf("%d", crashLinesFlag), "-v", "time")
		out, err := logcatCmd.Output()
		if err != nil {
			return fmt.Errorf("failed to retrieve logcat: %w", err)
		}

		crash := extractCrashInfo(string(out), pkgName, dev.Model, pRootDir, pAppDir)
		if crash == nil {
			if IsJSON() {
				fmt.Println(`{"status": "no_crash_found"}`)
				return nil
			}
			ui.Success("No recent crashes or ANRs detected for %s in last %d log lines! 🎉", pkgName, crashLinesFlag)
			return nil
		}

		if IsJSON() {
			data, _ := json.MarshalIndent(crash, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		// Human formatted crash report
		fmt.Println()
		redHeader := color.New(color.BgRed, color.FgHiWhite, color.Bold)
		redHeader.Printf(" 💥 CRASH DETECTED: %s \n", crash.CrashType)
		fmt.Printf("  %-16s %s\n", color.New(color.Bold).Sprint("Package:"), color.New(color.FgCyan).Sprint(crash.Package))
		fmt.Printf("  %-16s %s\n", color.New(color.Bold).Sprint("Exception:"), color.New(color.FgHiRed, color.Bold).Sprint(crash.ExceptionClass))
		if crash.Message != "" {
			fmt.Printf("  %-16s %s\n", color.New(color.Bold).Sprint("Message:"), color.New(color.FgYellow).Sprint(crash.Message))
		}
		if crash.RootCauseFile != "" {
			fmt.Printf("  %-16s %s (line %d)\n",
				color.New(color.Bold).Sprint("Root Cause:"),
				ui.ClickablePath(crash.RootCauseFile),
				crash.RootCauseLine,
			)
		}

		fmt.Println()
		color.New(color.Bold, color.Underline).Println("Stack Trace:")
		for _, frame := range crash.StackTrace {
			if strings.Contains(frame, pkgName) || (crash.RootCauseFile != "" && strings.Contains(frame, filepath.Base(crash.RootCauseFile))) {
				fmt.Printf("  %s\n", color.New(color.FgHiWhite, color.Bold).Sprint(frame))
			} else {
				fmt.Printf("  %s\n", color.New(color.Faint).Sprint(frame))
			}
		}
		fmt.Println()

		return nil
	},
}

func extractCrashInfo(logData string, pkgName string, deviceModel string, rootDir, appDir string) *CrashSummary {
	scanner := bufio.NewScanner(strings.NewReader(logData))
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	var crashBlocks [][]string
	var currentBlock []string
	inCrash := false

	for _, line := range lines {
		lower := strings.ToLower(line)
		isStart := strings.Contains(line, "FATAL EXCEPTION:") ||
			strings.Contains(line, "AndroidRuntime: FATAL EXCEPTION") ||
			strings.Contains(line, "ANR in "+pkgName) ||
			(strings.Contains(line, "DEBUG   :") && strings.Contains(line, "backtrace:")) ||
			(strings.Contains(line, "AndroidRuntime") && strings.Contains(lower, "exception"))

		if isStart {
			if inCrash && len(currentBlock) > 0 {
				crashBlocks = append(crashBlocks, currentBlock)
				currentBlock = nil
			}
			inCrash = true
		}

		if inCrash {
			currentBlock = append(currentBlock, line)
			// A typical exception block ends when logcat transitions to another tag or after blank lines
			if len(currentBlock) > 100 {
				crashBlocks = append(crashBlocks, currentBlock)
				currentBlock = nil
				inCrash = false
			}
		}
	}

	if inCrash && len(currentBlock) > 0 {
		crashBlocks = append(crashBlocks, currentBlock)
	}

	if len(crashBlocks) == 0 {
		// Try searching for any line with pkgName and Exception
		for i, line := range lines {
			if strings.Contains(line, pkgName) && (strings.Contains(line, "Exception:") || strings.Contains(line, "Error:")) {
				start := i
				if start > 2 {
					start -= 2
				}
				end := i + 35
				if end > len(lines) {
					end = len(lines)
				}
				crashBlocks = append(crashBlocks, lines[start:end])
				break
			}
		}
	}

	if len(crashBlocks) == 0 {
		return nil
	}

	// Pick the latest crash block
	lastBlock := crashBlocks[len(crashBlocks)-1]
	summary := &CrashSummary{
		Device:         deviceModel,
		Package:        pkgName,
		CrashType:      "FATAL EXCEPTION",
		ExceptionClass: "Exception",
		RawLog:         strings.Join(lastBlock, "\n"),
	}

	reException := regexp.MustCompile(`(?:Exception|Error|FATAL EXCEPTION.*):\s*([a-zA-Z0-9_.]+(?:Exception|Error))\s*(?::\s*(.*))?`)
	reCausedBy := regexp.MustCompile(`Caused by:\s*([a-zA-Z0-9_.]+(?:Exception|Error))\s*(?::\s*(.*))?`)

	for _, l := range lastBlock {
		clean := l
		if idx := strings.Index(l, "): "); idx != -1 {
			clean = l[idx+3:]
		} else if idx := strings.Index(l, "):"); idx != -1 {
			clean = l[idx+2:]
		}

		if strings.Contains(l, "ANR in") {
			summary.CrashType = "ANR (Application Not Responding)"
		}

		if m := reCausedBy.FindStringSubmatch(clean); len(m) > 1 {
			summary.ExceptionClass = m[1]
			if len(m) > 2 {
				summary.Message = m[2]
			}
		} else if m := reException.FindStringSubmatch(clean); len(m) > 1 && summary.ExceptionClass == "Exception" {
			summary.ExceptionClass = m[1]
			if len(m) > 2 {
				summary.Message = m[2]
			}
		}

		if strings.Contains(clean, "at ") || strings.Contains(clean, "Caused by:") || strings.HasPrefix(strings.TrimSpace(clean), "#") {
			summary.StackTrace = append(summary.StackTrace, strings.TrimSpace(clean))
		}
	}

	// Attempt to locate root cause source file & line
	reFileLine := regexp.MustCompile(`\(([a-zA-Z0-9_]+\.(?:kt|java)):([0-9]+)\)`)
	for _, frame := range summary.StackTrace {
		if m := reFileLine.FindStringSubmatch(frame); len(m) > 2 {
			fileName := m[1]
			var lineNum int
			fmt.Sscanf(m[2], "%d", &lineNum)

			localPath := findLocalSourceFile(fileName, rootDir, appDir)
			if localPath != "" {
				summary.RootCauseFile = localPath
				summary.RootCauseLine = lineNum
				break
			}
		}
	}

	return summary
}

func findLocalSourceFile(fileName string, rootDir, appDir string) string {
	dirs := []string{appDir, rootDir}
	for _, d := range dirs {
		if d == "" {
			continue
		}
		var found string
		_ = filepath.Walk(d, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() && info.Name() == fileName {
				found = path
				return filepath.SkipDir
			}
			return nil
		})
		if found != "" {
			return found
		}
	}
	return ""
}

func init() {
	crashCmd.Flags().StringVarP(&crashPackageFlag, "package", "p", "", "Explicit application package name to inspect")
	crashCmd.Flags().IntVar(&crashLinesFlag, "lines", 2000, "Number of recent logcat lines to analyze")
	rootCmd.AddCommand(crashCmd)
}
