package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const exportTableTestModel = `{
  "specification": {
    "elements": {
      "system": {"notation": "System", "container": true},
      "container": {"notation": "Container"},
      "actor": {"notation": "Actor"}
    }
  },
  "model": {
    "user": {"kind": "actor", "title": "User", "description": "End user"},
    "shop": {"kind": "system", "title": "Shop", "description": "E-commerce", "children": {
      "api": {"kind": "container", "title": "API", "description": "REST backend", "technology": "Go"}
    }}
  },
  "relationships": [],
  "views": {
    "context": {
      "title": "System Context",
      "include": ["user", "shop"]
    },
    "containers": {
      "title": "Container View",
      "scope": "shop",
      "include": ["user", "shop.*"]
    }
  }
}`

func writeExportTableModel(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "architecture.jsonc")
	if err := os.WriteFile(p, []byte(exportTableTestModel), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestExportTable_AsciiDocToStdout(t *testing.T) {
	modelPath := writeExportTableModel(t)

	out, err := executeRootCmd("export-table", "--model", modelPath, "--view", "containers", "--table-format", "adoc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "|===") {
		t.Error("expected AsciiDoc table delimiter")
	}
	if !strings.Contains(out, "API") {
		t.Error("expected element 'API' in output")
	}
	if !strings.Contains(out, "Container View") {
		t.Error("expected view title")
	}
}

func TestExportTable_MarkdownToStdout(t *testing.T) {
	modelPath := writeExportTableModel(t)

	out, err := executeRootCmd("export-table", "--model", modelPath, "--view", "context", "--table-format", "md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "| Element") {
		t.Error("expected Markdown table header")
	}
	if !strings.Contains(out, "Shop") {
		t.Error("expected element 'Shop' in output")
	}
}

func TestExportTable_AllViewsToStdout(t *testing.T) {
	modelPath := writeExportTableModel(t)

	out, err := executeRootCmd("export-table", "--model", modelPath, "--table-format", "adoc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "System Context") {
		t.Error("expected 'System Context' view title")
	}
	if !strings.Contains(out, "Container View") {
		t.Error("expected 'Container View' view title")
	}
}

func TestExportTable_CombinedToStdout(t *testing.T) {
	modelPath := writeExportTableModel(t)

	out, err := executeRootCmd("export-table", "--model", modelPath, "--combined", "--table-format", "adoc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "All Elements") {
		t.Error("expected 'All Elements' title")
	}
	if !strings.Contains(out, "API") {
		t.Error("expected 'API' in combined output")
	}
}

func TestExportTable_WriteToFile(t *testing.T) {
	modelPath := writeExportTableModel(t)
	outDir := t.TempDir()

	_, err := executeRootCmd("export-table", "--model", modelPath, "--view", "containers", "--table-format", "adoc", "--output", outDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outPath := filepath.Join(outDir, "containers-elements.adoc")
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("expected output file: %v", err)
	}
	if !strings.Contains(string(data), "API") {
		t.Error("expected 'API' in output file")
	}
}

func TestExportTable_InvalidView(t *testing.T) {
	modelPath := writeExportTableModel(t)

	_, err := executeRootCmd("export-table", "--model", modelPath, "--view", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent view")
	}
}

func TestExportTable_InvalidFormat(t *testing.T) {
	modelPath := writeExportTableModel(t)

	_, err := executeRootCmd("export-table", "--model", modelPath, "--table-format", "csv")
	if err == nil {
		t.Error("expected error for invalid table format")
	}
}
