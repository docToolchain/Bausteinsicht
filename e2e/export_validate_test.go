package e2e

// TestExportsRejectInvalidModel covers #636: the export commands used to run on
// models that `validate` rejects (here: a dynamic view step pointing at an
// element that does not exist), so a typo ended up in generated documentation
// and the command exited 0. They must fail with the validation error and a
// non-zero exit code, unless --no-validate is given.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const invalidModelForExport = `{
  "specification": {
    "elements": { "system": { "notation": "Software System" } }
  },
  "model": {
    "shop":    { "kind": "system", "title": "Shop" },
    "payment": { "kind": "system", "title": "Payment" }
  },
  "relationships": [],
  "views": { "context": { "title": "Context", "include": ["shop", "payment"] } },
  "dynamicViews": [
    { "key": "checkout", "title": "Checkout", "steps": [
      { "index": 1, "from": "shop", "to": "billing", "label": "charge" }
    ] }
  ]
}
`

func TestExportsRejectInvalidModel(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "architecture.jsonc"), []byte(invalidModelForExport), 0o644); err != nil {
		t.Fatal(err)
	}

	// Sanity check: validate itself rejects the model.
	if _, code := runCLIAllowFail(t, bin, dir, "validate"); code == 0 {
		t.Fatal("validate accepted the invalid model; the test model is wrong")
	}

	commands := map[string][]string{
		"export-sequence": {"export-sequence"},
		"export-diagram":  {"export-diagram", "--view", "context"},
		"export-table":    {"export-table", "--view", "context"},
		"export":          {"export", "--output", "out"},
	}

	for name, args := range commands {
		t.Run(name+" fails on an invalid model", func(t *testing.T) {
			stdout, stderr, code := runCLISplit(t, bin, dir, args...)
			if code == 0 {
				t.Fatalf("exit 0, want non-zero\nstdout: %s\nstderr: %s", stdout, stderr)
			}
			if !strings.Contains(stdout+stderr, `"billing" does not resolve`) {
				t.Errorf("output does not contain the validation error\nstdout: %s\nstderr: %s", stdout, stderr)
			}
			if !strings.Contains(stdout+stderr, "--no-validate") {
				t.Errorf("output does not mention --no-validate\nstdout: %s\nstderr: %s", stdout, stderr)
			}
		})
	}

	// The escape hatch keeps the old behaviour for the text exports.
	for _, name := range []string{"export-sequence", "export-diagram", "export-table"} {
		args := append(append([]string{}, commands[name]...), "--no-validate")
		t.Run(name+" --no-validate exports anyway", func(t *testing.T) {
			stdout, stderr, code := runCLISplit(t, bin, dir, args...)
			if code != 0 {
				t.Fatalf("exit %d, want 0\nstdout: %s\nstderr: %s", code, stdout, stderr)
			}
		})
	}
}
