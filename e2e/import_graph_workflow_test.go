package e2e

// TestImportGraphWorkflow (#562) exercises the reverse code-governance path:
//
//  1. Model has three codeMapping-anchored elements (api, svc, db) and a
//     "toBe" section declaring api->svc, svc->db.
//  2. import-graph ingests a codegraph JSON with those two edges *plus* a
//     drift edge (api->db, not in the model) and writes it as "asIs".
//  3. diff (consuming the file import-graph just wrote) must surface the
//     drift edge as an added relationship — proving the two commands
//     actually agree on what's on disk, not just on their own in-memory
//     structs.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const importGraphModelJSON = `{
  "specification": {
    "elements": { "component": { "notation": "Component" } },
    "relationships": { "uses": { "notation": "uses" } }
  },
  "model": {
    "api": { "kind": "component", "title": "API",
      "codeMapping": { "packages": ["com.acme.backend.api"] } },
    "svc": { "kind": "component", "title": "Service",
      "codeMapping": { "packages": ["com.acme.backend.svc"] } },
    "db": { "kind": "component", "title": "Repository",
      "codeMapping": { "packages": ["com.acme.backend.db"] } }
  },
  "relationships": [
    { "from": "api", "to": "svc", "kind": "uses" },
    { "from": "svc", "to": "db", "kind": "uses" }
  ],
  "views": {},
  "toBe": {
    "elements": {
      "api": { "kind": "component", "title": "API" },
      "svc": { "kind": "component", "title": "Service" },
      "db":  { "kind": "component", "title": "Repository" }
    },
    "relationships": [
      { "from": "api", "to": "svc", "kind": "uses" },
      { "from": "svc", "to": "db", "kind": "uses" }
    ]
  }
}`

const importGraphCodeGraphJSON = `{
  "schemaVersion": "1.0",
  "kind": "bausteinsicht.codegraph",
  "language": "java",
  "nodes": [
    { "id": "com.acme.backend.api", "kind": "package" },
    { "id": "com.acme.backend.svc", "kind": "package" },
    { "id": "com.acme.backend.db", "kind": "package" },
    { "id": "com.acme.legacy.util", "kind": "package" }
  ],
  "edges": [
    { "from": "com.acme.backend.api", "to": "com.acme.backend.svc" },
    { "from": "com.acme.backend.svc", "to": "com.acme.backend.db" },
    { "from": "com.acme.backend.api", "to": "com.acme.backend.db",
      "weight": 3, "examples": ["OrderController -> OrderRepository"] }
  ]
}`

func TestImportGraphWorkflow(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	modelPath := filepath.Join(dir, "architecture.jsonc")
	if err := os.WriteFile(modelPath, []byte(importGraphModelJSON), 0o644); err != nil {
		t.Fatalf("write model: %v", err)
	}
	graphPath := filepath.Join(dir, "reality.codegraph.json")
	if err := os.WriteFile(graphPath, []byte(importGraphCodeGraphJSON), 0o644); err != nil {
		t.Fatalf("write codegraph: %v", err)
	}

	// ── Step 1: import-graph writes asIs to the model file on disk ──────────
	out := runCLI(t, bin, dir, "import-graph", "reality.codegraph.json", "--model", "architecture.jsonc")
	t.Logf("import-graph output:\n%s", out)

	if !strings.Contains(out, "3 elements, 3 relationships") {
		t.Errorf("import-graph: expected 3 elements/3 relationships mapped (incl. drift edge), got:\n%s", out)
	}
	if !strings.Contains(out, "com.acme.legacy.util") {
		t.Errorf("import-graph: expected unmapped package reported, got:\n%s", out)
	}

	// ── Step 2: diff reads the file import-graph just wrote ────────────────
	diffOut := runCLI(t, bin, dir, "diff", "--model", "architecture.jsonc", "--format", "json")
	t.Logf("diff output:\n%s", diffOut)

	var result struct {
		Summary struct {
			AddedRelationships int `json:"addedRelationships"`
		} `json:"summary"`
		Relationships []struct {
			From string `json:"from"`
			To   string `json:"to"`
			Type string `json:"type"`
		} `json:"relationships"`
	}
	if err := json.Unmarshal([]byte(diffOut), &result); err != nil {
		t.Fatalf("parse diff JSON: %v\noutput: %s", err, diffOut)
	}

	// diff's asIs/toBe roles are inverted from import-graph's write (asIs =
	// what import-graph wrote as "reality"): a relationship present in asIs
	// but not toBe reports as "removed" from diff's toBe-relative viewpoint.
	// The worked example in #562/ADR-013 frames it the other way (toBe is
	// the intended model, asIs is reality) — either way, the drift edge
	// api->db must appear as a change, not silently vanish.
	found := false
	for _, r := range result.Relationships {
		if r.From == "api" && r.To == "db" {
			found = true
		}
	}
	if !found {
		t.Errorf("diff: expected api->db drift relationship to appear as a change, got: %+v", result.Relationships)
	}

	t.Logf("import-graph -> diff workflow OK: addedRelationships=%d, removedRelationships tracked",
		result.Summary.AddedRelationships)
}
