package cmd

import (
	"encoding/json"
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
	uiJSONFlag     bool
)

type UIElementJSON struct {
	Class       string          `json:"class"`
	ResourceID  string          `json:"resource_id,omitempty"`
	Text        string          `json:"text,omitempty"`
	ContentDesc string          `json:"content_desc,omitempty"`
	Clickable   bool            `json:"clickable"`
	Enabled     bool            `json:"enabled"`
	Focused     bool            `json:"focused"`
	Scrollable  bool            `json:"scrollable"`
	Bounds      string          `json:"bounds"`
	Children    []UIElementJSON `json:"children,omitempty"`
}

type UILayoutDumpResult struct {
	Device   string          `json:"device"`
	Serial   string          `json:"serial"`
	Elements []UIElementJSON `json:"elements"`
}

var uiDumpCmd = &cobra.Command{
	Use:     "ui",
	Aliases: []string{"layout", "hierarchy", "dump-ui", "nodes"},
	Short:   "Dump active screen UI hierarchy tree for inspection",
	Long: `Captures a UI Automator hierarchy dump from the active Android screen
and renders a colorized tree or structured JSON view of visible interactive elements
(text, resource-id, bounds, clickable state).

Examples:
  adx ui
  adx layout
  adx layout dump --json
  adx ui --json
  adx ui --filter Button
  adx ui --save screen_layout.xml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUIDump()
	},
}

var layoutDumpSubCmd = &cobra.Command{
	Use:   "dump",
	Short: "Dump screen layout hierarchy",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUIDump()
	},
}

func runUIDump() error {
	adbClient, err := GetADBClient()
	if err != nil {
		return err
	}

	targetDevices, err := ResolveTargetDevices(adbClient)
	if err != nil {
		return err
	}

	dev := targetDevices[0]

	if !IsJSON() && !uiJSONFlag {
		ui.Step("Dumping UI hierarchy from %s (%s)...", dev.Model, dev.Serial)
	}

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
			if !IsJSON() && !uiJSONFlag {
				ui.Success("Saved raw UI dump to: %s", ui.ClickablePath(uiSavePathFlag))
			}
		}
	}

	var hierarchy uiHierarchyXML
	if err := xml.Unmarshal(xmlData, &hierarchy); err != nil {
		return fmt.Errorf("failed to parse UI hierarchy XML: %w", err)
	}

	if IsJSON() || uiJSONFlag {
		jsonTree := convertNodeToJSON(hierarchy.RootNode, strings.ToLower(uiFilterFlag))
		dumpResult := UILayoutDumpResult{
			Device:   dev.Model,
			Serial:   dev.Serial,
			Elements: jsonTree,
		}
		data, _ := json.MarshalIndent(dumpResult, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Println()
	printNodeTree(hierarchy.RootNode, 0, strings.ToLower(uiFilterFlag))
	fmt.Println()

	return nil
}

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

func convertNodeToJSON(node uiNodeXML, filter string) []UIElementJSON {
	classShort := node.Class
	if idx := strings.LastIndex(classShort, "."); idx != -1 {
		classShort = classShort[idx+1:]
	}

	matchesFilter := filter == "" ||
		strings.Contains(strings.ToLower(classShort), filter) ||
		strings.Contains(strings.ToLower(node.ResourceID), filter) ||
		strings.Contains(strings.ToLower(node.Text), filter) ||
		strings.Contains(strings.ToLower(node.ContentDesc), filter)

	var childElements []UIElementJSON
	for _, child := range node.Children {
		childElements = append(childElements, convertNodeToJSON(child, filter)...)
	}

	if matchesFilter {
		elem := UIElementJSON{
			Class:       classShort,
			ResourceID:  node.ResourceID,
			Text:        node.Text,
			ContentDesc: node.ContentDesc,
			Clickable:   node.Clickable == "true",
			Enabled:     node.Enabled == "true",
			Focused:     node.Focused == "true",
			Scrollable:  node.Scrollable == "true",
			Bounds:      node.Bounds,
			Children:    childElements,
		}
		return []UIElementJSON{elem}
	}

	return childElements
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
	uiDumpCmd.Flags().BoolVar(&uiJSONFlag, "json", false, "Emit clean interactive JSON structure")

	layoutDumpSubCmd.Flags().StringVar(&uiSavePathFlag, "save", "", "Save raw XML layout dump to a file")
	layoutDumpSubCmd.Flags().StringVarP(&uiFilterFlag, "filter", "f", "", "Filter view nodes by class, ID, or text")
	layoutDumpSubCmd.Flags().BoolVar(&uiJSONFlag, "json", false, "Emit clean interactive JSON structure")

	uiDumpCmd.AddCommand(layoutDumpSubCmd)
	rootCmd.AddCommand(uiDumpCmd)
}
