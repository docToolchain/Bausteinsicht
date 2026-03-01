package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/docToolchain/Bauteinsicht/internal/model"
)

func writeRelTestModel(t *testing.T, dir string) string {
	t.Helper()
	p := filepath.Join(dir, "architecture.jsonc")
	content := `{
  "specification": {
    "elements": {
      "system": {"notation": "box"},
      "container": {"notation": "box", "container": true}
    },
    "relationships": {
      "uses": {"notation": "->"},
      "depends_on": {"notation": "-->", "dashed": true}
    }
  },
  "model": {
    "webshop": {
      "kind": "system",
      "title": "Webshop",
      "children": {
        "api": {
          "kind": "container",
          "title": "API"
        },
        "db": {
          "kind": "container",
          "title": "Database"
        }
      }
    },
    "payments": {
      "kind": "system",
      "title": "Payment Service"
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

func TestAddRelationshipValid(t *testing.T) {
	dir := t.TempDir()
	modelPath := writeRelTestModel(t, dir)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"add", "relationship",
		"--model", modelPath,
		"--from", "webshop.api",
		"--to", "webshop.db",
		"--label", "reads from",
		"--kind", "uses",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m, err := model.Load(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Relationships) != 1 {
		t.Fatalf("expected 1 relationship, got %d", len(m.Relationships))
	}
	r := m.Relationships[0]
	if r.From != "webshop.api" {
		t.Errorf("expected from 'webshop.api', got %q", r.From)
	}
	if r.To != "webshop.db" {
		t.Errorf("expected to 'webshop.db', got %q", r.To)
	}
	if r.Label != "reads from" {
		t.Errorf("expected label 'reads from', got %q", r.Label)
	}
	if r.Kind != "uses" {
		t.Errorf("expected kind 'uses', got %q", r.Kind)
	}
}

func TestAddRelationshipNonExistentFrom(t *testing.T) {
	dir := t.TempDir()
	modelPath := writeRelTestModel(t, dir)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"add", "relationship",
		"--model", modelPath,
		"--from", "nonexistent",
		"--to", "webshop.db",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-existent --from")
	}
	if e, ok := err.(*exitError); ok {
		if e.code != 1 {
			t.Errorf("expected exit code 1, got %d", e.code)
		}
	} else {
		t.Errorf("expected *exitError, got %T", err)
	}
}

func TestAddRelationshipNonExistentTo(t *testing.T) {
	dir := t.TempDir()
	modelPath := writeRelTestModel(t, dir)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"add", "relationship",
		"--model", modelPath,
		"--from", "webshop.api",
		"--to", "nonexistent",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-existent --to")
	}
	if e, ok := err.(*exitError); ok {
		if e.code != 1 {
			t.Errorf("expected exit code 1, got %d", e.code)
		}
	} else {
		t.Errorf("expected *exitError, got %T", err)
	}
}

func TestAddRelationshipInvalidKind(t *testing.T) {
	dir := t.TempDir()
	modelPath := writeRelTestModel(t, dir)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"add", "relationship",
		"--model", modelPath,
		"--from", "webshop.api",
		"--to", "webshop.db",
		"--kind", "nonexistent_kind",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid --kind")
	}
	if e, ok := err.(*exitError); ok {
		if e.code != 1 {
			t.Errorf("expected exit code 1, got %d", e.code)
		}
	} else {
		t.Errorf("expected *exitError, got %T", err)
	}
}

func TestAddRelationshipDuplicateBlocked(t *testing.T) {
	dir := t.TempDir()
	modelPath := writeRelTestModel(t, dir)

	cmd1 := NewRootCmd()
	cmd1.SetArgs([]string{"add", "relationship",
		"--model", modelPath,
		"--from", "webshop.api",
		"--to", "webshop.db",
	})
	if err := cmd1.Execute(); err != nil {
		t.Fatalf("first add failed: %v", err)
	}

	cmd2 := NewRootCmd()
	cmd2.SetArgs([]string{"add", "relationship",
		"--model", modelPath,
		"--from", "webshop.api",
		"--to", "webshop.db",
		"--label", "also reads",
	})
	cmd2.SilenceErrors = true
	cmd2.SilenceUsage = true
	err := cmd2.Execute()

	if err == nil {
		t.Fatal("expected error for duplicate relationship, got nil")
	}

	// Verify the duplicate was NOT added.
	m, err := model.Load(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Relationships) != 1 {
		t.Errorf("expected 1 relationship (duplicate blocked), got %d", len(m.Relationships))
	}
}

func TestAddRelationshipJSONOutput(t *testing.T) {
	dir := t.TempDir()
	modelPath := writeRelTestModel(t, dir)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"add", "relationship",
		"--model", modelPath,
		"--format", "json",
		"--from", "webshop.api",
		"--to", "payments",
		"--label", "calls",
		"--kind", "uses",
	})

	err := cmd.Execute()
	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	var result map[string]string
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON output: %v\noutput: %s", err, output)
	}
	if result["from"] != "webshop.api" {
		t.Errorf("expected from 'webshop.api', got %q", result["from"])
	}
	if result["to"] != "payments" {
		t.Errorf("expected to 'payments', got %q", result["to"])
	}
	if result["label"] != "calls" {
		t.Errorf("expected label 'calls', got %q", result["label"])
	}
	if result["kind"] != "uses" {
		t.Errorf("expected kind 'uses', got %q", result["kind"])
	}
}

func TestAddRelationshipWithDescription(t *testing.T) {
	dir := t.TempDir()
	modelPath := writeRelTestModel(t, dir)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"add", "relationship",
		"--model", modelPath,
		"--from", "webshop",
		"--to", "payments",
		"--label", "delegates to",
		"--description", "Payment processing delegation",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m, err := model.Load(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Relationships) != 1 {
		t.Fatalf("expected 1 relationship, got %d", len(m.Relationships))
	}
	r := m.Relationships[0]
	if r.Description != "Payment processing delegation" {
		t.Errorf("expected description 'Payment processing delegation', got %q", r.Description)
	}
}
