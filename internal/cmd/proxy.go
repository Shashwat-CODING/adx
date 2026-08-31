package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/Shashwat-CODING/adx/internal/ui"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Configure device HTTP network proxy (Proxyman, Charles, mitmproxy)",
	Long: `Inspect, set, or clear the global HTTP proxy on connected Android device(s)
for inspecting network traffic through Charles, Proxyman, mitmproxy, or Fiddler.

Examples:
  adx proxy                     # View current active proxy status
  adx proxy set 192.168.1.5:8888 # Route device traffic to proxy
  adx proxy clear               # Remove proxy and restore direct network`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return showProxyStatus()
	},
}

var proxySetCmd = &cobra.Command{
	Use:   "set <host:port>",
	Short: "Set global HTTP proxy on device",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		proxyTarget := strings.TrimSpace(args[0])
		if !strings.Contains(proxyTarget, ":") {
			return fmt.Errorf("invalid proxy format. Expected host:port (e.g. 192.168.1.5:8888)")
		}

		adbClient, err := GetADBClient()
		if err != nil {
			return err
		}

		targetDevices, err := ResolveTargetDevices(adbClient)
		if err != nil {
			return err
		}

		parts := strings.Split(proxyTarget, ":")
		host := parts[0]
		port := parts[1]

		for _, dev := range targetDevices {
			ui.Info("Configuring proxy %s on %s...", proxyTarget, dev.Model)
			_ = exec.Command(adbClient.AdbPath, "-s", dev.Serial, "shell", "settings", "put", "global", "http_proxy", proxyTarget).Run()
			_ = exec.Command(adbClient.AdbPath, "-s", dev.Serial, "shell", "settings", "put", "global", "global_http_proxy_host", host).Run()
			_ = exec.Command(adbClient.AdbPath, "-s", dev.Serial, "shell", "settings", "put", "global", "global_http_proxy_port", port).Run()
			ui.Success("Proxy set to %s on %s", ui.Orange().Sprint(proxyTarget), dev.Model)
		}

		return nil
	},
}

var proxyClearCmd = &cobra.Command{
	Use:     "clear",
	Aliases: []string{"reset", "remove", "off"},
	Short:   "Clear device HTTP proxy and restore direct connection",
	RunE: func(cmd *cobra.Command, args []string) error {
		adbClient, err := GetADBClient()
		if err != nil {
			return err
		}

		targetDevices, err := ResolveTargetDevices(adbClient)
		if err != nil {
			return err
		}

		for _, dev := range targetDevices {
			ui.Info("Clearing proxy on %s...", dev.Model)
			_ = exec.Command(adbClient.AdbPath, "-s", dev.Serial, "shell", "settings", "put", "global", "http_proxy", ":0").Run()
			_ = exec.Command(adbClient.AdbPath, "-s", dev.Serial, "shell", "settings", "delete", "global", "http_proxy").Run()
			_ = exec.Command(adbClient.AdbPath, "-s", dev.Serial, "shell", "settings", "delete", "global", "global_http_proxy_host").Run()
			_ = exec.Command(adbClient.AdbPath, "-s", dev.Serial, "shell", "settings", "delete", "global", "global_http_proxy_port").Run()
			ui.Success("Proxy removed on %s (direct network connection restored)", dev.Model)
		}

		return nil
	},
}

var proxyStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current proxy configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		return showProxyStatus()
	},
}

func showProxyStatus() error {
	adbClient, err := GetADBClient()
	if err != nil {
		return err
	}

	devices, err := adbClient.GetDevices()
	if err != nil || len(devices) == 0 {
		return fmt.Errorf("no connected Android devices found")
	}

	ui.Step("Active Network Proxy Configuration:")
	for _, dev := range devices {
		out, err := exec.Command(adbClient.AdbPath, "-s", dev.Serial, "shell", "settings", "get", "global", "http_proxy").Output()
		proxyVal := strings.TrimSpace(string(out))
		if err != nil || proxyVal == "null" || proxyVal == ":0" || proxyVal == "" {
			fmt.Printf("  📱 %-20s %s\n", color.New(color.Bold).Sprint(dev.Model), color.New(color.FgGreen).Sprint("Direct Connection (No proxy)"))
		} else {
			fmt.Printf("  📱 %-20s %s\n", color.New(color.Bold).Sprint(dev.Model), ui.Orange().Sprintf("Proxy: %s", proxyVal))
		}
	}
	fmt.Println()
	return nil
}

func init() {
	proxyCmd.AddCommand(proxySetCmd)
	proxyCmd.AddCommand(proxyClearCmd)
	proxyCmd.AddCommand(proxyStatusCmd)
	rootCmd.AddCommand(proxyCmd)
}
