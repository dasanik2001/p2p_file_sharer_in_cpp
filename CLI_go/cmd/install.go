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

var (
	removeInstall    bool
	reinstallInstall bool
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install transfera globally so it works in any terminal",
	Long:  "", // Dynamically configured in init()

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

		isInstalled := installer.IsInstalled()
		isInPath := installer.IsInPath()
		hasUpdate := isInstalled && installer.IsUpdateAvailable()

		if isInstalled && isInPath {
			if hasUpdate {
				fmt.Printf("\n  Updating transfera in PATH (new version detected)...\n")
			} else {
				fmt.Printf("\n  Reinstalling transfera in PATH...\n")
			}
		} else {
			fmt.Printf("\n  Installing transfera...\n")
		}
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
	installCmd.Long = fmt.Sprintf(`Install transfera to your system so you can type "transfera" in any
terminal window. This copies the binary to %s
and adds it to your user PATH.

If transfera is already installed, running this command will update/reinstall
the binary with the current version.

Examples:
  transfera install              Install or update transfera globally
  transfera install --reinstall  Reinstall / update transfera in PATH
  transfera install --remove     Uninstall transfera from PATH`, installer.InstallDir())

	rootCmd.AddCommand(installCmd)

	installCmd.Flags().BoolVar(
		&removeInstall,
		"remove",
		false,
		"Uninstall transfera from PATH",
	)

	installCmd.Flags().BoolVarP(
		&reinstallInstall,
		"reinstall", "r",
		false,
		"Reinstall or update transfera in PATH",
	)
}
