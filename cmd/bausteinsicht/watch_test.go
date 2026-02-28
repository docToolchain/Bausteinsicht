package main

import (
	"os"
	"testing"
)

func TestWatchNoModelFile(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"watch"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when no model file exists")
	}
}

func TestWatchCommandRegistered(t *testing.T) {
	cmd := NewRootCmd()
	watchCmd, _, err := cmd.Find([]string{"watch"})
	if err != nil {
		t.Fatalf("watch command not found: %v", err)
	}
	if watchCmd.Use != "watch" {
		t.Errorf("expected Use='watch', got %q", watchCmd.Use)
	}
}

func TestWatchMissingDrawioFile(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Create a minimal model file but no drawio file.
	if err := os.WriteFile("architecture.jsonc", []byte(`{"elements":{},"relationships":[],"views":{}}`), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"watch"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when drawio file does not exist")
	}
}
