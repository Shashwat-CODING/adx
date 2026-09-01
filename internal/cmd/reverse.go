package cmd

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/Shashwat-CODING/adx/internal/ui"
	"github.com/spf13/cobra"
)

type PortForwardResult struct {
	Device     string `json:"device"`
	Serial     string `json:"serial"`
	DevicePort string `json:"device_port"`
	HostPort   string `json:"host_port"`
	Type       string `json:"type"`
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
}

var reverseCmd = &cobra.Command{
	Use:     "reverse [device_port] [host_port]",
	Aliases: []string{"rev"},
	Short:   "Reverse port forward to access host localhost from Android",
	Long: `Runs 'adb reverse tcp:<device_port> tcp:<host_port>' so your Android app can reach your local development backend (e.g. localhost:8080 or localhost:3000).
Defaults to 8080 8080 if not specified.

Examples:
  adx reverse 8080
  adx reverse 8080 8080
  adx reverse 3000 8080`,
	RunE: func(cmd *cobra.Command, args []string) error {
		devicePort := "8080"
		hostPort := "8080"

		if len(args) == 1 {
			devicePort = args[0]
			hostPort = args[0]
		} else if len(args) >= 2 {
			devicePort = args[0]
			hostPort = args[1]
		}

		adbClient, err := GetADBClient()
		if err != nil {
			return err
		}

		targetDevices, err := ResolveTargetDevices(adbClient)
		if err != nil {
			return err
		}

		deviceProto := fmt.Sprintf("tcp:%s", devicePort)
		hostProto := fmt.Sprintf("tcp:%s", hostPort)
		var results []PortForwardResult

		for _, dev := range targetDevices {
			c := exec.Command(adbClient.AdbPath, "-s", dev.Serial, "reverse", deviceProto, hostProto)
			res := PortForwardResult{
				Device:     dev.Model,
				Serial:     dev.Serial,
				DevicePort: devicePort,
				HostPort:   hostPort,
				Type:       "reverse",
			}
			if out, err := c.CombinedOutput(); err != nil {
				res.Status = "failed"
				res.Error = fmt.Sprintf("%s (%v)", strings.TrimSpace(string(out)), err)
				if !IsJSON() {
					ui.Error("Failed to reverse port %s -> %s on %s: %s", devicePort, hostPort, dev.Model, res.Error)
				}
			} else {
				res.Status = "success"
				if !IsJSON() {
					ui.Success("Reversed %s on %s -> app can reach host http://localhost:%s", deviceProto, dev.Model, hostPort)
				}
			}
			results = append(results, res)
		}

		if IsJSON() {
			data, _ := json.MarshalIndent(results, "", "  ")
			fmt.Println(string(data))
		}

		return nil
	},
}

var portForwardCmd = &cobra.Command{
	Use:     "port-forward [host_port] [device_port]",
	Aliases: []string{"forward"},
	Short:   "Forward socket connections from host to Android device",
	Long: `Runs 'adb forward tcp:<host_port> tcp:<device_port>' to forward host ports into device ports.
Defaults to 8080 8080 if not specified.

Examples:
  adx port-forward 8080 8080
  adx forward 9222 9222`,
	RunE: func(cmd *cobra.Command, args []string) error {
		hostPort := "8080"
		devicePort := "8080"

		if len(args) == 1 {
			hostPort = args[0]
			devicePort = args[0]
		} else if len(args) >= 2 {
			hostPort = args[0]
			devicePort = args[1]
		}

		adbClient, err := GetADBClient()
		if err != nil {
			return err
		}

		targetDevices, err := ResolveTargetDevices(adbClient)
		if err != nil {
			return err
		}

		hostProto := fmt.Sprintf("tcp:%s", hostPort)
		deviceProto := fmt.Sprintf("tcp:%s", devicePort)
		var results []PortForwardResult

		for _, dev := range targetDevices {
			c := exec.Command(adbClient.AdbPath, "-s", dev.Serial, "forward", hostProto, deviceProto)
			res := PortForwardResult{
				Device:     dev.Model,
				Serial:     dev.Serial,
				DevicePort: devicePort,
				HostPort:   hostPort,
				Type:       "forward",
			}
			if out, err := c.CombinedOutput(); err != nil {
				res.Status = "failed"
				res.Error = fmt.Sprintf("%s (%v)", strings.TrimSpace(string(out)), err)
				if !IsJSON() {
					ui.Error("Failed to forward port %s -> %s on %s: %s", hostPort, devicePort, dev.Model, res.Error)
				}
			} else {
				res.Status = "success"
				if !IsJSON() {
					ui.Success("Forwarded host %s -> %s on %s", hostProto, deviceProto, dev.Model)
				}
			}
			results = append(results, res)
		}

		if IsJSON() {
			data, _ := json.MarshalIndent(results, "", "  ")
			fmt.Println(string(data))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(reverseCmd)
	rootCmd.AddCommand(portForwardCmd)
}
