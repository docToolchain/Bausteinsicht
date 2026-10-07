package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/export"
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

func assertJSONPathEntries(t *testing.T, entries []map[string]interface{}, expectedFormat string) {
	t.Helper()
	if len(entries) == 0 {
		t.Fatal("expected at least one entry in JSON output")
	}
	for i, e := range entries {
		if v, ok := e["view"].(string); !ok || v == "" {
			t.Errorf("entry[%d]: expected non-empty 'view' field, got: %v", i, e)
		}
		if f, ok := e["format"].(string); !ok || f != expectedFormat {
			t.Errorf("entry[%d]: expected format=%q, got: %v", i, expectedFormat, e["format"])
		}
		p, ok := e["path"].(string)
		if !ok || p == "" {
			t.Errorf("entry[%d]: expected non-empty 'path' field, got: %v", i, e)
			continue
		}
		if !filepath.IsAbs(p) {
			t.Errorf("entry[%d]: expected absolute path, got relative: %q", i, p)
		}
		if _, err := os.ReadFile(p); err != nil {
			t.Errorf("entry[%d]: expected file to exist at path %q: %v", i, p, err)
		}
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
	assertJSONPathEntries(t, entries, "plantuml")
}

func TestExportDiagram_JSONWithOutput_Mermaid(t *testing.T) {
	modelPath := writeExportDiagramModel(t)
	outDir := t.TempDir()
	entries := runExportDiagramJSON(t,
		"--model", modelPath, "--diagram-format", "mermaid", "--output", outDir, "--format", "json")
	assertJSONPathEntries(t, entries, "mermaid")
}

func TestExportDiagram_JSONWithOutput_DOT(t *testing.T) {
	modelPath := writeExportDiagramModel(t)
	outDir := t.TempDir()
	entries := runExportDiagramJSON(t,
		"--model", modelPath, "--diagram-format", "dot", "--output", outDir, "--format", "json")
	assertJSONPathEntries(t, entries, "dot")
	// DOT files must keep the "architecture-" prefix to match non-JSON mode.
	for i, e := range entries {
		p := e["path"].(string)
		view := e["view"].(string)
		expected := "architecture-" + export.SafeViewKey(view) + ".dot"
		if filepath.Base(p) != expected {
			t.Errorf("entry[%d]: expected filename %q, got %q", i, expected, filepath.Base(p))
		}
	}
}

func TestExportDiagram_JSONWithOutput_HTML(t *testing.T) {
	modelPath := writeExportDiagramModel(t)
	outDir := t.TempDir()
	entries := runExportDiagramJSON(t,
		"--model", modelPath, "--diagram-format", "html", "--output", outDir, "--format", "json")
	assertJSONPathEntries(t, entries, "html")
}

func TestExportDiagram_JSONWithOutput_D2(t *testing.T) {
	modelPath := writeExportDiagramModel(t)
	outDir := t.TempDir()
	entries := runExportDiagramJSON(t,
		"--model", modelPath, "--diagram-format", "d2", "--output", outDir, "--format", "json")
	assertJSONPathEntries(t, entries, "d2")
	// D2 files must keep the "architecture-" prefix to match non-JSON mode.
	for i, e := range entries {
		p := e["path"].(string)
		view := e["view"].(string)
		expected := "architecture-" + export.SafeViewKey(view) + ".d2"
		if filepath.Base(p) != expected {
			t.Errorf("entry[%d]: expected filename %q, got %q", i, expected, filepath.Base(p))
		}
	}
}

// TestExportDiagram_JSONWithOutput_Structurizr covers the structurizr JSON path
// (source-mode without --output, path-mode with --output).
func TestExportDiagram_JSONWithOutput_Structurizr(t *testing.T) {
	modelPath := writeExportDiagramModel(t)
	outDir := t.TempDir()

	// path-mode
	entries := runExportDiagramJSON(t,
		"--model", modelPath, "--diagram-format", "structurizr", "--output", outDir, "--format", "json")
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry for structurizr, got %d", len(entries))
	}
	e := entries[0]
	if v, ok := e["view"].(string); !ok || v != "workspace" {
		t.Errorf("expected view=workspace, got: %v", e["view"])
	}
	if f, ok := e["format"].(string); !ok || f != "structurizr" {
		t.Errorf("expected format=structurizr, got: %v", e["format"])
	}
	p, ok := e["path"].(string)
	if !ok || p == "" {
		t.Errorf("expected non-empty 'path', got: %v", e)
	}
	if !filepath.IsAbs(p) {
		t.Errorf("expected absolute path, got: %q", p)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		t.Errorf("expected file at %q: %v", p, err)
	} else if len(data) == 0 {
		t.Errorf("structurizr file at %q is empty", p)
	}
	if _, hasSource := e["source"]; hasSource {
		t.Error("expected no 'source' field in path-mode")
	}

	// source-mode (no --output)
	srcEntries := runExportDiagramJSON(t,
		"--model", modelPath, "--diagram-format", "structurizr", "--format", "json")
	if len(srcEntries) != 1 {
		t.Fatalf("expected 1 source-mode entry, got %d", len(srcEntries))
	}
	src, ok := srcEntries[0]["source"].(string)
	if !ok || src == "" {
		t.Errorf("expected non-empty 'source' in source-mode, got: %v", srcEntries[0])
	}
	if _, hasPath := srcEntries[0]["path"]; hasPath {
		t.Error("expected no 'path' field in source-mode")
	}
}

