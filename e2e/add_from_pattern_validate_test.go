package e2e

import (
	"os"
	"strings"
	"testing"
)

// TestAddFromPatternRefusesAnInvalidPatternEndToEnd (#703): a pattern that expands
// to an invalid model was saved and left the model unusable; now the command
// fails, lists the errors, and the model still validates and syncs.
func TestAddFromPatternRefusesAnInvalidPatternEndToEnd(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	model := `{
  "specification": {
    "elements": { "container": { "notation": "Container", "container": true } },
    "relationships": { "uses": { "notation": "uses" } },
    "patterns": {
      "bad": { "description": "unknown kind and a dangling relationship",
        "elements": [ { "id": "{base}", "kind": "nokind", "title": "t" } ],
        "relationships": [ { "id": "{base}-r", "from": "{base}", "to": "zzz", "kind": "uses" } ] },
      "ok": { "description": "one container", "elements": [ { "id": "{base}", "kind": "container", "title": "{Title}" } ], "relationships": [] }
    }
  },
  "model": { "existing": { "kind": "container", "title": "Existing" } },
  "relationships": [],
  "views": { "all": { "title": "All", "include": ["**"] } }
}`
	writeFile(t, dir+"/architecture.jsonc", model)

	out, code := runCLIAllowFail(t, bin, dir, "add", "add-from-pattern", "bad", "--id", "pay")
	if code != 1 || !strings.Contains(out, "unknown kind") || !strings.Contains(out, "nothing was saved") {
		t.Errorf("exit %d, want 1 with the errors and 'nothing was saved':\n%s", code, out)
	}
	if out, code := runCLIAllowFail(t, bin, dir, "add", "add-from-pattern", "ok", "--id", "bad id"); code != 1 || !strings.Contains(out, "invalid element ID") {
		t.Errorf("an invalid --id: exit %d\n%s", code, out)
	}
	if data, _ := os.ReadFile(dir + "/architecture.jsonc"); string(data) != model {
		t.Errorf("the model must not change when a pattern is refused:\n%s", data)
	}

	// A valid pattern is still applied, and the result validates and syncs (a model
	// without the invalid pattern, because validate and sync check the patterns too).
	okDir := t.TempDir()
	writeFile(t, okDir+"/architecture.jsonc", strings.Replace(model, `"bad": { "description": "unknown kind and a dangling relationship",
        "elements": [ { "id": "{base}", "kind": "nokind", "title": "t" } ],
        "relationships": [ { "id": "{base}-r", "from": "{base}", "to": "zzz", "kind": "uses" } ] },
      `, "", 1))
	runCLI(t, bin, okDir, "add", "add-from-pattern", "ok", "--id", "pay")
	runCLI(t, bin, okDir, "validate")
	runCLI(t, bin, okDir, "sync")
}
