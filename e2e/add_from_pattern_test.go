package e2e

import (
	"strings"
	"testing"
)

// TestAddFromPatternWithChildrenEndToEnd (#702): a pattern whose elements have
// children crashed after it had written the model. The pattern is applied, the
// model validates and syncs, and the children are drawn below their parent.
func TestAddFromPatternWithChildrenEndToEnd(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeFile(t, dir+"/architecture.jsonc", `{
  "specification": {
    "elements": { "container": { "notation": "Container", "container": true }, "datastore": { "notation": "Data Store" } },
    "relationships": { "uses": { "notation": "uses" } },
    "patterns": { "svc": { "description": "service with a database",
      "elements": [ { "id": "{base}", "kind": "container", "title": "{Title}",
        "children": [ { "id": "db", "kind": "datastore", "title": "{Title} DB" } ] } ],
      "relationships": [ { "id": "{base}-reads", "from": "{base}", "to": "{base}.db", "kind": "uses", "label": "reads" } ] } }
  },
  "model": {}, "relationships": [],
  "views": { "all": { "title": "All", "include": ["**"] } }
}`)

	out := runCLI(t, bin, dir, "add", "add-from-pattern", "svc", "--id", "pay", "--title", "payment")
	for _, want := range []string{"pay ", "pay.db"} {
		if !strings.Contains(out, want) {
			t.Errorf("the summary must list %q:\n%s", want, out)
		}
	}
	runCLI(t, bin, dir, "validate")
	runCLI(t, bin, dir, "sync")
	if shown := runCLI(t, bin, dir, "show", "pay.db"); !strings.Contains(shown, "Payment DB") {
		t.Errorf("the child must exist below its parent:\n%s", shown)
	}
	if out := runCLI(t, bin, dir, "sync"); !strings.Contains(out, "Already in sync") {
		t.Errorf("sync after the pattern is not a no-op:\n%s", out)
	}
}
