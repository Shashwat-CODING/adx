package adb

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Shashwat-CODING/adx/internal/ui"
)

// Device represents an attached ADB target
type Device struct {
	Serial  string
	State   string
	Model   string
	Product string
}

// Client wraps ADB CLI execution
type Client struct {
	AdbPath string
}

// NewClient resolves ADB binary path from PATH or Android SDK locations
func NewClient() (*Client, error) {
	// 1. Check system PATH
	if p, err := exec.LookPath("adb"); err == nil {
		return &Client{AdbPath: p}, nil
	}

	// 2. Candidate SDK locations
	candidates := []string{}
	if home := os.Getenv("ANDROID_HOME"); home != "" {
		candidates = append(candidates, filepath.Join(home, "platform-tools", "adb"))
	}
	if sdkRoot := os.Getenv("ANDROID_SDK_ROOT"); sdkRoot != "" {
		candidates = append(candidates, filepath.Join(sdkRoot, "platform-tools", "adb"))
	}

	userHome, err := os.UserHomeDir()
	if err == nil {
		candidates = append(candidates,
			filepath.Join(userHome, "Library", "Android", "sdk", "platform-tools", "adb"),
			filepath.Join(userHome, "Android", "Sdk", "platform-tools", "adb"),
			filepath.Join(userHome, "AppData", "Local", "Android", "Sdk", "platform-tools", "adb.exe"),
		)
	}

	for _, cand := range candidates {
		if runtime.GOOS == "windows" && !strings.HasSuffix(cand, ".exe") {
			cand += ".exe"
		}
		if _, err := os.Stat(cand); err == nil {
			return &Client{AdbPath: cand}, nil
		}
	}

	return nil, fmt.Errorf("adb not found in PATH or Android SDK platform-tools. Please install Android Platform Tools or set ANDROID_HOME")
}

// GetDevices returns all currently connected ADB devices
func (c *Client) GetDevices() ([]Device, error) {
	cmd := exec.Command(c.AdbPath, "devices", "-l")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run 'adb devices': %w", err)
	}

	var devices []Device
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "List of devices attached") || strings.HasPrefix(line, "* daemon") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		serial := fields[0]
		state := fields[1]
		model := serial
		product := ""

		for _, f := range fields[2:] {
			if strings.HasPrefix(f, "model:") {
				model = strings.TrimPrefix(f, "model:")
				model = strings.ReplaceAll(model, "_", " ")
			} else if strings.HasPrefix(f, "product:") {
				product = strings.TrimPrefix(f, "product:")
			}
		}

		devices = append(devices, Device{
			Serial:  serial,
			State:   state,
			Model:   model,
			Product: product,
		})
	}

	return devices, nil
}

// InstallAPK installs the specified APK file onto a device
func (c *Client) InstallAPK(serial string, apkPath string) error {
	cmd := exec.Command(c.AdbPath, "-s", serial, "install", "-r", "-t", "-d", apkPath)
	out, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "Success") {
		return fmt.Errorf("install failed on %s: %s (%v)", serial, strings.TrimSpace(string(out)), err)
	}
	return nil
}

// InstallParallel installs the APK across multiple devices concurrently
func (c *Client) InstallParallel(devices []Device, apkPath string) []error {
	var wg sync.WaitGroup
	errs := make([]error, len(devices))

	for i, d := range devices {
		wg.Add(1)
		go func(idx int, dev Device) {
			defer wg.Done()
			ui.Info("Installing on %s (%s)...", dev.Model, dev.Serial)
			if err := c.InstallAPK(dev.Serial, apkPath); err != nil {
				errs[idx] = err
			} else {
				ui.Success("Installed successfully on %s (%s)", dev.Model, dev.Serial)
			}
		}(i, d)
	}

	wg.Wait()
	return errs
}

