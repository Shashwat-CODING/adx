package cmd

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shashwat-CODING/adx/internal/gradle"
	"github.com/Shashwat-CODING/adx/internal/ui"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	testFailedOnlyFlag bool
	testSummaryFlag    bool
)

type TestSuiteXML struct {
	XMLName   xml.Name      `xml:"testsuite"`
	Name      string        `xml:"name,attr"`
	Tests     int           `xml:"tests,attr"`
	Skipped   int           `xml:"skipped,attr"`
	Failures  int           `xml:"failures,attr"`
	Errors    int           `xml:"errors,attr"`
	Time      float64       `xml:"time,attr"`
	TestCases []TestCaseXML `xml:"testcase"`
}

type TestCaseXML struct {
	Name      string       `xml:"name,attr"`
	ClassName string       `xml:"classname,attr"`
	Time      float64      `xml:"time,attr"`
	Failure   *TestFailXML `xml:"failure"`
	Error     *TestFailXML `xml:"error"`
}

type TestFailXML struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Content string `xml:",chardata"`
}

type TestResultJSON struct {
	TotalTests    int             `json:"total_tests"`
	PassedTests   int             `json:"passed_tests"`
	FailedTests   int             `json:"failed_tests"`
	SkippedTests  int             `json:"skipped_tests"`
	DurationSec   float64         `json:"duration_sec"`
	ReportPath    string          `json:"report_path,omitempty"`
	FailingTests  []FailingTestUI `json:"failing_tests,omitempty"`
	PassingTests  []PassingTestUI `json:"passing_tests,omitempty"`
}

type FailingTestUI struct {
	Class      string `json:"class"`
	Method     string `json:"method"`
	Message    string `json:"message"`
	Type       string `json:"type,omitempty"`
	StackTrace string `json:"stack_trace,omitempty"`
}

type PassingTestUI struct {
	Class  string  `json:"class"`
	Method string  `json:"method"`
	Time   float64 `json:"time"`
}

var testCmd = &cobra.Command{
	Use:   "test [variant]",
	Short: "Run unit tests for the Android project",
	Long: `Runs the Gradle unit test task for the specified variant (defaults to debug).
Includes smart test summary output, failure isolation, and machine-readable JSON.

Examples:
  adx test
  adx test --failed-only
  adx test --summary
  adx test --json
  adx test release -v`,
	RunE: func(cmd *cobra.Command, args []string) error {
		variant := "debug"
		if len(args) > 0 {
			variant = args[0]
		}

		p, err := GetProject()
		if err != nil {
			return err
		}

		capVariant := strings.ToUpper(variant[:1]) + strings.ToLower(variant[1:])
		taskName := fmt.Sprintf("test%sUnitTest", capVariant)

		if !IsJSON() {
			ui.Step("Running unit tests (%s)...", taskName)
		}

		runner := gradle.NewRunner(p, IsVerbose())
		gradleErr := runner.Run([]string{taskName})

		// Parse XML test results from candidate folders
		testResultsDir := filepath.Join(p.AppModuleDir, "build", "test-results", taskName)
		if _, err := os.Stat(testResultsDir); err != nil {
			testResultsDir = filepath.Join(p.RootDir, "build", "test-results", taskName)
		}

		result := parseTestResults(testResultsDir)
		reportPath := filepath.Join(p.AppModuleDir, "build", "reports", "tests", taskName, "index.html")
		if _, err := os.Stat(reportPath); err == nil {
			result.ReportPath = reportPath
		}

		if IsJSON() {
			if testFailedOnlyFlag {
				result.PassingTests = nil
			}
			data, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(data))
			if result.FailedTests > 0 || gradleErr != nil {
				return fmt.Errorf("tests failed")
			}
			return nil
		}

		// Human formatting
		fmt.Println()
		if result.TotalTests == 0 {
			if gradleErr != nil {
				return gradleErr
			}
			ui.Info("No unit tests were found or executed.")
			return nil
		}

		if result.FailedTests > 0 {
			ui.Error("Tests failed: %d failed, %d passed, %d total (%.2fs)",
				result.FailedTests, result.PassedTests, result.TotalTests, result.DurationSec)

			fmt.Println()
			redBold := color.New(color.FgRed, color.Bold)
			redBold.Println("Failed Tests Summary:")
			for i, ft := range result.FailingTests {
				fmt.Printf("  [%d] %s > %s\n", i+1, color.New(color.Bold).Sprint(ft.Class), color.New(color.FgHiRed).Sprint(ft.Method))
				if ft.Message != "" {
					lines := strings.Split(ft.Message, "\n")
					for _, l := range lines {
						if strings.TrimSpace(l) != "" {
							fmt.Printf("      %s\n", color.New(color.FgYellow).Sprint(strings.TrimSpace(l)))
						}
					}
				}
			}
		} else {
			ui.Success("All unit tests passed! (%d passed, 0 failed in %.2fs)", result.PassedTests, result.DurationSec)
		}

		if !testFailedOnlyFlag && !testSummaryFlag && len(result.PassingTests) > 0 && result.FailedTests == 0 {
			ui.Dim("Executed %d test methods successfully.", len(result.PassingTests))
		}

		if result.ReportPath != "" {
			fmt.Println()
			ui.Info("Full HTML test report: %s", ui.ClickablePath(result.ReportPath))
		}

		if result.FailedTests > 0 {
			return fmt.Errorf("unit test suite failed with %d failures", result.FailedTests)
		}

		return gradleErr
	},
}

func parseTestResults(resultsDir string) TestResultJSON {
	var result TestResultJSON

	files, err := os.ReadDir(resultsDir)
	if err != nil {
		return result
	}

	for _, f := range files {
		if !f.IsDir() && strings.HasPrefix(f.Name(), "TEST-") && strings.HasSuffix(f.Name(), ".xml") {
			xmlPath := filepath.Join(resultsDir, f.Name())
			data, err := os.ReadFile(xmlPath)
			if err != nil {
				continue
			}

			var suite TestSuiteXML
			if err := xml.Unmarshal(data, &suite); err != nil {
				continue
			}

			result.TotalTests += suite.Tests
			result.SkippedTests += suite.Skipped
			result.DurationSec += suite.Time

			for _, tc := range suite.TestCases {
				if tc.Failure != nil || tc.Error != nil {
					result.FailedTests++
					failObj := tc.Failure
					if failObj == nil {
						failObj = tc.Error
					}
					result.FailingTests = append(result.FailingTests, FailingTestUI{
						Class:      tc.ClassName,
						Method:     tc.Name,
						Message:    failObj.Message,
						Type:       failObj.Type,
						StackTrace: strings.TrimSpace(failObj.Content),
					})
				} else {
					result.PassedTests++
					result.PassingTests = append(result.PassingTests, PassingTestUI{
						Class:  tc.ClassName,
						Method: tc.Name,
						Time:   tc.Time,
					})
				}
			}
		}
	}

	return result
}

func init() {
	testCmd.Flags().BoolVar(&testFailedOnlyFlag, "failed-only", false, "Output only failing assertions with expected vs actual values")
	testCmd.Flags().BoolVar(&testSummaryFlag, "summary", false, "Output compact summary of test suite results")
	rootCmd.AddCommand(testCmd)
}
