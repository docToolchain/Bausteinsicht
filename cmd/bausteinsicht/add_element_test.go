package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/docToolchain/Bauteinsicht/internal/model"
)

func writeSampleModel(t *testing.T, dir string) string {
	t.Helper()
	p := filepath.Join(dir, "architecture.jsonc")
	content := `{
  "specification": {
    "elements": {
      "actor": {"notation": "person"},
      "system": {"notation": "box", "container": true},
      "container": {"notation": "box", "container": true}
    }
  },
  "model": {
    "customer": {
      "kind": "actor",
      "title": "Customer"
    },
    "webshop": {
      "kind": "system",
      "title": "Webshop",
      "children": {
        "api": {
          "kind": "container",
          "title": "API"
        }
      }
    }
  },
  "relationships": [],
  "views": {}
}`
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestAddElementTopLevel(t *testing.T) {
	dir := t.TempDir()
	modelPath := writeSampleModel(t, dir)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"add", "element",
		"--model", modelPath,
		"--id", "payments",
		"--kind", "system",
		"--title", "Payment Service",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m, err := model.Load(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	elem, ok := m.Model["payments"]
	if !ok {
		t.Fatal("expected element 'payments' in model")
	}
	if elem.Kind != "system" {
		t.Errorf("expected kind 'system', got %q", elem.Kind)
	}
	if elem.Title != "Payment Service" {
		t.Errorf("expected title 'Payment Service', got %q", elem.Title)
	}
}

func TestAddElementNested(t *testing.T) {
	dir := t.TempDir()
	modelPath := writeSampleModel(t, dir)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"add", "element",
		"--model", modelPath,
		"--id", "db",
		"--kind", "container",
		"--title", "Database",
		"--parent", "webshop",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m, err := model.Load(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	ws, ok := m.Model["webshop"]
	if !ok {
		t.Fatal("expected 'webshop' in model")
	}
	child, ok := ws.Children["db"]
	if !ok {
		t.Fatal("expected child 'db' under webshop")
	}
	if child.Title != "Database" {
		t.Errorf("expected title 'Database', got %q", child.Title)
	}
}

func TestAddElementInvalidKind(t *testing.T) {
	dir := t.TempDir()
	modelPath := writeSampleModel(t, dir)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"add", "element",
		"--model", modelPath,
		"--id", "foo",
		"--kind", "nonexistent",
		"--title", "Foo",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid kind")
	}
	if e, ok := err.(*exitError); ok {
		if e.code != 1 {
			t.Errorf("expected exit code 1, got %d", e.code)
		}
	}
}

func TestAddElementDuplicateID(t *testing.T) {
	dir := t.TempDir()
	modelPath := writeSampleModel(t, dir)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"add", "element",
		"--model", modelPath,
		"--id", "webshop",
		"--kind", "system",
		"--title", "Duplicate",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for duplicate ID")
	}
	if e, ok := err.(*exitError); ok {
		if e.code != 1 {
			t.Errorf("expected exit code 1, got %d", e.code)
		}
	}
}

func TestAddElementDuplicateNestedID(t *testing.T) {
	dir := t.TempDir()
	modelPath := writeSampleModel(t, dir)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"add", "element",
		"--model", modelPath,
		"--id", "api",
		"--kind", "container",
		"--title", "Duplicate API",
		"--parent", "webshop",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for duplicate nested ID")
	}
}

func TestAddElementNonExistentParent(t *testing.T) {
	dir := t.TempDir()
	modelPath := writeSampleModel(t, dir)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"add", "element",
		"--model", modelPath,
		"--id", "foo",
		"--kind", "system",
		"--title", "Foo",
		"--parent", "nonexistent",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-existent parent")
	}
	if e, ok := err.(*exitError); ok {
		if e.code != 1 {
			t.Errorf("expected exit code 1, got %d", e.code)
		}
	}
}

