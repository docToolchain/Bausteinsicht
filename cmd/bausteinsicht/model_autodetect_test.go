package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/templates"
)

// autodetectCommands are the commands that took no part in the auto-detection
// of the model file before #692 (or fell back to architecture.jsonc).
var autodetectCommands = map[string][]string{
	"health":            {"health"},
	"graph":             {"graph"},
	"status":            {"status"},
	"stale":             {"stale"},
	"diff":              {"diff"},
	"changelog":         {"changelog", "--since", "v0"},
	"snapshot save":     {"snapshot", "save", "x"},
	"snapshot diff":     {"snapshot", "diff", "x"},
	"overlay apply":     {"overlay", "apply", "m.json"},
	"overlay remove":    {"overlay", "remove"},
	"adr list":          {"adr", "list"},
	"adr show":          {"adr", "show", "ADR-1"},
	"validate (shared)": {"validate"},
}

// With a model of any name in the working directory, no command complains
// about a missing model file; without one, each exits with code 2 and names
// what it looked for (#692).
func TestCommandsFindTheModelThemselves(t *testing.T) {
	for name, args := range autodetectCommands {
		t.Run(name+"/found", func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "my-model.jsonc"), templates.SampleModel, 0o644); err != nil {
				t.Fatal(err)
			}
			t.Chdir(dir)
			if name == "snapshot diff" {
				args = []string{"snapshot", "diff", savedSnapshotID(t, "")}
			}
			_, err := executeRootCmd(args...)
			if err != nil && strings.Contains(err.Error(), "no model file found") {
				t.Errorf("the model file was not detected: %v", err)
			}
		})
		t.Run(name+"/none", func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			if name == "snapshot diff" {
				// The snapshot must exist for the command to get to the model.
				model := filepath.Join(dir, "m.jsonc")
				if err := os.WriteFile(model, templates.SampleModel, 0o644); err != nil {
					t.Fatal(err)
				}
				args = []string{"snapshot", "diff", savedSnapshotID(t, model)}
				if err := os.Remove(model); err != nil {
					t.Fatal(err)
				}
			}
			_, err := executeRootCmd(args...)
			if err == nil {
				t.Fatal("expected an error without a model file")
			}
			for _, want := range []string{"no model file found", "*.jsonc", "--model"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q lacks %q", err.Error(), want)
				}
			}
			if code := exitCodeOf(t, err); code != 2 {
				t.Errorf("exit code %d, want 2", code)
			}
		})
	}
}

// savedSnapshotID saves a snapshot of the model (the one in the working
// directory if model is empty) and returns its ID.
func savedSnapshotID(t *testing.T, model string) string {
	t.Helper()
	args := []string{"snapshot", "save"}
	if model != "" {
		args = append(args, "--model", model)
	}
	out, err := executeRootCmd(args...)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(out, "\n") {
		if id, ok := strings.CutPrefix(line, "Snapshot saved: "); ok {
			return strings.TrimSpace(id)
		}
	}
	t.Fatalf("no snapshot ID in %q", out)
	return ""
}

func TestResolveModelPath(t *testing.T) {
	if got, err := resolveModelPath("given.jsonc"); err != nil || got != "given.jsonc" {
		t.Errorf("a given path is used as it is: %q, %v", got, err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "only.jsonc"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	if got, err := resolveModelPath(""); err != nil || filepath.Base(got) != "only.jsonc" {
		t.Errorf("the single *.jsonc file is detected: %q, %v", got, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "second.jsonc"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveModelPath(""); err == nil || !strings.Contains(err.Error(), "multiple .jsonc files") {
		t.Errorf("two files must be refused: %v", err)
	}
}
