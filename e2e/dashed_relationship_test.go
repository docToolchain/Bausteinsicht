package e2e

// TestDashedRelationship (#518) verifies that a relationship kind's
// specification.relationships.<kind>.dashed flag is actually applied when
// rendering, across every renderer that draws relationships as arrows:
//
//  1. sync → draw.io connector style contains "dashed=1" for the dashed
//     kind's connector, and does not for the non-dashed kind's connector.
//  2. export-diagram --diagram-format plantuml → dashed relationship uses a
//     raw ".." arrow (C4-PlantUML's own per-relationship line-style
//     override errors on the tested PlantUML versions), non-dashed uses
//     the normal Rel(...) macro.
//  3. export-diagram --diagram-format dot → dashed relationship's edge has
//     style="dashed".
//  4. export-diagram --diagram-format d2 → dashed relationship's edge has
//     a style.stroke-dash block.
//
// Mermaid is deliberately not covered here: Mermaid's own C4 diagram docs
// mark the dashed-line style override as "not yet implemented" upstream,
// so there is nothing for this fix to wire up yet (see diagram.go comment
// next to writeMermaid).

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const dashedRelModelJSON = `{
  "specification": {
    "elements": {
      "container": {
        "notation": "Container",
        "description": "A deployable unit"
      }
    },
    "relationships": {
      "sync": {
        "notation": "uses"
      },
      "async": {
        "notation": "publishes/subscribes",
        "dashed": true
      }
    }
  },
  "model": {
    "api": {
      "kind": "container",
      "title": "API",
      "description": "Main backend service"
    },
    "worker": {
      "kind": "container",
      "title": "Worker",
      "description": "Background job processor"
    },
    "queue": {
      "kind": "container",
      "title": "Queue",
      "description": "Message queue"
    }
  },
  "relationships": [
    { "from": "api", "to": "worker", "kind": "sync", "label": "calls" },
    { "from": "api", "to": "queue", "kind": "async", "label": "publishes to" }
  ],
  "views": {
    "context": {
      "title": "System Context",
      "include": ["api", "worker", "queue"]
    }
  }
}`

func TestDashedRelationship(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	modelPath := filepath.Join(dir, "architecture.jsonc")
	writeFile(t, modelPath, dashedRelModelJSON)

	// ── Step 1: sync — draw.io connector style ──────────────────────────────
	runCLI(t, bin, dir, "sync", "--model", "architecture.jsonc")

	drawioBytes, err := os.ReadFile(filepath.Join(dir, "architecture.drawio"))
	if err != nil {
		t.Fatalf("read draw.io after sync: %v", err)
	}
	drawioContent := string(drawioBytes)

	syncConn := extractConnectorStyle(t, drawioContent, "api", "worker")
	asyncConn := extractConnectorStyle(t, drawioContent, "api", "queue")

	if strings.Contains(syncConn, "dashed=1") {
		t.Errorf("sync-kind connector (api->worker) unexpectedly dashed:\n%s", syncConn)
	}
	if !strings.Contains(asyncConn, "dashed=1") {
		t.Errorf("async-kind connector (api->queue) missing dashed=1 in style:\n%s", asyncConn)
	}

	// ── Step 2: export-diagram --diagram-format plantuml ────────────────────
	plantuml := runCLI(t, bin, dir, "export-diagram", "--model", "architecture.jsonc",
		"--diagram-format", "plantuml", "--view", "context")
	if !strings.Contains(plantuml, "Rel(api, worker") {
		t.Errorf("plantuml: expected solid Rel(api, worker, ...) for sync relationship, got:\n%s", plantuml)
	}
	if !strings.Contains(plantuml, "api ..> queue") {
		t.Errorf("plantuml: expected dashed 'api ..> queue' for async relationship, got:\n%s", plantuml)
	}
	if strings.Contains(plantuml, "Rel(api, queue") {
		t.Errorf("plantuml: async relationship should not use the solid Rel() macro, got:\n%s", plantuml)
	}

	// ── Step 3: export-diagram --diagram-format dot ──────────────────────────
	dot := runCLI(t, bin, dir, "export-diagram", "--model", "architecture.jsonc",
		"--diagram-format", "dot", "--view", "context")
	if !strings.Contains(dot, `api -> queue [label="publishes to" style="dashed"]`) {
		t.Errorf("dot: expected dashed style on api->queue edge, got:\n%s", dot)
	}
	if !strings.Contains(dot, `api -> worker [label="calls"]`) {
		t.Errorf("dot: expected plain (non-dashed) edge for sync relationship, got:\n%s", dot)
	}
	if strings.Contains(dot, `style="dashed"`) && strings.Count(dot, `style="dashed"`) != 1 {
		t.Errorf("dot: expected exactly one dashed edge, got:\n%s", dot)
	}

	// ── Step 4: export-diagram --diagram-format d2 ───────────────────────────
	d2 := runCLI(t, bin, dir, "export-diagram", "--model", "architecture.jsonc",
		"--diagram-format", "d2", "--view", "context")
	if !strings.Contains(d2, "style.stroke-dash") {
		t.Errorf("d2: expected a style.stroke-dash block for the dashed relationship, got:\n%s", d2)
	}
	if !strings.Contains(d2, `api -> worker: "calls"`) {
		t.Errorf("d2: expected plain (non-block) edge for sync relationship, got:\n%s", d2)
	}
}

// extractConnectorStyle finds the mxCell edge whose bausteinsicht_from and
// bausteinsicht_to attributes match from/to, and returns its style
// attribute value. Fails the test if no matching connector is found.
func extractConnectorStyle(t *testing.T, drawioContent, from, to string) string {
	t.Helper()
	// Cell refs are scoped per view ("<viewID>--<elementID>", see
	// scopedCellID in internal/sync/forward.go) — the "context" view here
	// makes them "context--api" etc.
	marker := `source="context--` + from + `" target="context--` + to + `"`
	idx := strings.Index(drawioContent, marker)
	if idx < 0 {
		t.Fatalf("no connector found for %s->%s in draw.io output:\n%s", from, to, drawioContent)
	}
	// The mxCell element containing this marker starts before it; search
	// backward for the enclosing "<mxCell" and forward for the style attribute.
	cellStart := strings.LastIndex(drawioContent[:idx], "<mxCell")
	cellEnd := strings.Index(drawioContent[idx:], ">")
	if cellStart < 0 || cellEnd < 0 {
		t.Fatalf("could not locate mxCell bounds for %s->%s connector", from, to)
	}
	cell := drawioContent[cellStart : idx+cellEnd]
	styleIdx := strings.Index(cell, `style="`)
	if styleIdx < 0 {
		t.Fatalf("connector %s->%s has no style attribute:\n%s", from, to, cell)
	}
	styleStart := styleIdx + len(`style="`)
	styleEnd := strings.Index(cell[styleStart:], `"`)
	if styleEnd < 0 {
		t.Fatalf("unterminated style attribute for %s->%s connector:\n%s", from, to, cell)
	}
	return cell[styleStart : styleStart+styleEnd]
}
