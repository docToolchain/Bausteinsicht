package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// threeTierModelJSONC mirrors #562/ADR-013's worked example: api -> svc,
// svc -> db, each anchored to its own Java package via codeMapping.
const threeTierModelJSONC = `{
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
  "views": {}
}`

const threeTierCodeGraphJSON = `{
  "schemaVersion": "1.0",
  "kind": "bausteinsicht.codegraph",
  "language": "java",
  "nodes": [
    { "id": "com.acme.backend.api", "kind": "package" },
    { "id": "com.acme.backend.svc", "kind": "package" },
    { "id": "com.acme.backend.db", "kind": "package" }
  ],
  "edges": [
    { "from": "com.acme.backend.api", "to": "com.acme.backend.svc" },
    { "from": "com.acme.backend.svc", "to": "com.acme.backend.db" }
  ]
}`

func TestImportGraph_Success(t *testing.T) {
	modelPath := createTempFile(t, threeTierModelJSONC)
	defer removeTempFile(t, modelPath)
	graphPath := createTempFile(t, threeTierCodeGraphJSON)
	defer removeTempFile(t, graphPath)

	out, err := executeImportCmd("import-graph", graphPath, "--model", modelPath)
	if err != nil {
		t.Fatalf("import-graph failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "3 nodes, 2 edges") {
		t.Errorf("expected node/edge counts in output, got: %s", out)
	}
	if !strings.Contains(out, "3 elements, 2 relationships") {
		t.Errorf("expected mapped counts in output, got: %s", out)
	}

	m, err := model.Load(modelPath)
	if err != nil {
		t.Fatalf("reloading model: %v", err)
	}
	if m.AsIs == nil {
		t.Fatal("expected AsIs to be written")
	}
	if len(m.AsIs.Elements) != 3 || len(m.AsIs.Relationships) != 2 {
		t.Errorf("unexpected AsIs snapshot: %+v", m.AsIs)
	}
}

func TestImportGraph_DriftRelationshipFeedsIntoDiff(t *testing.T) {
	modelPath := createTempFile(t, threeTierModelJSONC)
	defer removeTempFile(t, modelPath)

	// Real code additionally has api -> db, which the model doesn't.
	graphWithDrift := `{
  "schemaVersion": "1.0",
  "kind": "bausteinsicht.codegraph",
  "language": "java",
  "nodes": [
    { "id": "com.acme.backend.api", "kind": "package" },
    { "id": "com.acme.backend.svc", "kind": "package" },
    { "id": "com.acme.backend.db", "kind": "package" }
  ],
  "edges": [
    { "from": "com.acme.backend.api", "to": "com.acme.backend.svc" },
    { "from": "com.acme.backend.svc", "to": "com.acme.backend.db" },
    { "from": "com.acme.backend.api", "to": "com.acme.backend.db" }
  ]
}`
	graphPath := createTempFile(t, graphWithDrift)
	defer removeTempFile(t, graphPath)

	if _, err := executeImportCmd("import-graph", graphPath, "--model", modelPath); err != nil {
		t.Fatalf("import-graph failed: %v", err)
	}

	// import-graph only writes asIs; toBe must be set separately (or reuse
	// the model's own relationships) for `diff` to have something to compare
	// against — set it here to the model's declared relationships, then run
	// the consuming command for real.
	m, err := model.Load(modelPath)
	if err != nil {
		t.Fatalf("reloading model: %v", err)
	}
	flat, err := model.FlattenElements(m)
	if err != nil {
		t.Fatalf("flattening: %v", err)
	}
	elements := make(map[string]model.Element, len(flat))
	for id, e := range flat {
		elements[id] = *e
	}
	m.ToBe = &model.ModelSnapshot{Elements: elements, Relationships: m.Relationships}
	if err := model.Save(modelPath, m); err != nil {
		t.Fatalf("saving toBe: %v", err)
	}

	out, err := executeImportCmd("diff", "--model", modelPath, "--format", "json")
	if err != nil {
		t.Fatalf("diff failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, `"from": "api"`) || !strings.Contains(out, `"to": "db"`) {
		t.Errorf("expected diff to surface the api -> db drift relationship, got: %s", out)
	}
}

func TestImportGraph_NoCodeMapping(t *testing.T) {
	modelPath := createTempFile(t, `{
    "specification": { "elements": { "system": { "notation": "System" } } },
    "model": { "sys": { "kind": "system", "title": "Sys" } },
    "views": {}
  }`)
	defer removeTempFile(t, modelPath)
	graphPath := createTempFile(t, threeTierCodeGraphJSON)
	defer removeTempFile(t, graphPath)

	_, err := executeImportCmd("import-graph", graphPath, "--model", modelPath)
	if err == nil {
		t.Error("expected error for model with no codeMapping")
	}
}

func TestImportGraph_MissingCodeGraphFile(t *testing.T) {
	modelPath := createTempFile(t, threeTierModelJSONC)
	defer removeTempFile(t, modelPath)

	_, err := executeImportCmd("import-graph", "does-not-exist.json", "--model", modelPath)
	if err == nil {
		t.Error("expected error for missing codegraph file")
	}
}

func TestImportGraph_InvalidFormat(t *testing.T) {
	modelPath := createTempFile(t, threeTierModelJSONC)
	defer removeTempFile(t, modelPath)
	graphPath := createTempFile(t, threeTierCodeGraphJSON)
	defer removeTempFile(t, graphPath)

	_, err := executeImportCmd("import-graph", graphPath, "--model", modelPath, "--format", "xml")
	if err == nil {
		t.Error("expected error for invalid format")
	}
}

func TestImportGraph_UnmappedPackagesReported(t *testing.T) {
	modelPath := createTempFile(t, threeTierModelJSONC)
	defer removeTempFile(t, modelPath)

	graphWithUnmapped := `{
  "schemaVersion": "1.0",
  "kind": "bausteinsicht.codegraph",
  "language": "java",
  "nodes": [
    { "id": "com.acme.backend.api", "kind": "package" },
    { "id": "com.acme.legacy.util", "kind": "package" }
  ],
  "edges": []
}`
	graphPath := createTempFile(t, graphWithUnmapped)
	defer removeTempFile(t, graphPath)

	out, err := executeImportCmd("import-graph", graphPath, "--model", modelPath)
	if err != nil {
		t.Fatalf("import-graph failed: %v", err)
	}
	if !strings.Contains(out, "com.acme.legacy.util") {
		t.Errorf("expected unmapped package in output, got: %s", out)
	}
}

func TestImportGraph_JSONFormat(t *testing.T) {
	modelPath := createTempFile(t, threeTierModelJSONC)
	defer removeTempFile(t, modelPath)
	graphPath := createTempFile(t, threeTierCodeGraphJSON)
	defer removeTempFile(t, graphPath)

	out, err := executeImportCmd("import-graph", graphPath, "--model", modelPath, "--format", "json")
	if err != nil {
		t.Fatalf("import-graph failed: %v", err)
	}

	var summary importGraphSummary
	if err := json.Unmarshal([]byte(out), &summary); err != nil {
		t.Fatalf("invalid JSON output: %v\noutput: %s", err, out)
	}
	if summary.Language != "java" || summary.ElementsMapped != 3 || summary.RelationshipsMapped != 2 {
		t.Errorf("unexpected summary: %+v", summary)
	}
}

func TestImportGraph_MalformedCodeGraph(t *testing.T) {
	modelPath := createTempFile(t, threeTierModelJSONC)
	defer removeTempFile(t, modelPath)
	graphPath := createTempFile(t, "not json")
	defer removeTempFile(t, graphPath)

	_, err := executeImportCmd("import-graph", graphPath, "--model", modelPath)
	if err == nil {
		t.Error("expected error for malformed codegraph file")
	}
}
