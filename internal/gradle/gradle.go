package gradle

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/Shashwat-CODING/adx/internal/project"
	"github.com/Shashwat-CODING/adx/internal/ui"
	"github.com/fatih/color"
)

var (
	taskColor    = color.RGB(255, 145, 0)
	successColor = color.New(color.FgGreen, color.Bold)
	errorColor   = color.New(color.FgRed, color.Bold)
)

// Runner executes Gradle tasks for an Android project
type Runner struct {
	Project *project.Project
	Verbose bool
}

// NewRunner creates a new Gradle task runner
func NewRunner(p *project.Project, verbose bool) *Runner {
	return &Runner{Project: p, Verbose: verbose}
}

// Run executes specified Gradle tasks with live streaming or spinner
func (r *Runner) Run(tasks []string, extraArgs ...string) error {
	args := append(tasks, extraArgs...)
	cmd := exec.Command(r.Project.GradlewPath, args...)
	cmd.Dir = r.Project.RootDir
	cmd.Env = os.Environ()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		ui.Error("Failed to start Gradle (%s): %v", r.Project.GradlewPath, err)
		return err
	}

	if r.Verbose {
		// Full verbose mode: stream all output in real time
		go streamOutputVerbose(stdout, false)
		go streamOutputVerbose(stderr, true)

		if err := cmd.Wait(); err != nil {
			errorColor.Printf("\n✖ Gradle command failed: %v\n", err)
			return err
		}
		return nil
	}

	// Normal mode: clean animated spinner with live active task updates
	taskSummary := strings.Join(tasks, " ")
	spinner := ui.NewSpinner(fmt.Sprintf("Executing %s...", taskSummary))

	var outputMu sync.Mutex
	var recentLines []string
	var errorLines []string

	collectLines := func(reader io.Reader, isErr bool) {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Text()
			trimmed := strings.TrimSpace(line)

			outputMu.Lock()
			if len(recentLines) > 40 {
				recentLines = recentLines[1:]
			}
			recentLines = append(recentLines, line)

			lower := strings.ToLower(trimmed)
			if isErr || strings.Contains(lower, "error") || strings.Contains(lower, "failed") || strings.Contains(lower, "exception") || strings.HasPrefix(trimmed, "e:") || strings.HasPrefix(trimmed, "w:") {
				errorLines = append(errorLines, line)
			}

			// Update spinner with current task
			if strings.HasPrefix(trimmed, "> Task :") {
				spinner.Update(trimmed)
			}
			outputMu.Unlock()
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		collectLines(stdout, false)
	}()
	go func() {
		defer wg.Done()
		collectLines(stderr, true)
	}()

	cmdErr := cmd.Wait()
	wg.Wait()

	if cmdErr != nil {
		spinner.StopFail("Build failed!")

		outputMu.Lock()
		displayLines := errorLines
		if len(displayLines) == 0 {
			displayLines = recentLines
		}
		outputMu.Unlock()

		ui.PrintErrorBlock(fmt.Sprintf("Task execution failed: %s", taskSummary), displayLines)
		return cmdErr
	}

	spinner.StopSuccess("Build completed successfully")
	return nil
}

func streamOutputVerbose(r io.Reader, isErr bool) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "> Task :") {
			taskColor.Println(trimmed)
		} else if strings.HasPrefix(trimmed, "BUILD SUCCESSFUL") {
			successColor.Println(trimmed)
		} else if strings.HasPrefix(trimmed, "BUILD FAILED") || strings.Contains(strings.ToLower(trimmed), "error:") || strings.HasPrefix(trimmed, "e:") {
			errorColor.Println(trimmed)
		} else if isErr {
			errorColor.Fprintln(os.Stderr, line)
		} else {
			fmt.Println(line)
		}
	}
}

// BuildVariant runs assemble for the specified variant (e.g. "debug" or "release")
func (r *Runner) BuildVariant(variant string, extraArgs ...string) error {
	capitalized := strings.ToUpper(variant[:1]) + strings.ToLower(variant[1:])
	task := fmt.Sprintf("assemble%s", capitalized)
	ui.Step("Running Gradle task: %s", task)
	return r.Run([]string{task}, extraArgs...)
}

// Clean runs the clean task
func (r *Runner) Clean(extraArgs ...string) error {
	ui.Step("Running Gradle clean...")
	return r.Run([]string{"clean"}, extraArgs...)
}
