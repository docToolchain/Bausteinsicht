package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sequenceTestModel = `{
  "specification": {
    "elements": {
      "service": { "notation": "Service" }
    }
  },
  "model": {
    "frontend": { "kind": "service", "title": "Frontend" },
    "backend":  { "kind": "service", "title": "Backend" },
    "database": { "kind": "service", "title": "Database" }
  },
  "views": {},
  "dynamicViews": [
    {
      "key": "login-flow",
      "title": "Login Flow",
      "steps": [
        { "index": 1, "from": "frontend", "to": "backend",  "label": "POST /login",  "type": "sync" },
        { "index": 2, "from": "backend",  "to": "database", "label": "findUser()",   "type": "sync" },
        { "index": 3, "from": "database", "to": "backend",  "label": "user record",  "type": "return" },
        { "index": 4, "from": "backend",  "to": "frontend", "label": "200 OK + JWT", "type": "return" }
      ]
    }
  ]
}`

func writeSequenceModel(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "architecture.jsonc")
	if err := os.WriteFile(path, []byte(sequenceTestModel), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestExportSequenceCmd_PlantUMLStdout(t *testing.T) {
	modelPath := writeSequenceModel(t)
	var buf bytes.Buffer
	root := NewRootCmd()
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"export-sequence", "--model", modelPath, "--diagram-format", "plantuml"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "@startuml") {
		t.Errorf("expected PlantUML output:\n%s", out)
	}
	if !strings.Contains(out, "Login Flow") {
		t.Errorf("expected view title in output:\n%s", out)
	}
}

func TestExportSequenceCmd_MermaidStdout(t *testing.T) {
	modelPath := writeSequenceModel(t)
	var buf bytes.Buffer
	root := NewRootCmd()
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"export-sequence", "--model", modelPath, "--diagram-format", "mermaid"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "sequenceDiagram") {
		t.Errorf("expected Mermaid output:\n%s", out)
	}
}

func TestExportSequenceCmd_JSONOutput(t *testing.T) {
	modelPath := writeSequenceModel(t)
	var buf bytes.Buffer
	root := NewRootCmd()
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"export-sequence", "--model", modelPath, "--format", "json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var entries []map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entries); err != nil {
		t.Fatalf("invalid JSON: %v\noutput:\n%s", err, buf.String())
	}
	if len(entries) == 0 {
		t.Error("expected at least one entry in JSON output")
	}
	if entries[0]["view"] != "login-flow" {
		t.Errorf("expected view=login-flow, got %v", entries[0]["view"])
	}
}

func TestExportSequenceCmd_ViewFilter(t *testing.T) {
	modelPath := writeSequenceModel(t)
	var buf bytes.Buffer
	root := NewRootCmd()
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"export-sequence", "--model", modelPath, "--view", "login-flow"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "Login Flow") {
		t.Errorf("expected login-flow in output:\n%s", buf.String())
	}
}

func TestExportSequenceCmd_ViewNotFound(t *testing.T) {
	modelPath := writeSequenceModel(t)
	root := NewRootCmd()
	root.SetArgs([]string{"export-sequence", "--model", modelPath, "--view", "nonexistent"})
	if err := root.Execute(); err == nil {
		t.Error("expected error for unknown view key")
	}
}

func TestExportSequenceCmd_InvalidFormat(t *testing.T) {
	modelPath := writeSequenceModel(t)
	root := NewRootCmd()
	root.SetArgs([]string{"export-sequence", "--model", modelPath, "--diagram-format", "invalid"})
	if err := root.Execute(); err == nil {
		t.Error("expected error for invalid diagram format")
	}
}

func TestExportSequenceCmd_FileOutput(t *testing.T) {
	modelPath := writeSequenceModel(t)
	outDir := t.TempDir()
	var errBuf bytes.Buffer
	root := NewRootCmd()
	root.SetErr(&errBuf)
	root.SetArgs([]string{"export-sequence", "--model", modelPath, "--output", outDir})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	outFile := filepath.Join(outDir, "sequence-login-flow.puml")
	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("expected output file %s: %v", outFile, err)
	}
	if !strings.Contains(string(content), "@startuml") {
		t.Errorf("expected PlantUML content in file:\n%s", string(content))
	}
}

func TestExportSequenceCmd_NoDynamicViews(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "architecture.jsonc")
	emptyModel := `{
  "specification": { "elements": { "service": { "notation": "Service" } } },
  "model": { "a": { "kind": "service", "title": "A" } },
  "views": {}
}`
	if err := os.WriteFile(modelPath, []byte(emptyModel), 0600); err != nil {
		t.Fatal(err)
	}
	var errBuf bytes.Buffer
	root := NewRootCmd()
	root.SetErr(&errBuf)
	root.SetArgs([]string{"export-sequence", "--model", modelPath})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(errBuf.String(), "No dynamic views") {
		t.Errorf("expected 'No dynamic views' message:\n%s", errBuf.String())
	}
}
