package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/search"
)

const findTestModel = `{
  "specification": {
    "elements": {
      "service":  { "notation": "Service" },
      "database": { "notation": "Database" }
    }
  },
  "model": {
    "payment-service": { "kind": "service", "title": "Payment Service", "technology": "Go" },
    "order-service":   { "kind": "service", "title": "Order Service",   "technology": "Java" },
    "payment-db":      { "kind": "database", "title": "Payment Database", "technology": "PostgreSQL" }
  },
  "relationships": [
    { "from": "order-service", "to": "payment-service", "label": "charges via", "kind": "uses" }
  ],
  "views": {
    "context": { "title": "System Context", "include": ["payment-service", "order-service"] },
    "payment":  { "title": "Payment Domain", "include": ["payment-service", "payment-db"] }
  }
}`

func writeFindModel(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "architecture.jsonc")
	if err := os.WriteFile(path, []byte(findTestModel), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestFindCmd_TextOutput(t *testing.T) {
	modelPath := writeFindModel(t)
	var buf bytes.Buffer
	root := NewRootCmd()
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"find", "--model", modelPath, "payment"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "payment-service") {
		t.Errorf("expected payment-service in output, got:\n%s", out)
	}
	if !strings.Contains(out, "payment-db") {
		t.Errorf("expected payment-db in output, got:\n%s", out)
	}
}

func TestFindCmd_JSONOutput(t *testing.T) {
	modelPath := writeFindModel(t)
	var buf bytes.Buffer
	root := NewRootCmd()
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"find", "--model", modelPath, "--format", "json", "payment"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var resp search.Response
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v\noutput:\n%s", err, buf.String())
	}
	if resp.Total == 0 {
		t.Error("expected results in JSON response")
	}
	if resp.Query != "payment" {
		t.Errorf("expected query 'payment', got %q", resp.Query)
	}
}

func TestFindCmd_TypeFilter(t *testing.T) {
	modelPath := writeFindModel(t)
	var buf bytes.Buffer
	root := NewRootCmd()
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"find", "--model", modelPath, "--format", "json", "--type", "element", "payment"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var resp search.Response
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	for _, r := range resp.Results {
		if r.Type != search.ResultElement {
			t.Errorf("expected only element results, got %s", r.Type)
		}
	}
}

func TestFindCmd_NoResults(t *testing.T) {
	modelPath := writeFindModel(t)
	var buf bytes.Buffer
	root := NewRootCmd()
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"find", "--model", modelPath, "xyzzy-nonexistent"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "No results") {
		t.Errorf("expected 'No results' message, got:\n%s", buf.String())
	}
}

func TestFindCmd_InvalidType(t *testing.T) {
	modelPath := writeFindModel(t)
	root := NewRootCmd()
	root.SetArgs([]string{"find", "--model", modelPath, "--type", "invalid", "payment"})
	err := root.Execute()
	if err == nil {
		t.Error("expected error for invalid --type")
	}
}

func TestFindCmd_MultiWordQuery(t *testing.T) {
	modelPath := writeFindModel(t)
	var buf bytes.Buffer
	root := NewRootCmd()
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"find", "--model", modelPath, "--format", "json", "order", "service"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var resp search.Response
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	found := false
	for _, r := range resp.Results {
		if r.ID == "order-service" {
			found = true
		}
	}
	if !found {
		t.Error("expected order-service in multi-word query results")
	}
}
