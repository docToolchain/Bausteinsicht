package e2e

// TestDiffWorkflow (#489) exercises the as-is/to-be diff command:
//
//  1. Create a JSONC model with "asIs" and "toBe" sections (technology change Java→Go)
//  2. diff (text format) → changed element present in output with technology diff
//  3. diff --format json → parse JSON, verify changedElements count > 0

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// diffModel is a minimal Bausteinsicht JSONC-compatible JSON model with asIs/toBe sections.
// Written as plain JSON (no comments) so json.Unmarshal-compatible tools can also read it.
const diffModelJSON = `{
  "specification": {
    "elements": {
      "container": {
        "notation": "Container",
        "description": "A deployable unit"
      }
    },
    "relationships": {
      "uses": {
        "notation": "uses"
      }
    }
  },
  "model": {
    "api": {
      "kind": "container",
      "title": "API",
      "technology": "Go",
      "description": "Main backend service"
    }
  },
  "relationships": [],
  "views": {
    "context": {
      "title": "System Context",
      "include": ["api"]
    }
  },
  "asIs": {
    "elements": {
      "api": {
        "kind": "container",
        "title": "API",
        "technology": "Java",
        "description": "Main backend service (legacy)"
      },
      "legacy-db": {
        "kind": "container",
        "title": "Legacy DB",
        "technology": "Oracle",
        "description": "Will be migrated"
      }
    },
    "relationships": []
  },
  "toBe": {
    "elements": {
      "api": {
        "kind": "container",
        "title": "API",
        "technology": "Go",
        "description": "Main backend service (migrated)"
      }
    },
    "relationships": []
  }
}`

func TestDiffWorkflow(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	modelPath := filepath.Join(dir, "diff-model.jsonc")
	if err := os.WriteFile(modelPath, []byte(diffModelJSON), 0o644); err != nil {
		t.Fatalf("write diff model: %v", err)
	}

	// ── Step 1: diff (text format) ────────────────────────────────────────────
	textOut := runCLI(t, bin, dir, "diff", "--model", "diff-model.jsonc")
	t.Logf("diff text output:\n%s", textOut)

	// The "api" element changed technology from Java → Go.
	if !strings.Contains(textOut, "api") {
		t.Error("diff text output: expected 'api' element in changed list")
	}
	if !strings.Contains(textOut, "Java") || !strings.Contains(textOut, "Go") {
		t.Errorf("diff text output: expected technology change Java→Go, got:\n%s", textOut)
	}

	// "legacy-db" was removed in toBe.
	if !strings.Contains(textOut, "legacy-db") {
		t.Errorf("diff text output: expected 'legacy-db' in removed list, got:\n%s", textOut)
	}

	// ── Step 2: diff --format json ────────────────────────────────────────────
	jsonOut := runCLI(t, bin, dir, "diff", "--model", "diff-model.jsonc", "--format", "json")
	t.Logf("diff json output:\n%s", jsonOut)

	var result struct {
		Summary struct {
			AddedElements    int `json:"addedElements"`
			RemovedElements  int `json:"removedElements"`
			ChangedElements  int `json:"changedElements"`
		} `json:"summary"`
		Elements []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"elements"`
	}
	if err := json.Unmarshal([]byte(jsonOut), &result); err != nil {
		t.Fatalf("parse diff JSON: %v\noutput: %s", err, jsonOut)
	}

	if result.Summary.ChangedElements < 1 {
		t.Errorf("diff JSON: changedElements = %d, want ≥ 1", result.Summary.ChangedElements)
	}
	if result.Summary.RemovedElements < 1 {
		t.Errorf("diff JSON: removedElements = %d, want ≥ 1 (legacy-db removed)", result.Summary.RemovedElements)
	}

	// Verify "api" appears as a changed element.
	foundAPI := false
	for _, el := range result.Elements {
		if el.ID == "api" && el.Type == "changed" {
			foundAPI = true
		}
	}
	if !foundAPI {
		t.Errorf("diff JSON elements: 'api' with type 'changed' not found; got %+v", result.Elements)
	}

	t.Logf("diff workflow OK: changedElements=%d, removedElements=%d",
		result.Summary.ChangedElements, result.Summary.RemovedElements)
}
