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
