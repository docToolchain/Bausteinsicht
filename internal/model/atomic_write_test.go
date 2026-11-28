package model

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func skipWithoutPermissionBits(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs permission bits that are enforced (not Windows, not root)")
	}
}

// A file that does not exist yet is created.
func TestWriteFileAtomic_CreatesANewFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new.jsonc")
	if err := WriteFileAtomic(path, ".t-*", []byte("x")); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(path); string(data) != "x" {
		t.Errorf("got %q", data)
	}
}

// The error paths name what failed and leave no temporary file behind.
func TestWriteFileAtomic_Errors(t *testing.T) {
	t.Run("the target is a directory", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "d")
		if err := os.Mkdir(target, 0o750); err != nil {
			t.Fatal(err)
		}
		err := WriteFileAtomic(target, ".t-*", []byte("x"))
		if err == nil || !strings.Contains(err.Error(), "renaming temp file") {
			t.Errorf("want a rename error, got %v", err)
		}
		if entries, _ := os.ReadDir(dir); len(entries) != 1 {
			t.Errorf("the temporary file is left behind: %v", entries)
		}
	})
	t.Run("the directory cannot be searched", func(t *testing.T) {
		skipWithoutPermissionBits(t)
		dir := t.TempDir()
		locked := filepath.Join(dir, "locked")
		if err := os.Mkdir(locked, 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(locked, 0); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(locked, 0o750) })
		err := WriteFileAtomic(filepath.Join(locked, "m.jsonc"), ".t-*", []byte("x"))
		if err == nil || !strings.Contains(err.Error(), "checking") {
			t.Errorf("want a \"checking\" error, got %v", err)
		}
	})
	t.Run("the directory is not writable", func(t *testing.T) {
		skipWithoutPermissionBits(t)
		dir := t.TempDir()
		path := filepath.Join(dir, "m.jsonc")
		if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(dir, 0o500); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(dir, 0o750) })
		err := WriteFileAtomic(path, ".t-*", []byte("x"))
		if err == nil || !strings.Contains(err.Error(), "creating temp file") {
			t.Errorf("want a \"creating temp file\" error, got %v", err)
		}
		if data, _ := os.ReadFile(path); string(data) != "old" {
			t.Errorf("the file was changed: %q", data)
		}
	})
}

// fillTemp names the step that failed.
func TestFillTemp_Errors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	closed, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	_ = closed.Close()
	if err := fillTemp(closed, 0o600, true, []byte("y")); err == nil || !strings.Contains(err.Error(), "setting the mode") {
		t.Errorf("a closed file: want a mode error, got %v", err)
	}

	readOnly, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = readOnly.Close() }()
	if err := fillTemp(readOnly, 0, false, []byte("y")); err == nil || !strings.Contains(err.Error(), "writing temp file") {
		t.Errorf("a file that cannot be written: want a write error, got %v", err)
	}
}