// LaunchApp launches the application using its launcher activity or monkey fallback
func (c *Client) LaunchApp(serial string, packageName string, launcherActivity string) error {
	if packageName == "" {
		return fmt.Errorf("cannot launch app: packageName is unknown")
	}

	var launched bool

	// 1. Try explicit launcherActivity if provided
	if launcherActivity != "" {
		target := launcherActivity
		if !strings.Contains(target, "/") {
			target = packageName + "/" + launcherActivity
		}
		cmd := exec.Command(c.AdbPath, "-s", serial, "shell", "am", "start", "-n", target)
		out, err := cmd.CombinedOutput()
		outStr := string(out)
		// Note: am start exits with code 0 even if "Error: Activity class ... does not exist"
		if err == nil && !strings.Contains(outStr, "Error:") && !strings.Contains(outStr, "does not exist") {
			launched = true
		}
	}

	if launched {
		return nil
	}

	// 2. Fallback to monkey which queries the device's PackageManager for the exact launcher intent
	monkeyCmd := exec.Command(c.AdbPath, "-s", serial, "shell", "monkey", "-p", packageName, "-c", "android.intent.category.LAUNCHER", "1")
	out, err := monkeyCmd.CombinedOutput()
	outStr := string(out)
	if err == nil && (strings.Contains(outStr, "Events injected: 1") || strings.Contains(outStr, "Events injected")) {
		return nil
	}

	// 3. Fallback to querying resolve-activity on device
	resolveCmd := exec.Command(c.AdbPath, "-s", serial, "shell", "cmd", "package", "resolve-activity", "--brief", packageName)
	if resOut, resErr := resolveCmd.Output(); resErr == nil {
		lines := strings.Split(strings.TrimSpace(string(resOut)), "\n")
		for _, l := range lines {
			l = strings.TrimSpace(l)
			if strings.Contains(l, "/") && !strings.HasPrefix(l, "priority=") {
				startCmd := exec.Command(c.AdbPath, "-s", serial, "shell", "am", "start", "-n", l)
				if sOut, sErr := startCmd.CombinedOutput(); sErr == nil && !strings.Contains(string(sOut), "Error:") {
					return nil
				}
			}
		}
	}

	return fmt.Errorf("failed to launch %s on device (%s)", packageName, strings.TrimSpace(outStr))
}

// ForceStop force-stops an application
func (c *Client) ForceStop(serial string, packageName string) error {
	cmd := exec.Command(c.AdbPath, "-s", serial, "shell", "am", "force-stop", packageName)
	return cmd.Run()
}

// GetPID returns the process ID of the app if running
func (c *Client) GetPID(serial string, packageName string) (string, error) {
	cmd := exec.Command(c.AdbPath, "-s", serial, "shell", "pidof", "-s", packageName)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// StreamLogs streams logcat filtered for the current application package
func (c *Client) StreamLogs(ctx context.Context, serial string, packageName string, clearFirst bool) error {
	if clearFirst {
		_ = exec.Command(c.AdbPath, "-s", serial, "logcat", "-c").Run()
		ui.Info("Cleared previous logcat logs")
	}

	ui.Info("Streaming logs for [%s] on %s... (Ctrl+C to stop)", packageName, serial)

	// Modern Android devices support filtering by package or PID
	// Let's run logcat and filter dynamically for the package name or its active PID
	cmd := exec.CommandContext(ctx, c.AdbPath, "-s", serial, "logcat", "-v", "color")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(stdout)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var currentPID string
	var pidMutex sync.Mutex

	// Goroutine to periodically update app PID
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				pid, err := c.GetPID(serial, packageName)
				if err == nil && pid != "" {
					pidMutex.Lock()
					currentPID = pid
					pidMutex.Unlock()
				}
				time.Sleep(2 * time.Second)
			}
		}
	}()

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil
		default:
			line := scanner.Text()
			pidMutex.Lock()
			pid := currentPID
			pidMutex.Unlock()

			// Match by package name or PID
			if strings.Contains(line, packageName) || (pid != "" && (strings.Contains(line, " "+pid+" ") || strings.Contains(line, ":"+pid+":") || strings.Contains(line, "("+pid+")"))) {
				fmt.Println(line)
			}
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return err
	}

	return cmd.Wait()
}

