package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A relationship to an element that does not exist: invalid for 'validate',
// but harmless for the text exporters when --no-validate is given.
const invalidExportModel = `{
  "specification": {"elements": {"system": {"notation": "System"}}},
  "model": {"shop": {"kind": "system", "title": "Shop"}},
  "relationships": [{"from": "shop", "to": "missing", "label": "calls"}],
  "views": {"context": {"title": "Context", "include": ["shop"]}}
}`

func writeInvalidExportModel(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "architecture.jsonc")
	if err := os.WriteFile(p, []byte(invalidExportModel), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestExportCommands_AbortOnValidationErrors covers #636: every export command
// stops with the validation errors and exit code 1 instead of exporting a model
// that 'validate' rejects.
func TestExportCommands_AbortOnValidationErrors(t *testing.T) {
	modelPath := writeInvalidExportModel(t)
	commands := map[string][]string{
		"export":          {"export", "--output", t.TempDir()},
		"export-diagram":  {"export-diagram", "--view", "context"},
		"export-sequence": {"export-sequence"},
		"export-table":    {"export-table", "--view", "context"},
	}
	for name, args := range commands {
		t.Run(name, func(t *testing.T) {
			out, err := executeRootCmd(append(args, "--model", modelPath)...)
			if err == nil {
				t.Fatalf("expected an error, got none; output:\n%s", out)
			}
			var ee *exitError
			if !errors.As(err, &ee) || ee.code != 1 {
				t.Errorf("expected exit code 1, got %v", err)
			}
			if !strings.Contains(out, `"missing"`) {
				t.Errorf("validation error missing from output:\n%s", out)
			}
			if !strings.Contains(err.Error(), "--no-validate") {
				t.Errorf("error does not mention --no-validate: %v", err)
			}
		})
	}
}

func TestExportCommands_NoValidateExportsAnyway(t *testing.T) {
	modelPath := writeInvalidExportModel(t)
	for _, args := range [][]string{
		{"export-diagram", "--view", "context"},
		{"export-sequence"},
		{"export-table", "--view", "context"},
	} {
		t.Run(args[0], func(t *testing.T) {
			full := append(append([]string{}, args...), "--model", modelPath, "--no-validate")
			if out, err := executeRootCmd(full...); err != nil {
				t.Fatalf("unexpected error with --no-validate: %v\n%s", err, out)
			}
		})
	}
}

func TestExportCommands_ValidModelIsNotAffected(t *testing.T) {
	out, err := executeRootCmd("export-diagram", "--model", writeExportDiagramModel(t), "--view", "context")
	if err != nil {
		t.Fatalf("unexpected error on a valid model: %v", err)
	}
	if strings.Contains(out, "ERROR:") {
		t.Errorf("unexpected validation output on a valid model:\n%s", out)
	}
}
