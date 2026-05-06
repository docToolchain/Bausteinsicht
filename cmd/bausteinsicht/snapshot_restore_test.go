package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/docToolchain/Bausteinsicht/internal/snapshot"
)

func TestSnapshotRestoreCmd(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := tmpDir + "/restored.jsonc"

	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"system": {
				Title: "System",
				Kind:  "system",
			},
		},
		Relationships: []model.Relationship{},
	}

	manager := snapshot.NewManager(tmpDir)
	snap := snapshot.NewSnapshot("snapshot to restore", m)
	if err := manager.Save(snap); err != nil {
		t.Fatalf("failed to save snapshot: %v", err)
	}

	cmd := newSnapshotRestoreCmd()
	cmd.SetArgs([]string{snap.ID, outputFile})

	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	originalWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalWd)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	// Check if output file was created
	if _, err := os.Stat(outputFile); err != nil {
		t.Fatalf("output file not created: %v", err)
	}

	// Verify the content
	restoredModel, err := model.Load(outputFile)
	if err != nil {
		t.Fatalf("failed to load restored model: %v", err)
	}

	if len(restoredModel.Model) != 1 {
		t.Fatalf("expected 1 element, got %d", len(restoredModel.Model))
	}

	elem, ok := restoredModel.Model["system"]
	if !ok {
		t.Fatal("system element not found in restored model")
	}

	if elem.Title != "System" {
		t.Fatalf("expected title 'System', got %q", elem.Title)
	}
}

func TestSnapshotRestoreCmdOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := tmpDir + "/restored.jsonc"

	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"system": {
				Title: "System",
				Kind:  "system",
			},
		},
		Relationships: []model.Relationship{},
	}

	manager := snapshot.NewManager(tmpDir)
	snap := snapshot.NewSnapshot("snapshot", m)
	if err := manager.Save(snap); err != nil {
		t.Fatalf("failed to save snapshot: %v", err)
	}

	// Create existing file
	if err := os.WriteFile(outputFile, []byte("existing content"), 0o644); err != nil {
		t.Fatalf("failed to create existing file: %v", err)
	}

	// Try to restore without --force (should fail)
	cmd := newSnapshotRestoreCmd()
	cmd.SetArgs([]string{snap.ID, outputFile})

	originalWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalWd)

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error when file exists without --force")
	}

	// Try with --force (should succeed)
	cmd = newSnapshotRestoreCmd()
	cmd.SetArgs([]string{snap.ID, outputFile, "--force"})

	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed with --force: %v", err)
	}
}

func TestSnapshotRestoreCmdNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := newSnapshotRestoreCmd()
	cmd.SetArgs([]string{"nonexistent-snapshot", "/tmp/output.jsonc"})

	originalWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalWd)

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for nonexistent snapshot")
	}
}