// TakeScreenshot captures a PNG screenshot from the device and saves it locally
func (c *Client) TakeScreenshot(serial, destPath string) error {
	cmd := exec.Command(c.AdbPath, "-s", serial, "exec-out", "screencap", "-p")
	outFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	cmd.Stdout = outFile
	return cmd.Run()
}

// RecordScreen records the device display until context cancelled, then pulls the MP4
func (c *Client) RecordScreen(ctx context.Context, serial, destPath string) error {
	remotePath := "/sdcard/adx_screenrecord.mp4"
	cmd := exec.CommandContext(ctx, c.AdbPath, "-s", serial, "shell", "screenrecord", remotePath)
	_ = cmd.Run()

	time.Sleep(1 * time.Second)
	pullCmd := exec.Command(c.AdbPath, "-s", serial, "pull", remotePath, destPath)
	if err := pullCmd.Run(); err != nil {
		return fmt.Errorf("failed to retrieve recording from device: %w", err)
	}
	_ = exec.Command(c.AdbPath, "-s", serial, "shell", "rm", remotePath).Run()
	return nil
}

// OpenDeepLink opens a URI deep link on the device
func (c *Client) OpenDeepLink(serial, uri string) error {
	cmd := exec.Command(c.AdbPath, "-s", serial, "shell", "am", "start", "-a", "android.intent.action.VIEW", "-d", uri)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s (%w)", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// Uninstall uninstalls an application package from the device
func (c *Client) Uninstall(serial, packageName string) error {
	cmd := exec.Command(c.AdbPath, "-s", serial, "uninstall", packageName)
	out, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "Success") {
		return fmt.Errorf("%s (%v)", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// FindEmulatorBinary locates the Android SDK emulator binary
func FindEmulatorBinary() (string, error) {
	if p, err := exec.LookPath("emulator"); err == nil {
		return p, nil
	}

	var candidates []string
	if home := os.Getenv("ANDROID_HOME"); home != "" {
		candidates = append(candidates, filepath.Join(home, "emulator", "emulator"))
	}
	if sdkRoot := os.Getenv("ANDROID_SDK_ROOT"); sdkRoot != "" {
		candidates = append(candidates, filepath.Join(sdkRoot, "emulator", "emulator"))
	}

	userHome, err := os.UserHomeDir()
	if err == nil {
		candidates = append(candidates,
			filepath.Join(userHome, "Library", "Android", "sdk", "emulator", "emulator"),
			filepath.Join(userHome, "Android", "Sdk", "emulator", "emulator"),
			filepath.Join(userHome, "AppData", "Local", "Android", "Sdk", "emulator", "emulator.exe"),
		)
	}

	for _, cand := range candidates {
		if runtime.GOOS == "windows" && !strings.HasSuffix(cand, ".exe") {
			cand += ".exe"
		}
		if _, err := os.Stat(cand); err == nil {
			return cand, nil
		}
	}

	return "", fmt.Errorf("emulator binary not found in PATH or Android SDK emulator directory")
}

// ListAVDs returns a list of installed Android Virtual Devices
func ListAVDs() ([]string, error) {
	emuPath, err := FindEmulatorBinary()
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(emuPath, "-list-avds")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list AVDs: %w", err)
	}

	var avds []string
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			avds = append(avds, line)
		}
	}
	return avds, nil
}

// StartEmulator launches an AVD in the background
func StartEmulator(avdName string) error {
	emuPath, err := FindEmulatorBinary()
	if err != nil {
		return err
	}

	cmd := exec.Command(emuPath, "-avd", avdName)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to launch emulator '%s': %w", avdName, err)
	}
	_ = cmd.Process.Release()
	return nil
}