func TestExportDiagram_JSONSourceMode_AllFormats(t *testing.T) {
	modelPath := writeExportDiagramModel(t)
	for _, f := range []string{"mermaid", "dot", "d2", "html", "structurizr"} {
		t.Run(f, func(t *testing.T) {
			entries := runExportDiagramJSON(t, "--model", modelPath, "--diagram-format", f, "--format", "json")
			if len(entries) == 0 {
				t.Fatal("expected at least one entry")
			}
			for i, e := range entries {
				if s, ok := e["source"].(string); !ok || s == "" {
					t.Errorf("entry[%d]: expected non-empty 'source', got: %v", i, e)
				}
				if _, hasPath := e["path"]; hasPath {
					t.Errorf("entry[%d]: unexpected 'path' in source-mode", i)
				}
			}
		})
	}
}

// TestExportDiagram_JSONSourceMode verifies that --format json without --output
// emits "source" (not "path") for each entry — backward-compat source mode.
func TestExportDiagram_JSONSourceMode(t *testing.T) {
	modelPath := writeExportDiagramModel(t)
	entries := runExportDiagramJSON(t, "--model", modelPath, "--diagram-format", "plantuml", "--format", "json")
	if len(entries) == 0 {
		t.Fatal("expected at least one entry")
	}
	for i, e := range entries {
		src, ok := e["source"].(string)
		if !ok {
			t.Errorf("entry[%d]: expected 'source' field in source-mode, got: %v", i, e)
		}
		if src == "" {
			t.Errorf("entry[%d]: expected non-empty 'source' content", i)
		}
		if _, hasPath := e["path"]; hasPath {
			t.Errorf("entry[%d]: unexpected 'path' field when --output is not set", i)
		}
	}
}

func TestEmitExportItems_FilenameCollision(t *testing.T) {
	outDir := t.TempDir()
	var outBuf bytes.Buffer
	root := NewRootCmd()
	root.SetOut(&outBuf)
	items := []exportItem{
		{viewKey: "a/b", filename: "b.puml", content: "x"},
		{viewKey: "b", filename: "b.puml", content: "y"},
	}
	if err := emitExportItems(root, "plantuml", outDir, items); err == nil {
		t.Fatal("expected collision error")
	}
	if _, err := os.Stat(filepath.Join(outDir, "b.puml")); err == nil {
		t.Error("no file must be written when a collision is detected")
	}
	if outBuf.Len() != 0 {
		t.Errorf("expected no stdout on error, got: %s", outBuf.String())
	}
}

