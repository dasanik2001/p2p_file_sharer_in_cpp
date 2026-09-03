// cmd/install.go — Install Command
//
// This file defines the `transfera install` subcommand.
// It installs transfera.exe to a permanent location and adds it to the
// user's PATH so that typing `transfera` works in any terminal.
//
// Usage:
//   transfera install           Install transfera globally
//   transfera install --remove  Uninstall transfera from PATH

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/dasanik2001/transfera-client/cli/internal/installer"
)

var removeInstall bool

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install transfera globally so it works in any terminal",
	Long: `Install transfera.exe to your system so you can type "transfera" in any
terminal window. This copies the binary to %LOCALAPPDATA%\Programs\Transfera\
and adds it to your user PATH.

Examples:
  transfera install           Install transfera globally
  transfera install --remove  Uninstall transfera from PATH`,

	RunE: func(_ *cobra.Command, _ []string) error {
		if removeInstall {
			fmt.Printf("\n  Uninstalling transfera...\n\n")
			msg, err := installer.Uninstall()
			if err != nil {
				return fmt.Errorf("uninstall failed: %w", err)
			}
			fmt.Printf("  ✓ %s\n\n", msg)
			return nil
		}

		fmt.Printf("\n  Installing transfera...\n")
		fmt.Printf("  Target: %s\n\n", installer.InstallDir())

		msg, err := installer.Install()
		if err != nil {
			return fmt.Errorf("install failed: %w", err)
		}

		fmt.Printf("  ✓ %s\n\n", msg)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(installCmd)

	installCmd.Flags().BoolVar(
		&removeInstall,
		"remove",
		false,
		"Uninstall transfera from PATH",
	)
}
