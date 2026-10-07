package main

import (
	"bytes"
	"encoding/json"
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
	_, err := executeRootCmd("export-diagram", "--model", modelPath, "--diagram-format", "invalid")
	if err == nil {
		t.Error("expected error for invalid format")
	}
}

func TestExportDiagram_DOTFormat(t *testing.T) {
	modelPath := writeExportDiagramModel(t)
	out, err := executeRootCmd("export-diagram", "--model", modelPath, "--view", "context", "--diagram-format", "dot")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "digraph") {
		t.Error("expected 'digraph' in DOT output")
	}
	if !strings.Contains(out, "rankdir=LR") {
		t.Error("expected 'rankdir=LR' in DOT output")
	}
}

func TestExportDiagram_D2Format(t *testing.T) {
	modelPath := writeExportDiagramModel(t)
	out, err := executeRootCmd("export-diagram", "--model", modelPath, "--view", "context", "--diagram-format", "d2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "direction: right") {
		t.Error("expected 'direction: right' in D2 output")
	}
}

func TestExportDiagram_HTMLFormat(t *testing.T) {
	modelPath := writeExportDiagramModel(t)
	out, err := executeRootCmd("export-diagram", "--model", modelPath, "--view", "context", "--diagram-format", "html")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "<!DOCTYPE html>") {
		t.Error("expected HTML5 doctype")
	}
	if !strings.Contains(out, "DIAGRAM_DATA") {
		t.Error("expected embedded diagram data")
	}
}

func TestExportDiagram_InvalidView(t *testing.T) {
	modelPath := writeExportDiagramModel(t)
	_, err := executeRootCmd("export-diagram", "--model", modelPath, "--view", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent view")
	}
}

func TestExportDiagram_StructurizrToStdout(t *testing.T) {
	modelPath := writeExportDiagramModel(t)
	out, err := executeRootCmd("export-diagram", "--model", modelPath, "--diagram-format", "structurizr")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "workspace {") {
		t.Error("expected 'workspace {' in structurizr output")
	}
	if !strings.Contains(out, "model {") {
		t.Error("expected 'model {' in structurizr output")
	}
	if !strings.Contains(out, "views {") {
		t.Error("expected 'views {' in structurizr output")
	}
}

func TestExportDiagram_StructurizrToFile(t *testing.T) {
	modelPath := writeExportDiagramModel(t)
	outDir := t.TempDir()
	_, err := executeRootCmd("export-diagram", "--model", modelPath, "--diagram-format", "structurizr", "--output", outDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	dslPath := filepath.Join(outDir, "workspace.dsl")
	data, err := os.ReadFile(dslPath)
	if err != nil {
		t.Fatalf("expected workspace.dsl to be written: %v", err)
	}
	if !strings.Contains(string(data), "workspace {") {
		t.Errorf("workspace.dsl missing 'workspace {': %s", data)
	}
}

// TestExportDiagram_JSONWithOutput_* (#631): --output + --format json must write
// files to disk and report an absolute "path" (not "source") in the JSON array.
// Uses separate stdout/stderr buffers to avoid stderr warnings polluting JSON parse.

func runExportDiagramJSON(t *testing.T, extraArgs ...string) []map[string]interface{} {
	t.Helper()
	var outBuf, errBuf bytes.Buffer
	root := NewRootCmd()
	root.SetOut(&outBuf)
	root.SetErr(&errBuf)
	args := append([]string{"export-diagram"}, extraArgs...)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, errBuf.String())
	}
	var entries []map[string]interface{}
	if err := json.Unmarshal(outBuf.Bytes(), &entries); err != nil {
		t.Fatalf("invalid JSON: %v\nstdout:\n%s\nstderr:\n%s", err, outBuf.String(), errBuf.String())
	}
	return entries
}

func assertJSONPathEntries(t *testing.T, entries []map[string]interface{}) {
	t.Helper()
	if len(entries) == 0 {
		t.Fatal("expected at least one entry in JSON output")
	}
	for _, e := range entries {
		p, ok := e["path"].(string)
		if !ok || p == "" {
			t.Errorf("expected non-empty 'path' field in entry, got: %v", e)
			continue
		}
		if !filepath.IsAbs(p) {
			t.Errorf("expected absolute path, got relative: %q", p)
		}
		if _, err := os.ReadFile(p); err != nil {
			t.Errorf("expected file to exist at path %q: %v", p, err)
		}
	}
	for i, e := range entries {
		if _, hasSource := e["source"]; hasSource {
			t.Errorf("entry[%d]: expected no 'source' field when --output is set", i)
		}
	}
}

func TestExportDiagram_JSONWithOutput_PlantUML(t *testing.T) {
	modelPath := writeExportDiagramModel(t)
	outDir := t.TempDir()
	entries := runExportDiagramJSON(t,
		"--model", modelPath, "--diagram-format", "plantuml", "--output", outDir, "--format", "json")
	assertJSONPathEntries(t, entries)
}

func TestExportDiagram_JSONWithOutput_Mermaid(t *testing.T) {
	modelPath := writeExportDiagramModel(t)
	outDir := t.TempDir()
	entries := runExportDiagramJSON(t,
		"--model", modelPath, "--diagram-format", "mermaid", "--output", outDir, "--format", "json")
	assertJSONPathEntries(t, entries)
}

func TestExportDiagram_JSONWithOutput_DOT(t *testing.T) {
	modelPath := writeExportDiagramModel(t)
	outDir := t.TempDir()
	entries := runExportDiagramJSON(t,
		"--model", modelPath, "--diagram-format", "dot", "--output", outDir, "--format", "json")
	assertJSONPathEntries(t, entries)
}

func TestExportDiagram_JSONWithOutput_HTML(t *testing.T) {
	modelPath := writeExportDiagramModel(t)
	outDir := t.TempDir()
	entries := runExportDiagramJSON(t,
		"--model", modelPath, "--diagram-format", "html", "--output", outDir, "--format", "json")
	assertJSONPathEntries(t, entries)
}