func TestExportDiagram_NonJSONFileOutput_NewFormats(t *testing.T) {
	modelPath := writeExportDiagramModel(t)
	for format, want := range map[string]string{
		"dot":  "architecture-context.dot",
		"d2":   "architecture-context.d2",
		"html": "context.html",
	} {
		t.Run(format, func(t *testing.T) {
			outDir := t.TempDir()
			if _, err := executeRootCmd("export-diagram", "--model", modelPath, "--view", "context",
				"--diagram-format", format, "--output", outDir); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if _, err := os.Stat(filepath.Join(outDir, want)); err != nil {
				t.Errorf("expected %s: %v", want, err)
			}
		})
	}
}

func TestExportDiagram_NonJSONAllViewsToFiles(t *testing.T) {
	modelPath := writeExportDiagramModel(t)
	for _, format := range []string{"plantuml", "html", "dot"} {
		t.Run(format, func(t *testing.T) {
			outDir := t.TempDir()
			if _, err := executeRootCmd("export-diagram", "--model", modelPath,
				"--diagram-format", format, "--output", outDir); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			entries, _ := os.ReadDir(outDir)
			if len(entries) != 2 {
				t.Errorf("expected 2 files (one per view), got %d", len(entries))
			}
		})
	}
}

func TestExportDiagram_Structurizr_ViewRejectedAndWriteFailure(t *testing.T) {
	modelPath := writeExportDiagramModel(t)
	if _, err := executeRootCmd("export-diagram", "--model", modelPath, "--diagram-format", "structurizr", "--view", "context"); err == nil {
		t.Error("expected error for --view with structurizr")
	}
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"--diagram-format", "structurizr", "--format", "json"},
		{"--diagram-format", "structurizr"},
		{"--diagram-format", "plantuml", "--format", "json"},
		{"--diagram-format", "plantuml"},
		{"--diagram-format", "dot", "--format", "json"},
		{"--diagram-format", "dot"},
	} {
		full := append([]string{"export-diagram", "--model", modelPath, "--output", filepath.Join(blocker, "sub")}, args...)
		if _, err := executeRootCmd(full...); err == nil {
			t.Errorf("expected write failure for %v", args)
		}
	}
}

const c4KindTestModel = `{
  "specification": {
    "elements": {
      "person": {"notation": "Person"},
      "system": {"notation": "Software System"},
      "widget": {"notation": "Widget"}
    }
  },
  "model": {
    "customer": {"kind": "person", "title": "Customer"},
    "shop":     {"kind": "system", "title": "Shop"},
    "gizmo":    {"kind": "widget", "title": "Gizmo"}
  },
  "relationships": [{"from": "customer", "to": "shop", "label": "places order"}],
  "views": {"context": {"title": "Context", "include": ["*"]}}
}`

// TestExportDiagram_C4KindMapping (#633): person renders as Person(), and an
// unrecognised kind warns once on stderr instead of silently becoming System().
func TestExportDiagram_C4KindMapping(t *testing.T) {
	p := filepath.Join(t.TempDir(), "architecture.jsonc")
	if err := os.WriteFile(p, []byte(c4KindTestModel), 0600); err != nil {
		t.Fatal(err)
	}
	var outBuf, errBuf bytes.Buffer
	root := NewRootCmd()
	root.SetOut(&outBuf)
	root.SetErr(&errBuf)
	root.SetArgs([]string{"export-diagram", "--model", p, "--view", "context"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(outBuf.String(), "Person(customer,") {
		t.Errorf("expected Person(customer, ...), got:\n%s", outBuf.String())
	}
	if got := strings.Count(errBuf.String(), `element kind "widget"`); got != 1 {
		t.Errorf("expected exactly one widget warning, got %d:\n%s", got, errBuf.String())
	}
	if strings.Contains(errBuf.String(), `element kind "person"`) {
		t.Errorf("person must not be warned about:\n%s", errBuf.String())
	}
}
