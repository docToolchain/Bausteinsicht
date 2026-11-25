package e2e

// TestExportSequenceShapesAndBlocks covers #637 through the real CLI: actors
// as actors, autonumber instead of numbers baked into the labels, double quotes
// kept, and alt/else/end blocks, for PlantUML and Mermaid. The model is the
// reproduction from the issue plus a block.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sequenceShapesModel = `{
  "specification": {
    "elements": { "actor": { "notation": "Actor" }, "system": { "notation": "Software System" } }
  },
  "model": {
    "customer": { "kind": "actor",  "title": "Customer" },
    "shop":     { "kind": "system", "title": "Shop" }
  },
  "relationships": [],
  "views": { "context": { "title": "Context", "include": ["customer", "shop"] } },
  "dynamicViews": [
    { "key": "checkout", "title": "Checkout", "steps": [
      { "index": 1, "from": "customer", "to": "shop", "label": "submit \"order\"" },
      { "index": 2, "type": "alt", "label": "card accepted" },
      { "index": 3, "from": "shop", "to": "customer", "label": "confirmation", "type": "return" },
      { "index": 4, "type": "else", "label": "card declined" },
      { "index": 5, "from": "shop", "to": "customer", "label": "error", "type": "return" },
      { "index": 6, "type": "end" }
    ] }
  ]
}
`

func TestExportSequenceShapesAndBlocks(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "architecture.jsonc"), []byte(sequenceShapesModel), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, code := runCLIAllowFail(t, bin, dir, "validate"); code != 0 {
		t.Fatalf("the model must be valid (exit %d):\n%s", code, out)
	}

	t.Run("plantuml", func(t *testing.T) {
		out := runCLI(t, bin, dir, "export-sequence", "--view", "checkout")
		for _, want := range []string{
			"autonumber\n",
			`actor "Customer" as customer`,
			`participant "Shop" as shop`,
			`customer -> shop : submit "order"`,
			"alt card accepted\n",
			"  shop --> customer : confirmation\n",
			"else card declined\n",
			"\nend\n",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("missing %q in:\n%s", want, out)
			}
		}
		if strings.Contains(out, "1. submit") {
			t.Errorf("step number baked into the label:\n%s", out)
		}
	})

	t.Run("mermaid", func(t *testing.T) {
		out := runCLI(t, bin, dir, "export-sequence", "--view", "checkout", "--diagram-format", "mermaid")
		for _, want := range []string{
			"autonumber\n",
			"actor customer as Customer",
			`customer->>shop: submit "order"`,
			"alt card accepted\n",
			"else card declined\n",
			"    end\n",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("missing %q in:\n%s", want, out)
			}
		}
	})

	t.Run("an unclosed block fails validation and the export", func(t *testing.T) {
		broken := strings.Replace(sequenceShapesModel, `,
      { "index": 6, "type": "end" }`, "", 1)
		bdir := t.TempDir()
		if err := os.WriteFile(filepath.Join(bdir, "architecture.jsonc"), []byte(broken), 0o644); err != nil {
			t.Fatal(err)
		}
		out, code := runCLIAllowFail(t, bin, bdir, "export-sequence")
		if code == 0 || !strings.Contains(out, "never closed") {
			t.Errorf("expected exit != 0 and a 'never closed' error, got exit %d:\n%s", code, out)
		}
	})
}
