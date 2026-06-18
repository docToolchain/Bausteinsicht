package main

import (
	"os"
	"strings"
	"testing"
)

// chdirTemp switches into a fresh temp directory for the duration of the test
// so commands that operate on the current working directory (snapshot list,
// changelog) do not touch the repository.
func chdirTemp(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWd)
	})
}

// TestSnapshotList_DefaultFormatThroughRoot verifies that invoking
// "snapshot list" through the root command with NO format flag succeeds.
// Previously the local default "table" was rejected by the root's global
// --format validation (text|json) — bug #425.
func TestSnapshotList_DefaultFormatThroughRoot(t *testing.T) {
	chdirTemp(t)

	out, err := executeRootCmd("snapshot", "list")
	if err != nil {
		t.Fatalf("snapshot list with default format should succeed, got error: %v", err)
	}
	if !strings.Contains(out, "No snapshots found") {
		t.Fatalf("expected 'No snapshots found', got: %q", out)
	}
}

// TestChangelog_DefaultFormatThroughRoot verifies that invoking "changelog"
// through the root command with NO format flag does not fail with a format
// error. Previously the local default "markdown" was rejected by the root's
// global --format validation (text|json) — bug #425. The command may still
// fail for other reasons (no git history), but never with "unknown format".
func TestChangelog_DefaultFormatThroughRoot(t *testing.T) {
	chdirTemp(t)

	_, err := executeRootCmd("changelog")
	if err != nil && strings.Contains(err.Error(), "unknown format") {
		t.Fatalf("changelog default format must not trip global format validation, got: %v", err)
	}
}
