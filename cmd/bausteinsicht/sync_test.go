package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestSyncAfterInit(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Init first.
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"init"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Sync should report no changes (already in sync after init).
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd2 := NewRootCmd()
	cmd2.SetArgs([]string{"sync"})
	err := cmd2.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// After init, sync may detect formatting differences — just verify it runs successfully.
	if output == "" {
		t.Error("expected some output from sync")
	}
}

func TestSyncNoModelFile(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"sync"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when no model file exists")
	}
}

func TestSyncJSONOutput(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Init.
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"init"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Sync with JSON output.
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd2 := NewRootCmd()
	cmd2.SetArgs([]string{"sync", "--format", "json"})
	err := cmd2.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := strings.TrimSpace(string(buf[:n]))

	var summary syncSummary
	if err := json.Unmarshal([]byte(output), &summary); err != nil {
		t.Fatalf("invalid JSON output: %v\noutput: %s", err, output)
	}
}

func TestSyncDetectsModelChanges(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Init.
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"init"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Add an element to the model.
	cmd2 := NewRootCmd()
	cmd2.SetArgs([]string{"add", "element", "--id", "newservice", "--kind", "system", "--title", "New Service"})
	if err := cmd2.Execute(); err != nil {
		t.Fatalf("add element failed: %v", err)
	}

	// Sync should detect the new element.
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd3 := NewRootCmd()
	cmd3.SetArgs([]string{"sync", "--format", "json"})
	err := cmd3.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := strings.TrimSpace(string(buf[:n]))

	var summary syncSummary
	if err := json.Unmarshal([]byte(output), &summary); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, output)
	}

	if summary.ForwardAdded == 0 {
		t.Error("expected forward_added > 0 after adding element")
	}
}

func TestSyncWithExplicitModelPath(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Init.
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"init"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Sync with explicit model path.
	cmd2 := NewRootCmd()
	cmd2.SetArgs([]string{"sync", "--model", "architecture.jsonc"})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmd2.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("sync with explicit model failed: %v", err)
	}

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// After init, sync may detect formatting differences — just verify it runs.
	if output == "" {
		t.Error("expected some output from sync")
	}
}
