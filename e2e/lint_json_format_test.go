package e2e

// TestLintJSONFormatNoConstraints (#632): `lint --format json` on a model
// without a constraints section must emit valid JSON with passed/total/violations
// fields (same schema as the normal path), not plain text.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const lintNoConstraintsModel = `{
  "specification": {
    "elements": { "system": { "notation": "System" } }
  },
  "model": {
    "shop": { "kind": "system", "title": "Shop" }
  },
  "views": { "main": { "title": "Main" } }
}`

func TestLintJSONFormat_NoConstraints(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	modelPath := filepath.Join(dir, "architecture.jsonc")
	if err := os.WriteFile(modelPath, []byte(lintNoConstraintsModel), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, _, code := runCLISplit(t, bin, dir,
		"lint", "--model", modelPath, "--format", "json",
	)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d\noutput: %s", code, stdout)
	}

	var result struct {
		Passed     bool              `json:"passed"`
		Total      int               `json:"total"`
		Violations []json.RawMessage `json:"violations"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %v\noutput: %s", err, stdout)
	}
	if !result.Passed {
		t.Errorf("expected passed=true, got false; output: %s", stdout)
	}
	if result.Total != 0 {
		t.Errorf("expected total=0, got %d", result.Total)
	}
	if result.Violations == nil || len(result.Violations) != 0 {
		t.Errorf("expected violations to be an empty array (not null), got: %s", stdout)
	}
}
