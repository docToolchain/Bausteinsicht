package chaos

import (
	"os"
	"path/filepath"
	"testing"
)

type TestChaos struct {
	t      *testing.T
	tmpDir string
}

// NewTestChaos creates a chaos injection helper for tests.
func NewTestChaos(t *testing.T) *TestChaos {
	tmpDir := t.TempDir()
	return &TestChaos{
		t:      t,
		tmpDir: tmpDir,
	}
}

// TmpDir returns the temporary directory for this test.
func (tc *TestChaos) TmpDir() string {
	return tc.tmpDir
}

// CorruptFile truncates a file (simulating partial write).
func (tc *TestChaos) CorruptFile(path string) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		tc.t.Fatalf("CorruptFile: %v", err)
	}
	defer f.Close() //nolint:errcheck
	if _, err := f.WriteString(""); err != nil {
		tc.t.Fatalf("CorruptFile truncate: %v", err)
	}
}

// CorruptFilePartial truncates file to partial content.
func (tc *TestChaos) CorruptFilePartial(path string, content string) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		tc.t.Fatalf("CorruptFilePartial: %v", err)
	}
	defer f.Close() //nolint:errcheck
	if _, err := f.WriteString(content); err != nil {
		tc.t.Fatalf("CorruptFilePartial write: %v", err)
	}
}

// MakeReadOnly sets file permissions to read-only.
func (tc *TestChaos) MakeReadOnly(path string) {
	if err := os.Chmod(path, 0444); err != nil {
		tc.t.Fatalf("MakeReadOnly: %v", err)
	}
}

// MakeWritable sets file permissions to writable.
func (tc *TestChaos) MakeWritable(path string) {
	if err := os.Chmod(path, 0644); err != nil {
		tc.t.Fatalf("MakeWritable: %v", err)
	}
}

// DeleteFile removes a file.
func (tc *TestChaos) DeleteFile(path string) {
	if err := os.Remove(path); err != nil {
		tc.t.Fatalf("DeleteFile: %v", err)
	}
}

// CreateEmptyFile creates an empty file at path.
func (tc *TestChaos) CreateEmptyFile(path string) string {
	absPath := filepath.Join(tc.tmpDir, path)
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		tc.t.Fatalf("CreateEmptyFile mkdir: %v", err)
	}
	f, err := os.Create(absPath)
	if err != nil {
		tc.t.Fatalf("CreateEmptyFile: %v", err)
	}
	defer f.Close() //nolint:errcheck
	return absPath
}

// CreateFileWithContent creates a file with specific content.
func (tc *TestChaos) CreateFileWithContent(path string, content string) string {
	absPath := filepath.Join(tc.tmpDir, path)
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		tc.t.Fatalf("CreateFileWithContent mkdir: %v", err)
	}
	if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
		tc.t.Fatalf("CreateFileWithContent: %v", err)
	}
	return absPath
}

// FileExists checks if a file exists.
func (tc *TestChaos) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ReadFile reads a file's content.
func (tc *TestChaos) ReadFile(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		tc.t.Fatalf("ReadFile: %v", err)
	}
	return string(content)
}
