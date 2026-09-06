//go:build !windows

// internal/installer/installer_unix.go — Linux & macOS PATH Installer
//
// This file handles installing the transfera binary on Linux and macOS so that
// typing "transfera" in any terminal window works globally. It:
//   1. Copies the current binary to ~/.local/bin/ (or $XDG_BIN_HOME) with 0755 permissions
//   2. Checks if the install directory is already in the user's PATH
//   3. If not, adds it to the user's shell configuration (~/.zshrc, ~/.bashrc, etc.)

package installer

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	transferaMarker = "# Added by Transfera CLI"
)

// InstallDir returns the target installation directory on Linux/macOS.
// Standard is ~/.local/bin (per XDG Base Directory specification).
func InstallDir() string {
	if xdgBin := os.Getenv("XDG_BIN_HOME"); xdgBin != "" {
		return xdgBin
	}

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "/usr/local/bin"
	}
	return filepath.Join(home, ".local", "bin")
}

// InstalledBinaryPath returns the full path to the installed transfera binary.
func InstalledBinaryPath() string {
	return filepath.Join(InstallDir(), "transfera")
}

// IsInPath checks if the install directory is in PATH or configured in shell rc.
func IsInPath() bool {
	installDir := filepath.Clean(InstallDir())

	// 1. Check current process PATH
	for _, p := range filepath.SplitList(os.Getenv("PATH")) {
		if filepath.Clean(p) == installDir {
			return true
		}
	}

	// 2. Check if already configured in shell rc file
	rcPath := detectShellRC()
	if rcPath != "" && isConfiguredInRC(rcPath, installDir) {
		return true
	}

	return false
}

// Install copies transfera to the install directory and ensures it is in PATH.
// Returns a human-readable status message.
func Install() (string, error) {
	installDir := InstallDir()
	destPath := InstalledBinaryPath()

	// --- Step 1: Create the installation directory ---
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return "", fmt.Errorf("cannot create directory %s: %w", installDir, err)
	}

	// --- Step 2: Copy the current binary to the install location ---
	srcPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("cannot determine current executable path: %w", err)
	}

	// Resolve symlinks
	srcPath, err = filepath.EvalSymlinks(srcPath)
	if err != nil {
		return "", fmt.Errorf("cannot resolve executable path: %w", err)
	}

	// Don't copy if source and destination are the same
	srcAbs, _ := filepath.Abs(srcPath)
	dstAbs, _ := filepath.Abs(destPath)
	if srcAbs != dstAbs {
		if err := copyFile(srcPath, destPath); err != nil {
			return "", fmt.Errorf("cannot copy binary to %s: %w", destPath, err)
		}
	}

	// Ensure binary has executable permissions (0755)
	if err := os.Chmod(destPath, 0755); err != nil {
		return "", fmt.Errorf("cannot set executable permissions on %s: %w", destPath, err)
	}

	// --- Step 3: Ensure directory is in PATH ---
	if IsInPath() {
		return fmt.Sprintf("Installed to %s (already in PATH).\nType: transfera", destPath), nil
	}

	// Add to user's shell configuration
	rcPath, err := addToShellRC(installDir)
	if err != nil {
		return fmt.Sprintf("Installed to %s.\nNotice: Could not automatically update shell configuration: %v\nPlease add %s to your PATH manually.", destPath, err, installDir), nil
	}

	relRC := formatRelativePath(rcPath)
	return fmt.Sprintf("Installed to %s and added to PATH in %s.\nOpen a NEW terminal (or run 'source %s') and type: transfera", destPath, relRC, relRC), nil
}

// Uninstall removes transfera from the install directory and cleans PATH entries.
func Uninstall() (string, error) {
	installDir := InstallDir()
	destPath := InstalledBinaryPath()

	// Remove binary
	if _, err := os.Stat(destPath); err == nil {
		if err := os.Remove(destPath); err != nil {
			return "", fmt.Errorf("cannot remove %s: %w", destPath, err)
		}
	}

	// Remove from all candidate shell configuration files
	removeFromAllShellRCs(installDir)

	// Try to remove install directory (only succeeds if empty)
	_ = os.Remove(installDir)

	return "Transfera has been uninstalled and removed from PATH.", nil
}

// =========================================================================
// Shell Configuration Helpers
// =========================================================================

