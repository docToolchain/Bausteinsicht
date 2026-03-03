package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const exportDiagramTestModel = `{
  "specification": {
    "elements": {
      "system": {"notation": "System", "container": true},
      "container": {"notation": "Container"},
      "actor": {"notation": "Actor"},
      "external_system": {"notation": "External System"}
    }
  },
  "model": {
    "user": {"kind": "actor", "title": "User", "description": "End user"},
    "shop": {"kind": "system", "title": "Shop", "description": "E-commerce", "children": {
      "api": {"kind": "container", "title": "API", "description": "REST", "technology": "Go"}
    }},
    "ext": {"kind": "external_system", "title": "External", "description": "Third party"}
  },
  "relationships": [
    {"from": "user", "to": "shop", "label": "uses", "kind": "uses"}
  ],
  "views": {
    "context": {
      "title": "System Context",
      "include": ["user", "shop", "ext"]
    },
    "containers": {
      "title": "Container View",
      "scope": "shop",
      "include": ["user", "shop.*"]
    }
  }
}`

func writeExportDiagramModel(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "architecture.jsonc")
	if err := os.WriteFile(p, []byte(exportDiagramTestModel), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestExportDiagram_PlantUMLToStdout(t *testing.T) {
	modelPath := writeExportDiagramModel(t)
	out, err := executeRootCmd("export-diagram", "--model", modelPath, "--view", "context", "--diagram-format", "plantuml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "@startuml") {
		t.Error("expected @startuml in output")
	}
	if !strings.Contains(out, "Person(") {
		t.Error("expected Person() macro")
	}
}

func TestExportDiagram_MermaidToStdout(t *testing.T) {
	modelPath := writeExportDiagramModel(t)
	out, err := executeRootCmd("export-diagram", "--model", modelPath, "--view", "context", "--diagram-format", "mermaid")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "C4Context") {
		t.Error("expected C4Context in output")
	}
}

func TestExportDiagram_WriteToFile(t *testing.T) {
	modelPath := writeExportDiagramModel(t)
	outDir := t.TempDir()
	_, err := executeRootCmd("export-diagram", "--model", modelPath, "--view", "context", "--diagram-format", "plantuml", "--output", outDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	outPath := filepath.Join(outDir, "context.puml")
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("expected output file: %v", err)
	}
	if !strings.Contains(string(data), "@startuml") {
		t.Error("expected @startuml in file")
	}
}

func TestExportDiagram_MermaidFile(t *testing.T) {
	modelPath := writeExportDiagramModel(t)
	outDir := t.TempDir()
	_, err := executeRootCmd("export-diagram", "--model", modelPath, "--view", "context", "--diagram-format", "mermaid", "--output", outDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	outPath := filepath.Join(outDir, "context.mmd")
	if _, err := os.ReadFile(outPath); err != nil {
		t.Fatalf("expected .mmd output file: %v", err)
	}
}

func TestExportDiagram_InvalidFormat(t *testing.T) {
	modelPath := writeExportDiagramModel(t)
	_, err := executeRootCmd("export-diagram", "--model", modelPath, "--diagram-format", "dot")
	if err == nil {
		t.Error("expected error for invalid format")
	}
}

func TestExportDiagram_InvalidView(t *testing.T) {
	modelPath := writeExportDiagramModel(t)
	_, err := executeRootCmd("export-diagram", "--model", modelPath, "--view", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent view")
	}
}
