package cmd

import (
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Shashwat-CODING/adx/internal/ui"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	uiSavePathFlag string
	uiFilterFlag   string

	uiDumpCmd = &cobra.Command{
		Use:     "ui",
		Aliases: []string{"hierarchy", "dump-ui", "nodes"},
		Short:   "Dump active screen UI hierarchy tree for inspection",
		Long: `Captures a UI Automator hierarchy dump from the active Android screen
and renders a colorized tree showing view types, resource IDs, text, bounds, and states.

Examples:
  adx ui
  adx hierarchy
  adx ui --filter Button
  adx ui --save screen_layout.xml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			adbClient, err := GetADBClient()
			if err != nil {
				return err
			}

			devices, err := adbClient.GetDevices()
			if err != nil {
				return err
			}
			if len(devices) == 0 {
				return fmt.Errorf("no connected Android devices found")
			}

			dev := devices[0]
			if deviceFlag != "" {
				for _, d := range devices {
					if d.Serial == deviceFlag {
						dev = d
						break
					}
				}
			}

			ui.Step("Dumping UI hierarchy from %s (%s)...", dev.Model, dev.Serial)

			// Execute uiautomator dump on device
			dumpCmd := exec.Command(adbClient.AdbPath, "-s", dev.Serial, "shell", "uiautomator", "dump", "/sdcard/window_dump.xml")
			if out, err := dumpCmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to dump UI hierarchy: %s (%w)", string(out), err)
			}

			catCmd := exec.Command(adbClient.AdbPath, "-s", dev.Serial, "shell", "cat", "/sdcard/window_dump.xml")
			xmlData, err := catCmd.Output()
			if err != nil {
				return fmt.Errorf("failed to read UI hierarchy XML: %w", err)
			}

			// Clean up remote temp file
			_ = exec.Command(adbClient.AdbPath, "-s", dev.Serial, "shell", "rm", "-f", "/sdcard/window_dump.xml").Run()

			if uiSavePathFlag != "" {
				if err := os.WriteFile(uiSavePathFlag, xmlData, 0644); err == nil {
					ui.Success("Saved raw UI dump to: %s", ui.ClickablePath(uiSavePathFlag))
				}
			}

			var hierarchy uiHierarchyXML
			if err := xml.Unmarshal(xmlData, &hierarchy); err != nil {
				return fmt.Errorf("failed to parse UI hierarchy XML: %w", err)
			}

			fmt.Println()
			printNodeTree(hierarchy.RootNode, 0, strings.ToLower(uiFilterFlag))
			fmt.Println()

			return nil
		},
	}
)

type uiNodeXML struct {
	Index       string      `xml:"index,attr"`
	Text        string      `xml:"text,attr"`
	ResourceID  string      `xml:"resource-id,attr"`
	Class       string      `xml:"class,attr"`
	Package     string      `xml:"package,attr"`
	ContentDesc string      `xml:"content-desc,attr"`
	Clickable   string      `xml:"clickable,attr"`
	Enabled     string      `xml:"enabled,attr"`
	Focused     string      `xml:"focused,attr"`
	Scrollable  string      `xml:"scrollable,attr"`
	Bounds      string      `xml:"bounds,attr"`
	Children    []uiNodeXML `xml:"node"`
}

type uiHierarchyXML struct {
	XMLName  xml.Name  `xml:"hierarchy"`
	RootNode uiNodeXML `xml:"node"`
}

func printNodeTree(node uiNodeXML, depth int, filter string) {
	classShort := node.Class
	if idx := strings.LastIndex(classShort, "."); idx != -1 {
		classShort = classShort[idx+1:]
	}

	matchesFilter := filter == "" ||
		strings.Contains(strings.ToLower(classShort), filter) ||
		strings.Contains(strings.ToLower(node.ResourceID), filter) ||
		strings.Contains(strings.ToLower(node.Text), filter) ||
		strings.Contains(strings.ToLower(node.ContentDesc), filter)

	if matchesFilter {
		indent := strings.Repeat("  ", depth)
		classFormatted := color.New(color.Bold, color.FgCyan).Sprint(classShort)
		
		var details []string
		if node.ResourceID != "" {
			resShort := node.ResourceID
			if idx := strings.LastIndex(resShort, ":id/"); idx != -1 {
				resShort = "id/" + resShort[idx+4:]
			}
			details = append(details, ui.Orange().Sprint(resShort))
		}
		if node.Text != "" {
			details = append(details, fmt.Sprintf("%s", color.New(color.FgYellow).Sprintf("\"%s\"", node.Text)))
		}
		if node.ContentDesc != "" {
			details = append(details, fmt.Sprintf("%s", color.New(color.FgMagenta).Sprintf("desc:\"%s\"", node.ContentDesc)))
		}
		if node.Clickable == "true" {
			details = append(details, color.New(color.FgGreen).Sprint("[clickable]"))
		}
		if node.Bounds != "" {
			details = append(details, color.New(color.Faint).Sprint(node.Bounds))
		}

		if len(details) > 0 {
			fmt.Printf("%s• %s %s\n", indent, classFormatted, strings.Join(details, " "))
		} else {
			fmt.Printf("%s• %s\n", indent, classFormatted)
		}
	}

	for _, child := range node.Children {
		printNodeTree(child, depth+1, filter)
	}
}

func init() {
	uiDumpCmd.Flags().StringVar(&uiSavePathFlag, "save", "", "Save raw XML layout dump to a file")
	uiDumpCmd.Flags().StringVarP(&uiFilterFlag, "filter", "f", "", "Filter view nodes by class, ID, or text")
	rootCmd.AddCommand(uiDumpCmd)
}
