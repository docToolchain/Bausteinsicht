package e2e

// TestRelationshipTechnologyAndC4Header covers #642 through the real CLI:
// `add relationship --technology`, the technology as fourth argument of Rel(),
// and the title and legend in the C4 PlantUML header.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const technologyModel = `{
  "specification": {
    "elements": { "system": { "notation": "Software System" } },
    "relationships": { "uses": { "notation": "uses" } }
  },
  "model": {
    "shop":    { "kind": "system", "title": "Shop" },
    "payment": { "kind": "system", "title": "Payment" }
  },
  "relationships": [],
  "views": {
    "context": { "title": "System Context", "include": ["*"] },
    "quiet":   { "title": "Quiet", "include": ["*"], "legend": false }
  }
}
`

func TestRelationshipTechnologyAndC4Header(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "architecture.jsonc"), []byte(technologyModel), 0o644); err != nil {
		t.Fatal(err)
	}

	runCLI(t, bin, dir, "add", "relationship", "--from", "shop", "--to", "payment",
		"--label", "charges", "--technology", "HTTPS/JSON", "--kind", "uses")

	data, err := os.ReadFile(filepath.Join(dir, "architecture.jsonc"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"technology": "HTTPS/JSON"`) {
		t.Errorf("technology not written to the model:\n%s", data)
	}
	if out, code := runCLIAllowFail(t, bin, dir, "validate"); code != 0 {
		t.Fatalf("model with a relationship technology must validate (exit %d):\n%s", code, out)
	}

	out := runCLI(t, bin, dir, "export-diagram", "--view", "context")
	for _, want := range []string{
		"!include <C4/C4_Context>\ntitle System Context\nLAYOUT_WITH_LEGEND()\n",
		`Rel(shop, payment, "charges", "HTTPS/JSON")`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}

	quiet := runCLI(t, bin, dir, "export-diagram", "--view", "quiet")
	if strings.Contains(quiet, "LAYOUT_WITH_LEGEND") || !strings.Contains(quiet, "title Quiet\n") {
		t.Errorf("legend: false must drop only the legend:\n%s", quiet)
	}

	dsl := runCLI(t, bin, dir, "export-diagram", "--diagram-format", "structurizr")
	if !strings.Contains(dsl, `shop -> payment "charges" "HTTPS/JSON"`) {
		t.Errorf("Structurizr DSL lacks the technology:\n%s", dsl)
	}
}
