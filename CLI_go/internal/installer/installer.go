// internal/installer/installer.go — Cross-Platform Installer Interface & Helpers
//
// This file provides shared logic for installing the Transfera CLI binary so that
// typing "transfera" in any terminal window works globally.
//
// Platform-specific implementations:
//   - Windows: installer_windows.go (Registry HKCU\Environment, WM_SETTINGCHANGE)
//   - Unix:    installer_unix.go    (~/.local/bin, shell configuration files)

package installer

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
)

// IsInstalled checks if transfera is already installed in the target dir.
func IsInstalled() bool {
	_, err := os.Stat(InstalledBinaryPath())
	return err == nil
}

// IsUpdateAvailable checks if the running binary is different from the installed binary
// (meaning an update or new version is available).
func IsUpdateAvailable() bool {
	destPath := InstalledBinaryPath()
	destInfo, err := os.Stat(destPath)
	if err != nil {
		return false
	}

	srcPath, err := os.Executable()
	if err != nil {
		return false
	}
	srcPath, err = filepath.EvalSymlinks(srcPath)
	if err != nil {
		return false
	}

	srcAbs, err1 := filepath.Abs(srcPath)
	dstAbs, err2 := filepath.Abs(destPath)
	if err1 == nil && err2 == nil && srcAbs == dstAbs {
		// Currently running directly from the installed location
		return false
	}

	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		return false
	}

	// Quick check: size difference
	if srcInfo.Size() != destInfo.Size() {
		return true
	}

	// Full check: content hash difference
	srcHash, err := fileHash(srcPath)
	if err != nil {
		return false
	}
	dstHash, err := fileHash(destPath)
	if err != nil {
		return false
	}

	return srcHash != dstHash
}

func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// copyFile writes src to dst atomically via a temporary file.
func copyFile(src, dst string) error {
	tmpDst := dst + ".tmp"

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(tmpDst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		os.Remove(tmpDst)
		return err
	}

	if err := dstFile.Sync(); err != nil {
		os.Remove(tmpDst)
		return err
	}

	if err := dstFile.Close(); err != nil {
		os.Remove(tmpDst)
		return err
	}

	// Ensure executable permissions before replacing
	_ = os.Chmod(tmpDst, 0755)

	// Remove old destination if it exists (atomic rename requires it on Windows)
	_ = os.Remove(dst)

	return os.Rename(tmpDst, dst)
}
