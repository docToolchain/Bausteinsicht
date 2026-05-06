package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/docToolchain/Bausteinsicht/internal/snapshot"
)

func TestSnapshotListCmd(t *testing.T) {
	tmpDir := t.TempDir()

	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"system": {
				Title: "System",
				Kind:  "system",
			},
		},
		Relationships: []model.Relationship{
			{
				From:  "system",
				To:    "external",
				Label: "communicates",
			},
		},
	}

	manager := snapshot.NewManager(tmpDir)
	snap := snapshot.NewSnapshot("test snapshot", m)
	if err := manager.Save(snap); err != nil {
		t.Fatalf("failed to save snapshot: %v", err)
	}

	cmd := newSnapshotListCmd()
	cmd.SetArgs([]string{"--format", "table"})

	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	originalWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalWd)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := buf.String()
	if len(output) == 0 {
		t.Fatal("expected output, got empty")
	}
	if !bytes.Contains([]byte(output), []byte("ID")) {
		t.Fatal("expected table header")
	}
}

func TestSnapshotListCmdEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := newSnapshotListCmd()
	cmd.SetArgs([]string{})

	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	originalWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalWd)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("No snapshots found")) {
		t.Fatalf("expected 'No snapshots found', got: %s", output)
	}
}

func TestSnapshotListCmdJSON(t *testing.T) {
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
	snap := snapshot.NewSnapshot("json test", m)
	if err := manager.Save(snap); err != nil {
		t.Fatalf("failed to save snapshot: %v", err)
	}

	cmd := newSnapshotListCmd()
	cmd.SetArgs([]string{"--format", "json"})

	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	originalWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalWd)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("\"id\"")) {
		t.Fatalf("expected JSON output, got: %s", output)
	}
}
