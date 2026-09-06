package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolvePath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("skipping test: cannot get user home dir")
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"~", home},
		{"~/test.txt", filepath.Join(home, "test.txt")},
		{`"~/test.txt"`, filepath.Join(home, "test.txt")},
		{`'/tmp/file.txt'`, "/tmp/file.txt"},
		{"/var/log", "/var/log"},
		{"   /var/log   ", "/var/log"},
	}

	for _, tt := range tests {
		got := ResolvePath(tt.input)
		if got != tt.expected {
			t.Errorf("ResolvePath(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.00 MB"},
		{104857600, "100.00 MB"},
		{1073741824, "1.00 GB"},
	}

	for _, tt := range tests {
		got := FormatFileSize(tt.bytes)
		if got != tt.expected {
			t.Errorf("FormatFileSize(%d) = %q; want %q", tt.bytes, got, tt.expected)
		}
	}
}

func TestValidateFile(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Non-existent file
	_, err := ValidateFile(filepath.Join(tmpDir, "missing.txt"), 100)
	if err == nil || !strings.Contains(err.Error(), "file not found") {
		t.Errorf("expected file not found error, got: %v", err)
	}

	// 2. Directory instead of file
	subDir := filepath.Join(tmpDir, "folder")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	_, err = ValidateFile(subDir, 100)
	if err == nil || !strings.Contains(err.Error(), "is a directory") {
		t.Errorf("expected directory error, got: %v", err)
	}

	// 3. Empty file
	emptyFile := filepath.Join(tmpDir, "empty.txt")
	if err := os.WriteFile(emptyFile, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	_, err = ValidateFile(emptyFile, 100)
	if err == nil || !strings.Contains(err.Error(), "is empty") {
		t.Errorf("expected empty file error, got: %v", err)
	}

	// 4. Valid file
	validFile := filepath.Join(tmpDir, "valid.txt")
	content := []byte("Hello Transfera")
	if err := os.WriteFile(validFile, content, 0644); err != nil {
		t.Fatal(err)
	}
	size, err := ValidateFile(validFile, 100)
	if err != nil {
		t.Errorf("unexpected error for valid file: %v", err)
	}
	if size != int64(len(content)) {
		t.Errorf("expected size %d, got %d", len(content), size)
	}

	// 5. File exceeds max size
	_, err = ValidateFile(validFile, 0) // default 100MB is larger than len(content)
	if err != nil {
		t.Errorf("expected file to be within default limit: %v", err)
	}

	// Exceeds custom limit (file has 15 bytes, limit 1 byte is ~1MB so let's test small bytes)
	// Note: maxSizeMB is in MB, so we test with a file larger than maxSizeMB
	bigFile := filepath.Join(tmpDir, "big.bin")
	bigContent := make([]byte, 2*1024*1024) // 2MB
	if err := os.WriteFile(bigFile, bigContent, 0644); err != nil {
		t.Fatal(err)
	}
	_, err = ValidateFile(bigFile, 1) // 1MB limit
	if err == nil || !strings.Contains(err.Error(), "file must be 1 MB or smaller") {
		t.Errorf("expected size exceeded error, got: %v", err)
	}
}

