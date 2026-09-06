package installer

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInstallDirAndBinaryPath(t *testing.T) {
	dir := InstallDir()
	if dir == "" {
		t.Errorf("InstallDir() returned empty string")
	}

	bin := InstalledBinaryPath()
	if bin == "" {
		t.Errorf("InstalledBinaryPath() returned empty string")
	}

	if runtime.GOOS == "windows" {
		if !strings.HasSuffix(bin, "transfera.exe") {
			t.Errorf("expected transfera.exe suffix on Windows, got %s", bin)
		}
	} else {
		if !strings.HasSuffix(bin, "transfera") {
			t.Errorf("expected transfera suffix on Unix, got %s", bin)
		}
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "source.txt")
	dst := filepath.Join(tmpDir, "dest.txt")

	data := []byte("binary data test")
	if err := os.WriteFile(src, data, 0644); err != nil {
		t.Fatal(err)
	}

	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("reading copied file failed: %v", err)
	}

	if string(got) != string(data) {
		t.Errorf("copied content mismatch: got %q, want %q", string(got), string(data))
	}

	// Verify permissions on Unix
	if runtime.GOOS != "windows" {
		info, err := os.Stat(dst)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode()&0111 == 0 {
			t.Errorf("expected executable bit to be set, mode: %v", info.Mode())
		}
	}
}

func TestShellRCHelpers(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping shell RC helpers test on Windows")
	}

	tmpDir := t.TempDir()
	fakeRC := filepath.Join(tmpDir, ".testrc")

	// Test initial content
	initialContent := "# Existing config\nexport FOO=bar\n"
	if err := os.WriteFile(fakeRC, []byte(initialContent), 0644); err != nil {
		t.Fatal(err)
	}

	testDir := "/test/custom/bin"
	if isConfiguredInRC(fakeRC, testDir) {
		t.Errorf("expected isConfiguredInRC to be false initially")
	}

	// Append export manually to fakeRC to test removal
	exportLine := "\n# Added by Transfera CLI\nexport PATH=\"/test/custom/bin:$PATH\"\n"
	f, err := os.OpenFile(fakeRC, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(exportLine)
	f.Close()

	if !isConfiguredInRC(fakeRC, testDir) {
		t.Errorf("expected isConfiguredInRC to be true after adding")
	}

	// Test removeFromFile
	removeFromFile(fakeRC, testDir, "")
	afterRemove, err := os.ReadFile(fakeRC)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(afterRemove), testDir) {
		t.Errorf("after removeFromFile, testDir was still found in file: %s", string(afterRemove))
	}
	if strings.Contains(string(afterRemove), transferaMarker) {
		t.Errorf("after removeFromFile, marker was still found in file: %s", string(afterRemove))
	}
	if !strings.Contains(string(afterRemove), "export FOO=bar") {
		t.Errorf("original content was corrupted: %s", string(afterRemove))
	}
}

func TestFileHash(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := filepath.Join(tmpDir, "f1.txt")
	f2 := filepath.Join(tmpDir, "f2.txt")
	f3 := filepath.Join(tmpDir, "f3.txt")

	if err := os.WriteFile(f1, []byte("version 1.0"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f2, []byte("version 1.0"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f3, []byte("version 2.0"), 0644); err != nil {
		t.Fatal(err)
	}

	h1, err := fileHash(f1)
	if err != nil {
		t.Fatalf("fileHash(f1) failed: %v", err)
	}
	h2, err := fileHash(f2)
	if err != nil {
		t.Fatalf("fileHash(f2) failed: %v", err)
	}
	h3, err := fileHash(f3)
	if err != nil {
		t.Fatalf("fileHash(f3) failed: %v", err)
	}

	if h1 != h2 {
		t.Errorf("expected identical hashes for f1 and f2, got %s and %s", h1, h2)
	}
	if h1 == h3 {
		t.Errorf("expected different hashes for f1 and f3, both got %s", h1)
	}
}

