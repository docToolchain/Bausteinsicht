package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
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

// The identity of a relationship is from, to, kind and label, as in validate
// (BR-012): the same four are refused, another label is another relationship (#697).
func TestAddRelationshipDuplicateBlocked(t *testing.T) {
	dir := t.TempDir()
	modelPath := writeRelTestModel(t, dir)

	add := func(label string) error {
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"add", "relationship",
			"--model", modelPath,
			"--from", "webshop.api",
			"--to", "webshop.db",
			"--kind", "uses",
			"--label", label,
		})
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		return cmd.Execute()
	}
	if err := add("reads"); err != nil {
		t.Fatalf("first add failed: %v", err)
	}
	if err := add("reads"); err == nil {
		t.Fatal("expected an error for a relationship with the same from, to, kind and label")
	}
	if err := add("also reads"); err != nil {
		t.Fatalf("another label is another relationship, got: %v", err)
	}

	m, err := model.Load(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Relationships) != 2 {
		t.Errorf("expected 2 relationships (the duplicate blocked), got %d", len(m.Relationships))
	}
	if errs := model.Validate(m); len(errs) > 0 {
		t.Errorf("the model must stay valid, got %v", errs)
	}
}

// A relationship of an element to itself is allowed, but on an element that is
// not a container it is probably a mistake, so it is reported (#697).
func TestAddRelationshipSelfRelationshipWarning(t *testing.T) {
	for _, tc := range []struct {
		id   string
		warn bool
	}{
		{"payments", true},     // its kind is not a container kind
		{"webshop", true},      // not a container kind either
		{"webshop.api", false}, // a container kind: it may call itself
	} {
		dir := t.TempDir()
		modelPath := writeRelTestModel(t, dir)
		cmd := NewRootCmd()
		stderr := &bytes.Buffer{}
		cmd.SetErr(stderr)
		cmd.SetArgs([]string{"add", "relationship", "--model", modelPath, "--from", tc.id, "--to", tc.id})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("%s: a self-relationship is allowed, got %v", tc.id, err)
		}
		if got := strings.Contains(stderr.String(), "itself"); got != tc.warn {
			t.Errorf("%s: warning = %v, want %v (stderr %q)", tc.id, got, tc.warn, stderr.String())
		}
	}
}

func TestAddRelationshipDifferentKindAllowed(t *testing.T) {
	dir := t.TempDir()
	modelPath := writeRelTestModel(t, dir)

	// Add first relationship with kind "uses".
	cmd1 := NewRootCmd()
	cmd1.SetArgs([]string{"add", "relationship",
		"--model", modelPath,
		"--from", "webshop.api",
		"--to", "webshop.db",
		"--kind", "uses",
		"--label", "reads from",
	})
	if err := cmd1.Execute(); err != nil {
		t.Fatalf("first add failed: %v", err)
	}

	// Add second relationship with different kind "depends_on" — should succeed.
	cmd2 := NewRootCmd()
	cmd2.SetArgs([]string{"add", "relationship",
		"--model", modelPath,
		"--from", "webshop.api",
		"--to", "webshop.db",
		"--kind", "depends_on",
		"--label", "depends on",
	})
	if err := cmd2.Execute(); err != nil {
		t.Fatalf("second add with different kind should succeed, got: %v", err)
	}

	// Verify both relationships exist.
	m, err := model.Load(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Relationships) != 2 {
		t.Fatalf("expected 2 relationships, got %d", len(m.Relationships))
	}
	if m.Relationships[0].Kind != "uses" {
		t.Errorf("expected first relationship kind 'uses', got %q", m.Relationships[0].Kind)
	}
	if m.Relationships[1].Kind != "depends_on" {
		t.Errorf("expected second relationship kind 'depends_on', got %q", m.Relationships[1].Kind)
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

// --technology is stored on the relationship, written to the file and part of
// the JSON result (#642).
func TestAddRelationshipTechnology(t *testing.T) {
	dir := t.TempDir()
	modelPath := writeRelTestModel(t, dir)

	out := captureJSON(t, "add", "relationship",
		"--model", modelPath,
		"--from", "webshop.api",
		"--to", "webshop.db",
		"--label", "reads from",
		"--technology", "JDBC",
		"--format", "json",
	)
	var result struct {
		Technology string `json:"technology"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if result.Technology != "JDBC" {
		t.Errorf("JSON result technology = %q, want JDBC", result.Technology)
	}

	m, err := model.Load(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Relationships) != 1 || m.Relationships[0].Technology != "JDBC" {
		t.Errorf("technology not stored: %+v", m.Relationships)
	}
}
