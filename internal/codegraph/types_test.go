package codegraph

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempCodeGraph(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "reality.codegraph.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	return path
}

func TestLoad_ValidCodeGraph(t *testing.T) {
	path := writeTempCodeGraph(t, `{
  "schemaVersion": "1.0",
  "kind": "bausteinsicht.codegraph",
  "language": "java",
  "nodes": [ { "id": "com.acme.backend.api", "kind": "package" } ],
  "edges": [ { "from": "com.acme.backend.api", "to": "com.acme.backend.svc",
               "weight": 12, "examples": ["OrderController -> OrderService"] } ]
}`)

	cg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cg.Language != "java" {
		t.Errorf("Language = %q, want %q", cg.Language, "java")
	}
	if len(cg.Nodes) != 1 || cg.Nodes[0].ID != "com.acme.backend.api" {
		t.Errorf("unexpected Nodes: %+v", cg.Nodes)
	}
	if len(cg.Edges) != 1 || cg.Edges[0].Weight != 12 {
		t.Errorf("unexpected Edges: %+v", cg.Edges)
	}
}

func TestLoad_MinimalRequiredFieldsOnly(t *testing.T) {
	// weight/examples are optional per #562's contract.
	path := writeTempCodeGraph(t, `{
  "schemaVersion": "1.0",
  "kind": "bausteinsicht.codegraph",
  "language": "go",
  "nodes": [ { "id": "pkg", "kind": "package" } ],
  "edges": [ { "from": "pkg", "to": "pkg2" } ]
}`)

	cg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cg.Edges[0].Weight != 0 || cg.Edges[0].Examples != nil {
		t.Errorf("expected zero-value optional fields, got %+v", cg.Edges[0])
	}
}

func TestLoad_WrongKind(t *testing.T) {
	path := writeTempCodeGraph(t, `{"schemaVersion":"1.0","kind":"something.else","language":"java","nodes":[],"edges":[]}`)

	if _, err := Load(path); err == nil {
		t.Error("expected error for wrong kind")
	}
}

func TestLoad_MissingSchemaVersion(t *testing.T) {
	path := writeTempCodeGraph(t, `{"kind":"bausteinsicht.codegraph","language":"java","nodes":[],"edges":[]}`)

	if _, err := Load(path); err == nil {
		t.Error("expected error for missing schemaVersion")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	path := writeTempCodeGraph(t, `not json`)

	if _, err := Load(path); err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Error("expected error for missing file")
	}
}
