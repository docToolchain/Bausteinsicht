package e2e

// TestLintHexagonalPortConstraints verifies the `no-outgoing-dependency`
// constraint rule end-to-end: a hexagonal port is a pure interface, so it must
// never be the source of a relationship. The test lints a clean model (pass),
// then wires an outgoing dependency out of an outbound port and lints again
// (fail, naming the constraint and the offending port). It also confirms the
// `allowed-relationship` infrastructure guard stays green on the clean model.
//
// Covers the new HEX-* constraints shipped with examples/hexagonal-ddd.

import (
	"path/filepath"
	"strings"
	"testing"
)

const hexModelClean = `{
  "specification": {
    "elements": {
      "drivingAdapter":     { "notation": "Driving Adapter" },
      "inboundPort":        { "notation": "Inbound Port" },
      "applicationService": { "notation": "Application Service" },
      "outboundPort":       { "notation": "Outbound Port" },
      "drivenAdapter":      { "notation": "Driven Adapter" },
      "datastore":          { "notation": "Data Store" }
    },
    "relationships": {
      "uses":       { "notation": "uses" },
      "implements": { "notation": "implements" }
    }
  },
  "model": {
    "rest":       { "kind": "drivingAdapter",     "title": "REST Controller" },
    "placeOrder": { "kind": "inboundPort",        "title": "PlaceOrder Port" },
    "app":        { "kind": "applicationService", "title": "Order App Service" },
    "repo":       { "kind": "outboundPort",       "title": "Repository Port" },
    "pg":         { "kind": "drivenAdapter",      "title": "Postgres Adapter" },
    "db":         { "kind": "datastore",          "title": "Orders DB" }
  },
  "relationships": [
    { "from": "rest", "to": "placeOrder", "kind": "uses" },
    { "from": "app",  "to": "placeOrder", "kind": "implements" },
    { "from": "app",  "to": "repo",       "kind": "uses" },
    { "from": "pg",   "to": "repo",       "kind": "implements" },
    { "from": "pg",   "to": "db",         "kind": "uses" }
  ],
  "constraints": [
    { "id": "HEX-002", "rule": "no-outgoing-dependency", "from-kind": "outboundPort",
      "description": "Outbound ports are interfaces and must not depend on anything" },
    { "id": "HEX-003", "rule": "allowed-relationship", "to-kind": "datastore",
      "from-kinds": ["drivenAdapter"],
      "description": "Only driven adapters may access a data store" }
  ]
}`

// hexModelPortLeaks adds a dependency OUT of the outbound port (repo -> pg),
// which breaks the "ports are interfaces" rule.
const hexModelPortLeaks = `{
  "specification": {
    "elements": {
      "drivingAdapter":     { "notation": "Driving Adapter" },
      "inboundPort":        { "notation": "Inbound Port" },
      "applicationService": { "notation": "Application Service" },
      "outboundPort":       { "notation": "Outbound Port" },
      "drivenAdapter":      { "notation": "Driven Adapter" },
      "datastore":          { "notation": "Data Store" }
    },
    "relationships": {
      "uses":       { "notation": "uses" },
      "implements": { "notation": "implements" }
    }
  },
  "model": {
    "rest":       { "kind": "drivingAdapter",     "title": "REST Controller" },
    "placeOrder": { "kind": "inboundPort",        "title": "PlaceOrder Port" },
    "app":        { "kind": "applicationService", "title": "Order App Service" },
    "repo":       { "kind": "outboundPort",       "title": "Repository Port" },
    "pg":         { "kind": "drivenAdapter",      "title": "Postgres Adapter" },
    "db":         { "kind": "datastore",          "title": "Orders DB" }
  },
  "relationships": [
    { "from": "rest", "to": "placeOrder", "kind": "uses" },
    { "from": "app",  "to": "placeOrder", "kind": "implements" },
    { "from": "app",  "to": "repo",       "kind": "uses" },
    { "from": "pg",   "to": "repo",       "kind": "implements" },
    { "from": "pg",   "to": "db",         "kind": "uses" },
    { "from": "repo", "to": "pg",         "kind": "uses" }
  ],
  "constraints": [
    { "id": "HEX-002", "rule": "no-outgoing-dependency", "from-kind": "outboundPort",
      "description": "Outbound ports are interfaces and must not depend on anything" },
    { "id": "HEX-003", "rule": "allowed-relationship", "to-kind": "datastore",
      "from-kinds": ["drivenAdapter"],
      "description": "Only driven adapters may access a data store" }
  ]
}`

func TestLintHexagonalPortConstraints(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "architecture.jsonc")

	// ── Clean model: ports have no outgoing edges → lint passes ────────────────
	writeFile(t, modelPath, hexModelClean)
	out, code := runCLIAllowFail(t, bin, dir, "lint")
	if code != 0 {
		t.Fatalf("clean model should lint clean, got exit %d:\n%s", code, out)
	}
	if !strings.Contains(out, "passed") {
		t.Errorf("clean lint output does not indicate passing: %q", out)
	}

	// ── Port leaks a dependency (repo -> pg) → HEX-002 violation ───────────────
	writeFile(t, modelPath, hexModelPortLeaks)
	badOut, badCode := runCLIAllowFail(t, bin, dir, "lint")
	if badCode == 0 {
		t.Fatalf("lint should fail when an outbound port has an outgoing dependency:\n%s", badOut)
	}
	if !strings.Contains(badOut, "HEX-002") {
		t.Errorf("violation output should name constraint HEX-002: %q", badOut)
	}
	if !strings.Contains(badOut, "repo") {
		t.Errorf("violation output should name the offending port 'repo': %q", badOut)
	}

	// ── JSON output surfaces the same failure ──────────────────────────────────
	jsonOut, _ := runCLIAllowFail(t, bin, dir, "lint", "--format", "json")
	if !strings.Contains(jsonOut, `"passed": false`) {
		t.Errorf("lint --format json should report passed:false: %s", jsonOut)
	}
}
