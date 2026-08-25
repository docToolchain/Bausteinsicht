package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func executeImportCmd(args ...string) (string, error) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return buf.String(), err
}

// absDSL returns the absolute path to a testdata DSL file (avoids .. in paths
// which the security validator rejects).
//
// IMPORTANT: This helper assumes tests run with working directory in cmd/bausteinsicht/.
// Tests must be run with: go test ./cmd/bausteinsicht or make test
func absDSL(t *testing.T, parts ...string) string {
	t.Helper()
	rel := filepath.Join(parts...)
	abs, err := filepath.Abs(filepath.Join("..", "..", rel))
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	return abs
}

func TestImportCmd_Structurizr_DryRun(t *testing.T) {
	dsl := absDSL(t, "internal", "importer", "structurizr", "testdata", "simple.dsl")
	out, err := executeImportCmd("import", "--from", "structurizr", "--dry-run", dsl)
	if err != nil {
		t.Fatalf("expected no error, got %v\nOutput: %s", err, out)
	}
	if !strings.Contains(out, `"model"`) {
		t.Errorf("expected JSON model in output, got: %s", out)
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &m); err != nil {
		t.Errorf("output is not valid JSON: %v\nOutput: %s", err, out)
	}
}

func TestImportCmd_LikeC4_DryRun(t *testing.T) {
	c4 := absDSL(t, "internal", "importer", "likec4", "testdata", "simple.c4")
	out, err := executeImportCmd("import", "--from", "likec4", "--dry-run", c4)
	if err != nil {
		t.Fatalf("expected no error, got %v\nOutput: %s", err, out)
	}
	if !strings.Contains(out, `"model"`) {
		t.Errorf("expected JSON model in output, got: %s", out)
	}
}

func TestImportCmd_OutputFile(t *testing.T) {
	dsl := absDSL(t, "internal", "importer", "structurizr", "testdata", "simple.dsl")
	outFile := filepath.Join(t.TempDir(), "architecture.jsonc")
	out, err := executeImportCmd("import", "--from", "structurizr", "--output", outFile, dsl)
	if err != nil {
		t.Fatalf("expected no error, got %v\nOutput: %s", err, out)
	}
	if !strings.Contains(out, "Imported model written to") {
		t.Errorf("expected success message, got: %s", out)
	}
	if _, err := os.Stat(outFile); err != nil {
		t.Errorf("output file not created: %v", err)
	}
}

