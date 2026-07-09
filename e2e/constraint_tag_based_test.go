package e2e

// TestConstraintTagBasedTargeting (#542) verifies that `bausteinsicht lint`
// enforces constraints targeted by tag and by exact element ID, not just by
// kind — the motivating case is a layering rule like "UI must not talk to a
// datastore directly" where both sides share the same kind ("container") and
// can only be told apart by tag.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const tagConstraintModelJSON = `{
  "specification": {
    "elements": {
      "container": { "notation": "Container" }
    }
  },
  "model": {
    "adminConsole": { "kind": "container", "title": "Admin Console", "tags": ["ui"] },
    "batchJob": { "kind": "container", "title": "Batch Job", "tags": ["internal"] },
    "database": { "kind": "container", "title": "Database", "tags": ["datastore"] }
  },
  "relationships": [
    { "from": "adminConsole", "to": "database", "label": "queries" },
    { "from": "batchJob", "to": "database", "label": "writes" }
  ],
  "constraints": [
    {
      "id": "ui-not-to-datastore",
      "description": "UI must not talk to a datastore directly",
      "rule": "no-relationship",
      "from-tag": "ui",
      "to-tag": "datastore"
    }
  ]
}`

func TestConstraintTagBasedTargeting(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	modelPath := filepath.Join(dir, "architecture.jsonc")
	writeFile(t, modelPath, tagConstraintModelJSON)

	lintOut, code := runCLIAllowFail(t, bin, dir, "lint", "--model", "architecture.jsonc")
	t.Logf("lint output: %s", lintOut)

	if code == 0 {
		t.Fatal("lint should have failed: adminConsole (tagged 'ui') relates to database (tagged 'datastore')")
	}
	if !strings.Contains(lintOut, "adminConsole") {
		t.Errorf("lint output does not mention the violating element 'adminConsole': %q", lintOut)
	}
	if strings.Contains(lintOut, "batchJob") {
		t.Errorf("lint output should not flag 'batchJob' (not tagged 'ui'), got: %q", lintOut)
	}

	// Removing the violating relationship should make lint pass again —
	// confirms the tag-based rule is actually re-evaluated, not a fixed
	// false positive.
	fixedModel := strings.Replace(tagConstraintModelJSON,
		`{ "from": "adminConsole", "to": "database", "label": "queries" },`, "", 1)
	if err := os.WriteFile(modelPath, []byte(fixedModel), 0o644); err != nil {
		t.Fatalf("rewrite model: %v", err)
	}

	lintFixedOut, fixedCode := runCLIAllowFail(t, bin, dir, "lint", "--model", "architecture.jsonc")
	t.Logf("lint output after removing violation: %s", lintFixedOut)
	if fixedCode != 0 {
		t.Errorf("lint should pass once the ui->datastore relationship is removed, got exit %d: %s", fixedCode, lintFixedOut)
	}
}
