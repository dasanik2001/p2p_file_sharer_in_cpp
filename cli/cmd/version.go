// cmd/version.go — Version Command
//
// Displays the CLI version, target platform, and PATH installation status.

package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/dasanik2001/transfera-client/cli/internal/installer"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version and installation status",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("\n  Transfera CLI v%s (%s/%s)\n\n", Version, runtime.GOOS, runtime.GOARCH)

		if installer.IsInstalled() && installer.IsInPath() {
			if installer.IsUpdateAvailable() {
				fmt.Printf("  ● Installed binary: %s\n", installer.InstalledBinaryPath())
				fmt.Printf("  ● Status:           🔄 Update available (running binary differs from installed)\n")
				fmt.Printf("  ● To update:        Run 'transfera install' to update the installed binary.\n\n")
			} else {
				fmt.Printf("  ● Installed binary: %s\n", installer.InstalledBinaryPath())
				fmt.Printf("  ● Status:           ✓ Up to date\n\n")
			}
		} else {
			fmt.Printf("  ● Status:           ⚠ Not installed to PATH\n")
			fmt.Printf("  ● To install:       Run 'transfera install' to make transfera available globally.\n\n")
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
