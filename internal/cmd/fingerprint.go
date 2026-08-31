package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Shashwat-CODING/adx/internal/ui"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	keystorePathFlag string
	keystoreAliasFlag string
	keystorePassFlag  string

	fingerprintCmd = &cobra.Command{
		Use:     "sha",
		Aliases: []string{"fingerprint", "keystore"},
		Short:   "Display SHA-1 and SHA-256 fingerprints for Firebase & Google APIs",
		Long: `Extracts certificate fingerprints (SHA-1, SHA-256, MD5) from your debug (or release) keystore.
Crucial for configuring Firebase, Google Sign-In, and Maps APIs.

Examples:
  adx sha
  adx fingerprint
  adx sha --keystore /path/to/release.jks --alias mykey --password secret`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ksPath := keystorePathFlag
			if ksPath == "" {
				userHome, err := os.UserHomeDir()
				if err != nil {
					return err
				}
				ksPath = filepath.Join(userHome, ".android", "debug.keystore")
			}

			if _, err := os.Stat(ksPath); err != nil {
				return fmt.Errorf("keystore not found at %s", ksPath)
			}

			alias := keystoreAliasFlag
			pass := keystorePassFlag

			keytoolCmd := exec.Command("keytool", "-list", "-v", "-keystore", ksPath, "-alias", alias, "-storepass", pass)
			out, err := keytoolCmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("keytool failed: %s (%w)", strings.TrimSpace(string(out)), err)
			}

			var sha1, sha256, md5 string
			scanner := bufio.NewScanner(bytes.NewReader(out))
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if strings.HasPrefix(line, "SHA1:") || strings.HasPrefix(line, "SHA-1:") {
					sha1 = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "SHA1:"), "SHA-1:"))
				} else if strings.HasPrefix(line, "SHA256:") || strings.HasPrefix(line, "SHA-256:") {
					sha256 = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "SHA256:"), "SHA-256:"))
				} else if strings.HasPrefix(line, "MD5:") {
					md5 = strings.TrimSpace(strings.TrimPrefix(line, "MD5:"))
				}
			}

			ui.Step("Certificate Fingerprints (%s)", filepath.Base(ksPath))
			fmt.Println()
			if sha1 != "" {
				fmt.Printf("  %-12s %s\n", ui.Orange().Sprint("SHA-1:"), color.New(color.FgGreen, color.Bold).Sprint(sha1))
			}
			if sha256 != "" {
				fmt.Printf("  %-12s %s\n", ui.Orange().Sprint("SHA-256:"), ui.OrangeSoft().Sprint(sha256))
			}
			if md5 != "" {
				fmt.Printf("  %-12s %s\n", ui.Orange().Sprint("MD5:"), color.New(color.Faint).Sprint(md5))
			}
			fmt.Println()
			ui.Dim("Ready to paste directly into Firebase Console or Google Cloud Credentials.")
			return nil
		},
	}
)

func init() {
	fingerprintCmd.Flags().StringVar(&keystorePathFlag, "keystore", "", "Path to keystore file (defaults to ~/.android/debug.keystore)")
	fingerprintCmd.Flags().StringVar(&keystoreAliasFlag, "alias", "androiddebugkey", "Keystore alias")
	fingerprintCmd.Flags().StringVar(&keystorePassFlag, "password", "android", "Keystore password")
	rootCmd.AddCommand(fingerprintCmd)
}
