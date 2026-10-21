package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSchemaGenerate_Codegraph(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "codegraph.schema.json")

	if _, err := executeImportCmd("schema", "generate", "--type", "codegraph", "--output", out); err != nil {
		t.Fatalf("schema generate --type codegraph failed: %v", err)
	}

	data, err := os.ReadFile(out) //nolint:gosec // test-controlled path
	if err != nil {
		t.Fatalf("reading generated schema: %v", err)
	}
	if !strings.Contains(string(data), "Bausteinsicht CodeGraph") {
		t.Errorf("expected codegraph schema title in output, got: %s", data)
	}
}

func TestSchemaGenerate_InvalidType(t *testing.T) {
	_, err := executeImportCmd("schema", "generate", "--type", "bogus")
	if err == nil {
		t.Error("expected error for invalid --type")
	}
}