func TestImportCmd_OutputExists_ExitCode2(t *testing.T) {
	dsl := absDSL(t, "internal", "importer", "structurizr", "testdata", "simple.dsl")
	outFile := filepath.Join(t.TempDir(), "architecture.jsonc")
	if err := os.WriteFile(outFile, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := executeImportCmd("import", "--from", "structurizr", "--output", outFile, dsl)
	if err == nil {
		t.Fatal("expected error for existing output file")
	}
	ee, ok := err.(*exitError)
	if !ok {
		t.Fatalf("expected exitError, got %T: %v", err, err)
	}
	if ee.code != 2 {
		t.Errorf("expected exit code 2, got %d", ee.code)
	}
}

func TestImportCmd_Force_OverwritesFile(t *testing.T) {
	dsl := absDSL(t, "internal", "importer", "structurizr", "testdata", "simple.dsl")
	outFile := filepath.Join(t.TempDir(), "architecture.jsonc")
	if err := os.WriteFile(outFile, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := executeImportCmd("import", "--from", "structurizr", "--output", outFile, "--force", dsl)
	if err != nil {
		t.Fatalf("expected no error with --force, got %v", err)
	}
}

func TestImportCmd_UnknownFormat_ExitCode1(t *testing.T) {
	_, err := executeImportCmd("import", "--from", "unknown", "--dry-run", "anyfile.dsl")
	if err == nil {
		t.Fatal("expected error for unknown format")
	}
	ee, ok := err.(*exitError)
	if !ok {
		t.Fatalf("expected exitError, got %T: %v", err, err)
	}
	if ee.code != 1 {
		t.Errorf("expected exit code 1, got %d", ee.code)
	}
}

func TestImportCmd_NonExistentFile_ExitCode1(t *testing.T) {
	_, err := executeImportCmd("import", "--from", "structurizr", "--dry-run", "/nonexistent/model.dsl")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
	ee, ok := err.(*exitError)
	if !ok {
		t.Fatalf("expected exitError, got %T: %v", err, err)
	}
	if ee.code != 1 {
		t.Errorf("expected exit code 1, got %d", ee.code)
	}
}

// TestImportCmd_TextSummary covers the post-import summary text (#482 v1):
// per-kind counts, views count, and the fresh-import next-steps hint.
func TestImportCmd_TextSummary(t *testing.T) {
	dsl := absDSL(t, "internal", "importer", "structurizr", "testdata", "simple.dsl")
	outFile := filepath.Join(t.TempDir(), "architecture.jsonc")
	out, err := executeImportCmd("import", "--from", "structurizr", "--output", outFile, dsl)
	if err != nil {
		t.Fatalf("expected no error, got %v\nOutput: %s", err, out)
	}
	for _, want := range []string{
		"person: 1",
		"container: 3",
		"system: 2",
		"views: 2",
		"Next steps: bausteinsicht sync, bausteinsicht export diagram",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected summary to contain %q, got:\n%s", want, out)
		}
	}
}

// TestImportCmd_JSONSummary covers --json's machine-readable summary shape.
func TestImportCmd_JSONSummary(t *testing.T) {
	dsl := absDSL(t, "internal", "importer", "structurizr", "testdata", "simple.dsl")
	outFile := filepath.Join(t.TempDir(), "architecture.jsonc")
	out, err := executeImportCmd("import", "--from", "structurizr", "--output", outFile, "--json", dsl)
	if err != nil {
		t.Fatalf("expected no error, got %v\nOutput: %s", err, out)
	}

	var summary struct {
		Elements   map[string]int `json:"elements"`
		OutputPath string         `json:"outputPath"`
		Views      int            `json:"views"`
		Warnings   []string       `json:"warnings"`
	}
	if err := json.Unmarshal([]byte(out), &summary); err != nil {
		t.Fatalf("--json output is not valid JSON: %v\noutput:\n%s", err, out)
	}
	if summary.Elements["container"] != 3 {
		t.Errorf("expected 3 containers, got %d (elements=%v)", summary.Elements["container"], summary.Elements)
	}
	if summary.OutputPath != outFile {
		t.Errorf("expected outputPath %q, got %q", outFile, summary.OutputPath)
	}
	if summary.Views != 2 {
		t.Errorf("expected views=2, got %d", summary.Views)
	}
	if summary.Warnings == nil {
		t.Error("expected warnings to be a non-nil (possibly empty) array in JSON output")
	}
}

// TestImportCmd_DryRunSuppressesSummary covers that --dry-run prints only the
// model JSON — no per-kind counts, no next-steps hint, since nothing was
// persisted.
func TestImportCmd_DryRunSuppressesSummary(t *testing.T) {
	dsl := absDSL(t, "internal", "importer", "structurizr", "testdata", "simple.dsl")
	out, err := executeImportCmd("import", "--from", "structurizr", "--dry-run", dsl)
	if err != nil {
		t.Fatalf("expected no error, got %v\nOutput: %s", err, out)
	}
	if strings.Contains(out, "Next steps:") {
		t.Errorf("--dry-run must not print a next-steps hint, got:\n%s", out)
	}
	if strings.Contains(out, "container:") {
		t.Errorf("--dry-run must not print per-kind counts, got:\n%s", out)
	}
}

// TestImportCmd_ForceOverwriteHintsDiff covers that overwriting an existing
// output file with --force hints at `bausteinsicht diff` instead of `sync`.
func TestImportCmd_ForceOverwriteHintsDiff(t *testing.T) {
	dsl := absDSL(t, "internal", "importer", "structurizr", "testdata", "simple.dsl")
	outFile := filepath.Join(t.TempDir(), "architecture.jsonc")
	if err := os.WriteFile(outFile, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := executeImportCmd("import", "--from", "structurizr", "--output", outFile, "--force", dsl)
	if err != nil {
		t.Fatalf("expected no error with --force, got %v", err)
	}
	if !strings.Contains(out, "Next steps: bausteinsicht diff") {
		t.Errorf("expected --force overwrite to hint at \"bausteinsicht diff\", got:\n%s", out)
	}
	if strings.Contains(out, "bausteinsicht sync") {
		t.Errorf("--force overwrite must not also hint at \"bausteinsicht sync\", got:\n%s", out)
	}
}

// TestImportCmd_XMINoViewsLine covers that the views line is omitted in text
// mode when the importer produced zero views (XMI never populates Views).
func TestImportCmd_XMINoViewsLine(t *testing.T) {
	xmi := absDSL(t, "internal", "importer", "xmi", "testdata", "basic.xmi")
	outFile := filepath.Join(t.TempDir(), "architecture.jsonc")
	out, err := executeImportCmd("import", "--from", "xmi", "--output", outFile, xmi)
	if err != nil {
		t.Fatalf("expected no error, got %v\nOutput: %s", err, out)
	}
	if strings.Contains(out, "views:") {
		t.Errorf("expected no \"views:\" line for an XMI import with zero views, got:\n%s", out)
	}
}
