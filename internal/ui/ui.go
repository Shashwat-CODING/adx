package ui

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
)

var (
	orange     = color.RGB(255, 140, 0).Add(color.Bold)
	orangeSoft = color.RGB(255, 165, 0)
	green      = color.New(color.FgGreen, color.Bold)
	yellow     = color.RGB(255, 180, 0).Add(color.Bold)
	red        = color.New(color.FgRed, color.Bold)
	bold       = color.New(color.Bold)
	dim        = color.New(color.Faint)
)

// Orange returns the primary orange theme color
func Orange() *color.Color {
	return orange
}

// OrangeSoft returns soft orange color
func OrangeSoft() *color.Color {
	return orangeSoft
}

var bannerOnce sync.Once

// PrintBanner prints a medium-sized stylized adx banner in orange
func PrintBanner() {
	banner := `
           _       
  __ _  __| |_  __ 
 / _  |/ _  | \ \/ /
| (_| | (_| |  ><  
 \__,_|\__,_|/_/\_\
`
	orange.Print(banner)
	orangeSoft.Println(" android developer experience cli")
	fmt.Println()
}

// PrintBannerOnce ensures the banner is printed exactly once
func PrintBannerOnce() {
	bannerOnce.Do(func() {
		PrintBanner()
	})
}

// Info prints an informational message
func Info(format string, a ...interface{}) {
	fmt.Printf("%s %s\n", orange.Sprint("ℹ"), fmt.Sprintf(format, a...))
}

// Success prints a success message
func Success(format string, a ...interface{}) {
	fmt.Printf("%s %s\n", green.Sprint("✔"), fmt.Sprintf(format, a...))
}

// Warn prints a warning message
func Warn(format string, a ...interface{}) {
	fmt.Printf("%s %s\n", yellow.Sprint("⚠"), fmt.Sprintf(format, a...))
}

// Error prints an error message
func Error(format string, a ...interface{}) {
	fmt.Printf("%s %s\n", red.Sprint("✖"), fmt.Sprintf(format, a...))
}

// ClickablePath formats a file path into an IDE-clickable file:// link in orange
func ClickablePath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	uri := "file://" + abs
	return color.RGB(255, 150, 0).Add(color.Underline, color.Bold).Sprint(uri)
}

// PrintErrorBlock prints an error banner and trace in high-visibility bold red
func PrintErrorBlock(title string, lines []string) {
	fmt.Println()
	red.Println("━━━━━━━━━━━━━━━━━━━━ ERROR ━━━━━━━━━━━━━━━━━━━━")
	if title != "" {
		red.Println(title)
	}
	for _, l := range lines {
		red.Println(l)
	}
	red.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
}

// Spinner provides an animated terminal loading indicator
type Spinner struct {
	message string
	mu      sync.Mutex
	stopCh  chan struct{}
	doneCh  chan struct{}
}

// NewSpinner creates and starts an animated spinner
func NewSpinner(initialMessage string) *Spinner {
	s := &Spinner{
		message: initialMessage,
		stopCh:  make(chan struct{}),
		doneCh:  make(chan struct{}),
	}
	go s.run()
	return s
}

func (s *Spinner) run() {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	idx := 0
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()
	defer close(s.doneCh)

	for {
		select {
		case <-s.stopCh:
			// Clear line
			fmt.Print("\r\033[K")
			return
		case <-ticker.C:
			s.mu.Lock()
			msg := s.message
			s.mu.Unlock()

			frame := orange.Sprint(frames[idx%len(frames)])
			fmt.Printf("\r\033[K%s %s", frame, msg)
			idx++
		}
	}
}

// Update updates the active spinner text
func (s *Spinner) Update(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.message = msg
}

// StopSuccess stops the spinner and prints a green checkmark
func (s *Spinner) StopSuccess(format string, a ...interface{}) {
	close(s.stopCh)
	<-s.doneCh
	Success(format, a...)
}

// StopFail stops the spinner and prints a red cross
func (s *Spinner) StopFail(format string, a ...interface{}) {
	close(s.stopCh)
	<-s.doneCh
	Error(format, a...)
}

// Step prints a major workflow step
func Step(format string, a ...interface{}) {
	fmt.Printf("\n%s %s\n", orange.Sprint("➜"), bold.Sprintf(format, a...))
}

// Dim prints faint text
func Dim(format string, a ...interface{}) {
	fmt.Println(dim.Sprintf(format, a...))
}

// DeviceOption represents an option in the device selection menu
type DeviceOption struct {
	Serial      string
	Model       string
	Description string
}

// SelectDeviceInteractive prompts the user to select one device or all devices
func SelectDeviceInteractive(devices []DeviceOption) ([]DeviceOption, error) {
	if len(devices) == 0 {
		return nil, fmt.Errorf("no connected devices found")
	}
	if len(devices) == 1 {
		Info("Using connected device: %s (%s)", bold.Sprint(devices[0].Model), devices[0].Serial)
		return devices, nil
	}

	fmt.Println()
	orange.Println("Multiple devices detected. Please choose a target:")
	for i, d := range devices {
		fmt.Printf("  [%s] %s %s (%s)\n",
			orange.Sprintf("%d", i+1),
			orangeSoft.Sprint("📱"),
			bold.Sprint(d.Model),
			dim.Sprint(d.Serial),
		)
	}
	fmt.Printf("  [%s] %s\n", orange.Sprint("a"), bold.Sprint("All connected devices"))

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\nEnter choice [1-" + strconv.Itoa(len(devices)) + " or a]: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		choice := strings.TrimSpace(input)
		if strings.ToLower(choice) == "a" || strings.ToLower(choice) == "all" {
			return devices, nil
		}

		num, err := strconv.Atoi(choice)
		if err == nil && num >= 1 && num <= len(devices) {
			return []DeviceOption{devices[num-1]}, nil
		}

		Warn("Invalid option: %s. Please enter a valid number or 'a'.", choice)
	}
}
