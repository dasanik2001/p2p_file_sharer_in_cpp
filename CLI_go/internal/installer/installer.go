// internal/installer/installer.go — Windows PATH Installer
//
// This file handles installing transfera.exe so that typing "transfera"
// in any terminal window works globally. It:
//   1. Copies the current binary to %LOCALAPPDATA%\Programs\Transfera\
//   2. Adds that directory to the user's PATH via the Windows Registry
//   3. Broadcasts WM_SETTINGCHANGE so new terminals pick up the change

package installer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

// InstallDir returns the target installation directory.
func InstallDir() string {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		localAppData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
	}
	return filepath.Join(localAppData, "Programs", "Transfera")
}

// InstalledBinaryPath returns the full path to the installed transfera.exe.
func InstalledBinaryPath() string {
	return filepath.Join(InstallDir(), "transfera.exe")
}

// IsInstalled checks if transfera.exe is already installed in the target dir.
func IsInstalled() bool {
	_, err := os.Stat(InstalledBinaryPath())
	return err == nil
}

// IsInPath checks if the install directory is already in the user's PATH.
func IsInPath() bool {
	installDir := InstallDir()

	k, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()

	currentPath, _, err := k.GetStringValue("Path")
	if err != nil {
		return false
	}

	for _, p := range strings.Split(currentPath, ";") {
		cleaned := strings.TrimSpace(p)
		if strings.EqualFold(cleaned, installDir) {
			return true
		}
	}
	return false
}

// Install copies transfera.exe to the install directory and adds it to PATH.
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
	if !strings.EqualFold(srcAbs, dstAbs) {
		if err := copyFile(srcPath, destPath); err != nil {
			return "", fmt.Errorf("cannot copy binary to %s: %w", destPath, err)
		}
	}

	// --- Step 3: Add to PATH if not already there ---
	if !IsInPath() {
		if err := addToPath(installDir); err != nil {
			return "", fmt.Errorf("cannot add to PATH: %w", err)
		}

		// Broadcast WM_SETTINGCHANGE so new terminals pick up the change
		broadcastSettingChange()

		return fmt.Sprintf("Installed to %s and added to PATH.\nOpen a NEW terminal and type: transfera", destPath), nil
	}

	return fmt.Sprintf("Updated binary at %s (already in PATH).\nType: transfera", destPath), nil
}

// Uninstall removes transfera from the install directory and PATH.
func Uninstall() (string, error) {
	installDir := InstallDir()
	destPath := InstalledBinaryPath()

	// Remove binary
	if _, err := os.Stat(destPath); err == nil {
		if err := os.Remove(destPath); err != nil {
			return "", fmt.Errorf("cannot remove %s: %w", destPath, err)
		}
	}

	// Remove from PATH
	if IsInPath() {
		if err := removeFromPath(installDir); err != nil {
			return "", fmt.Errorf("cannot remove from PATH: %w", err)
		}
		broadcastSettingChange()
	}

	// Try to remove the install directory (only if empty)
	_ = os.Remove(installDir)

	return "Transfera has been uninstalled and removed from PATH.", nil
}

// =========================================================================
// Internal helpers
// =========================================================================

func copyFile(src, dst string) error {
	// Write to a temp file first, then rename for atomic replacement
	tmpDst := dst + ".tmp"

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(tmpDst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		os.Remove(tmpDst)
		return err
	}

	if err := dstFile.Close(); err != nil {
		os.Remove(tmpDst)
		return err
	}

	// Remove old destination if it exists (can't rename over a running exe)
	_ = os.Remove(dst)

	return os.Rename(tmpDst, dst)
}

func addToPath(dir string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	currentPath, _, err := k.GetStringValue("Path")
	if err != nil && err != registry.ErrNotExist {
		return err
	}

	// Append our directory
	newPath := currentPath
	if newPath != "" && !strings.HasSuffix(newPath, ";") {
		newPath += ";"
	}
	newPath += dir

	return k.SetExpandStringValue("Path", newPath)
}

func removeFromPath(dir string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	currentPath, _, err := k.GetStringValue("Path")
	if err != nil {
		return err
	}

	var filtered []string
	for _, p := range strings.Split(currentPath, ";") {
		cleaned := strings.TrimSpace(p)
		if cleaned == "" || strings.EqualFold(cleaned, dir) {
			continue
		}
		filtered = append(filtered, cleaned)
	}

	return k.SetExpandStringValue("Path", strings.Join(filtered, ";"))
}

// broadcastSettingChange notifies Windows that environment variables have changed.
// This makes new cmd/PowerShell windows pick up the PATH change immediately.
func broadcastSettingChange() {
	user32 := syscall.NewLazyDLL("user32.dll")
	sendMessageTimeout := user32.NewProc("SendMessageTimeoutW")

	// HWND_BROADCAST = 0xFFFF, WM_SETTINGCHANGE = 0x001A
	envStr, _ := syscall.UTF16PtrFromString("Environment")
	sendMessageTimeout.Call(
		uintptr(0xFFFF),        // HWND_BROADCAST
		uintptr(0x001A),        // WM_SETTINGCHANGE
		0,                      // wParam
		uintptr(unsafe.Pointer(envStr)), // lParam
		uintptr(0x0002),        // SMTO_ABORTIFHUNG
		uintptr(5000),          // timeout ms
		0,                      // result (unused)
	)
}