// detectShellRC finds the primary rc/profile file for the user's active shell.
func detectShellRC() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}

	shell := filepath.Base(os.Getenv("SHELL"))
	switch shell {
	case "zsh":
		return filepath.Join(home, ".zshrc")
	case "fish":
		return filepath.Join(home, ".config", "fish", "config.fish")
	case "bash":
		if runtime.GOOS == "darwin" {
			// macOS bash prefers .bash_profile for login shells
			bashProfile := filepath.Join(home, ".bash_profile")
			if _, err := os.Stat(bashProfile); err == nil {
				return bashProfile
			}
			bashrc := filepath.Join(home, ".bashrc")
			if _, err := os.Stat(bashrc); err == nil {
				return bashrc
			}
			return bashProfile
		}
		bashrc := filepath.Join(home, ".bashrc")
		if _, err := os.Stat(bashrc); err == nil {
			return bashrc
		}
		bashProfile := filepath.Join(home, ".bash_profile")
		if _, err := os.Stat(bashProfile); err == nil {
			return bashProfile
		}
		return bashrc
	default:
		// Check existing files in order
		for _, name := range []string{".zshrc", ".bashrc", ".bash_profile", ".profile"} {
			p := filepath.Join(home, name)
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
		return filepath.Join(home, ".profile")
	}
}

// candidateRCs returns all potential shell configuration files that exist.
func candidateRCs() []string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return nil
	}

	names := []string{
		filepath.Join(home, ".zshrc"),
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".bash_profile"),
		filepath.Join(home, ".profile"),
		filepath.Join(home, ".config", "fish", "config.fish"),
	}

	var existing []string
	for _, path := range names {
		if _, err := os.Stat(path); err == nil {
			existing = append(existing, path)
		}
	}
	return existing
}

func isConfiguredInRC(rcPath, dir string) bool {
	content, err := os.ReadFile(rcPath)
	if err != nil {
		return false
	}
	text := string(content)
	return strings.Contains(text, transferaMarker) || strings.Contains(text, dir)
}

func addToShellRC(installDir string) (string, error) {
	rcPath := detectShellRC()
	if rcPath == "" {
		return "", fmt.Errorf("unable to determine user home or shell configuration file")
	}

	// Ensure parent directory exists (e.g. for ~/.config/fish/)
	if err := os.MkdirAll(filepath.Dir(rcPath), 0755); err != nil {
		return "", err
	}

	// Build export line
	home, _ := os.UserHomeDir()
	var pathSnippet string
	if home != "" && strings.HasPrefix(installDir, home) {
		pathSnippet = "$HOME" + installDir[len(home):]
	} else {
		pathSnippet = installDir
	}

	isFish := strings.HasSuffix(rcPath, "config.fish")
	var exportCmd string
	if isFish {
		exportCmd = fmt.Sprintf("\n%s\nset -gx PATH \"%s\" $PATH\n", transferaMarker, pathSnippet)
	} else {
		exportCmd = fmt.Sprintf("\n%s\nexport PATH=\"%s:$PATH\"\n", transferaMarker, pathSnippet)
	}

	// Open file in append mode (create if does not exist)
	f, err := os.OpenFile(rcPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := f.WriteString(exportCmd); err != nil {
		return "", err
	}

	return rcPath, nil
}

func removeFromAllShellRCs(installDir string) {
	rcs := candidateRCs()
	home, _ := os.UserHomeDir()
	var pathSnippet string
	if home != "" && strings.HasPrefix(installDir, home) {
		pathSnippet = "$HOME" + installDir[len(home):]
	}

	for _, rc := range rcs {
		removeFromFile(rc, installDir, pathSnippet)
	}
}

func removeFromFile(filePath, installDir, pathSnippet string) {
	f, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer f.Close()

	var keptLines []string
	scanner := bufio.NewScanner(f)
	skipNext := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == transferaMarker {
			skipNext = true
			continue
		}

		if skipNext {
			skipNext = false
			if strings.Contains(line, "PATH") && (strings.Contains(line, installDir) || (pathSnippet != "" && strings.Contains(line, pathSnippet))) {
				continue
			}
		}

		if strings.Contains(line, installDir) && strings.Contains(line, "PATH") {
			continue
		}
		if pathSnippet != "" && strings.Contains(line, pathSnippet) && strings.Contains(line, "PATH") {
			continue
		}

		keptLines = append(keptLines, line)
	}

	if err := scanner.Err(); err != nil {
		return
	}

	newContent := strings.Join(keptLines, "\n")
	if len(keptLines) > 0 {
		newContent += "\n"
	}
	_ = os.WriteFile(filePath, []byte(newContent), 0644)
}

func formatRelativePath(path string) string {
	home, err := os.UserHomeDir()
	if err == nil && home != "" && strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

