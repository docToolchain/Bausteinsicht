package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const noViewsModel = `{"specification":{"elements":{"actor":{"notation":"A"}}},"model":{"u":{"kind":"actor","title":"U"}}}`

func noViewsProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile(filepath.Join(dir, "architecture.jsonc"), []byte(noViewsModel), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

// A model without views gives no export; the commands say why instead of printing
// nothing (#737).
func TestExports_SayThatTheModelHasNoViews(t *testing.T) {
	noViewsProject(t)
	for _, args := range [][]string{
		{"export-diagram"},
		{"export-diagram", "--diagram-format", "mermaid"},
		{"export-table"},
		{"export-table", "--combined"},
		{"export-diagram", "--format", "json"},
		{"export-table", "--format", "json"},
	} {
		out, err := executeImportCmd(args...)
		if err != nil {
			t.Errorf("%v: a model without views is not an error, got %v", args, err)
		}
		if !strings.Contains(out, "No views defined in model") || !strings.Contains(out, "bausteinsicht add view") {
			t.Errorf("%v: want the notice \"No views defined in model; add one with 'bausteinsicht add view'\", got %q", args, out)
		}
	}
}

// With a view there is no notice, and Structurizr (the whole workspace, no views
// needed) has none either.
func TestExports_NoNoticeWhenThereIsNothingToSay(t *testing.T) {
	noViewsProject(t)
	if out, err := executeImportCmd("export-diagram", "--diagram-format", "structurizr"); err != nil || strings.Contains(out, "No views defined") {
		t.Errorf("structurizr exports the workspace: %q (%v)", out, err)
	}
	if out, err := executeImportCmd("add", "view", "v", "--title", "V", "--include", "u"); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if out, err := executeImportCmd("export-diagram"); err != nil || strings.Contains(out, "No views defined") {
		t.Errorf("a model with a view has no notice: %q (%v)", out, err)
	}
}

// layout before the first sync says to run sync (#737).
func TestLayout_WithoutADiagramSaysToSync(t *testing.T) {
	dir := noViewsProject(t)
	_, err := executeImportCmd("layout")
	if err == nil || !strings.Contains(err.Error(), "no diagram") || !strings.Contains(err.Error(), "bausteinsicht sync") ||
		strings.Contains(err.Error(), "LoadDocument") {
		t.Errorf("want \"no diagram ...; run 'bausteinsicht sync' first\", got %v", err)
	}

	// A diagram without pages (all views removed) is the same case.
	empty := `<?xml version="1.0" encoding="UTF-8"?><mxfile host="bausteinsicht"></mxfile>`
	if werr := os.WriteFile(filepath.Join(dir, "architecture.drawio"), []byte(empty), 0o600); werr != nil {
		t.Fatal(werr)
	}
	_, err = executeImportCmd("layout")
	if err == nil || !strings.Contains(err.Error(), "no diagram") || !strings.Contains(err.Error(), "bausteinsicht sync") {
		t.Errorf("a diagram without pages: want the same message, got %v", err)
	}
}
