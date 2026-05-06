package main

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/docToolchain/Bausteinsicht/internal/snapshot"
)

func TestSnapshotDiffCmd(t *testing.T) {
	tmpDir := t.TempDir()

	m1 := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"system": {
				Title: "System",
				Kind:  "system",
			},
		},
		Relationships: []model.Relationship{},
	}

	m2 := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"system": {
				Title: "Updated System",
				Kind:  "system",
			},
			"new": {
				Title: "New Element",
				Kind:  "component",
			},
		},
		Relationships: []model.Relationship{},
	}

	manager := snapshot.NewManager(tmpDir)
	snap1 := snapshot.NewSnapshot("first snapshot", m1)
	if err := manager.Save(snap1); err != nil {
		t.Fatalf("failed to save first snapshot: %v", err)
	}

	time.Sleep(1 * time.Second) // Ensure different snapshot IDs

	snap2 := snapshot.NewSnapshot("second snapshot", m2)
	if err := manager.Save(snap2); err != nil {
		t.Fatalf("failed to save second snapshot: %v", err)
	}

	cmd := newSnapshotDiffCmd()
	cmd.SetArgs([]string{snap1.ID, snap2.ID, "--format", "text"})

	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	originalWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalWd)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("new")) {
		t.Fatalf("expected diff output with 'new' element, got: %s", output)
	}
}

func TestSnapshotDiffCmdJSON(t *testing.T) {
	tmpDir := t.TempDir()

	m1 := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"system": {
				Title: "System",
				Kind:  "system",
			},
		},
		Relationships: []model.Relationship{},
	}

	m2 := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"system": {
				Title: "System",
				Kind:  "system",
			},
		},
		Relationships: []model.Relationship{},
	}

	manager := snapshot.NewManager(tmpDir)
	snap1 := snapshot.NewSnapshot("snapshot", m1)
	if err := manager.Save(snap1); err != nil {
		t.Fatalf("failed to save snapshot: %v", err)
	}

	time.Sleep(1 * time.Second) // Ensure different snapshot IDs

	snap2 := snapshot.NewSnapshot("snapshot2", m2)
	if err := manager.Save(snap2); err != nil {
		t.Fatalf("failed to save snapshot: %v", err)
	}

	cmd := newSnapshotDiffCmd()
	cmd.SetArgs([]string{snap1.ID, snap2.ID, "--format", "json"})

	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	originalWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalWd)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("{")) {
		t.Fatalf("expected JSON output, got: %s", output)
	}
}

func TestSnapshotDiffCmdNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := newSnapshotDiffCmd()
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