func TestAddElementJSONOutput(t *testing.T) {
	dir := t.TempDir()
	modelPath := writeSampleModel(t, dir)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"add", "element",
		"--model", modelPath,
		"--format", "json",
		"--id", "payments",
		"--kind", "system",
		"--title", "Payment Service",
	})

	err := cmd.Execute()
	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	var result map[string]string
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON output: %v\noutput: %s", err, output)
	}
	if result["id"] != "payments" {
		t.Errorf("expected id 'payments', got %q", result["id"])
	}
	if result["kind"] != "system" {
		t.Errorf("expected kind 'system', got %q", result["kind"])
	}
}

func TestAddElementJSONOutputIncludesAllFields(t *testing.T) {
	dir := t.TempDir()
	modelPath := writeSampleModel(t, dir)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"add", "element",
		"--model", modelPath,
		"--format", "json",
		"--id", "payments",
		"--kind", "system",
		"--title", "Payment Service",
		"--technology", "Go",
		"--description", "Handles payments",
	})

	err := cmd.Execute()
	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	var result map[string]string
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON output: %v\noutput: %s", err, output)
	}
	if result["technology"] != "Go" {
		t.Errorf("expected technology 'Go', got %q", result["technology"])
	}
	if result["description"] != "Handles payments" {
		t.Errorf("expected description 'Handles payments', got %q", result["description"])
	}
}

func TestAddElementJSONErrorOutput(t *testing.T) {
	dir := t.TempDir()
	modelPath := writeSampleModel(t, dir)

	var errBuf bytes.Buffer
	cmd := NewRootCmd()
	cmd.SetErr(&errBuf)
	cmd.SetArgs([]string{"add", "element",
		"--model", modelPath,
		"--format", "json",
		"--id", "foo",
		"--kind", "nonexistent",
		"--title", "Foo",
	})

	err := ExecuteRoot(cmd)
	if err == nil {
		t.Fatal("expected error for invalid kind")
	}

	output := errBuf.String()

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("expected JSON error output, got: %s", output)
	}
	if _, ok := result["error"]; !ok {
		t.Error("expected 'error' key in JSON error output")
	}
}

func TestAddElementTextErrorOutput(t *testing.T) {
	dir := t.TempDir()
	modelPath := writeSampleModel(t, dir)

	var errBuf bytes.Buffer
	cmd := NewRootCmd()
	cmd.SetErr(&errBuf)
	cmd.SetArgs([]string{"add", "element",
		"--model", modelPath,
		"--id", "foo",
		"--kind", "nonexistent",
		"--title", "Foo",
	})

	err := ExecuteRoot(cmd)
	if err == nil {
		t.Fatal("expected error for invalid kind")
	}

	output := errBuf.String()
	// Should be plain text, not JSON
	var js map[string]interface{}
	if json.Unmarshal([]byte(output), &js) == nil {
		t.Error("expected plain text error, got JSON")
	}
	if !bytes.Contains(errBuf.Bytes(), []byte("nonexistent")) {
		t.Errorf("expected error message to mention 'nonexistent', got: %s", output)
	}
}

func TestAddElementNonContainerParentRejected(t *testing.T) {
	dir := t.TempDir()
	modelPath := writeSampleModel(t, dir)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"add", "element",
		"--model", modelPath,
		"--id", "subactor",
		"--kind", "container",
		"--title", "Sub Actor",
		"--parent", "customer",
	})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error adding child to non-container element, got nil")
	}
}

func TestAddElementWithOptionalFlags(t *testing.T) {
	dir := t.TempDir()
	modelPath := writeSampleModel(t, dir)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"add", "element",
		"--model", modelPath,
		"--id", "payments",
		"--kind", "system",
		"--title", "Payment Service",
		"--technology", "Go",
		"--description", "Handles payments",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m, err := model.Load(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	elem := m.Model["payments"]
	if elem.Technology != "Go" {
		t.Errorf("expected technology 'Go', got %q", elem.Technology)
	}
	if elem.Description != "Handles payments" {
		t.Errorf("expected description 'Handles payments', got %q", elem.Description)
	}
}
