package e2e

import (
	"fmt"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/drawio"
)

// liftedFixture returns a model whose "detail" view (scope shop) shows
// shop.web and shop.lib but not shop.lib.core, so web→lib.core is lifted to
// web→lib. extraRels is spliced in before that relationship.
func liftedFixture(extraRels string) string {
	return fmt.Sprintf(`{
  "specification": {
    "elements": {
      "system": { "notation": "System", "container": true },
      "container": { "notation": "Container", "container": true },
      "component": { "notation": "Component" }
    }
  },
  "model": {
    "shop": { "kind": "system", "title": "Shop", "children": {
      "web": { "kind": "container", "title": "Web" },
      "lib": { "kind": "container", "title": "Lib", "children": {
        "core": { "kind": "component", "title": "Core" }
      }}
    }}
  },
  "relationships": [
    %s
    { "from": "shop.web", "to": "shop.lib.core", "label": "calls core" }
  ],
  "views": {
    "detail": { "title": "Detail", "scope": "shop", "include": ["shop.web", "shop.lib"] }
  }
}
`, extraRels)
}

// TestSyncLiftedConnectorIndexShift: inserting relationships before a
// lifted one shifted its index on every re-sync and left one stale copy of
// the lifted connector per shift (self-hosted arc42 model showed sync→drawio
// three times). After the fix the page always carries exactly one.
func TestSyncLiftedConnectorIndexShift(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	path := dir + "/architecture.jsonc"

	writeFile(t, path, liftedFixture(""))
	runCLI(t, bin, dir, "sync")
	writeFile(t, path, liftedFixture(`{ "from": "shop.lib", "to": "shop.web", "label": "notifies" },`))
	runCLI(t, bin, dir, "sync")
	writeFile(t, path, liftedFixture(`{ "from": "shop.lib", "to": "shop.web", "label": "notifies" },
    { "from": "shop.lib", "to": "shop.web", "label": "pushes" },`))
	runCLI(t, bin, dir, "sync")

	doc, err := drawio.LoadDocument(dir + "/architecture.drawio")
	if err != nil {
		t.Fatalf("LoadDocument: %v", err)
	}
	page := doc.GetPage("view-detail")
	if page == nil {
		t.Fatal("view-detail page missing")
	}
	var lifted []string
	for _, c := range page.FindAllConnectors() {
		if c.SelectAttrValue("source", "") == "detail--shop.web" && c.SelectAttrValue("target", "") == "detail--shop.lib" {
			lifted = append(lifted, c.SelectAttrValue("id", ""))
		}
	}
	if len(lifted) != 1 {
		t.Fatalf("want exactly 1 lifted connector web→lib after two index shifts, got %d: %v", len(lifted), lifted)
	}
}
