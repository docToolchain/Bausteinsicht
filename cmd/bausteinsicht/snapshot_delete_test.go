package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/docToolchain/Bausteinsicht/internal/snapshot"
)

func TestSnapshotDeleteCmd(t *testing.T) {
	tmpDir := t.TempDir()

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
	snap := snapshot.NewSnapshot("snapshot to delete", m)
	if err := manager.Save(snap); err != nil {
		t.Fatalf("failed to save snapshot: %v", err)
	}

	snapID := snap.ID
	if !manager.Exists(snapID) {
		t.Fatal("snapshot should exist after saving")
	}

	cmd := newSnapshotDeleteCmd()
	cmd.SetArgs([]string{snapID})

	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	originalWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalWd)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	if manager.Exists(snapID) {
		t.Fatal("snapshot should be deleted")
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Snapshot deleted")) {
		t.Fatalf("expected deletion confirmation, got: %s", output)
	}
}

func TestSnapshotDeleteCmdNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := newSnapshotDeleteCmd()
	cmd.SetArgs([]string{"nonexistent-snapshot"})

	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	originalWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalWd)

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for nonexistent snapshot")
	}
}
