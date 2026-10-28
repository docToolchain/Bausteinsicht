package e2e

import (
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/drawio"
)

// TestSyncNoConnectorLiftedOntoScopeBoundary: on a partial container view
// (scope shop, only some containers included), a relationship whose other
// endpoint is not on the page was lifted to the scope boundary itself —
// the self-hosted arc42 model showed edges like "exports Mermaid diagrams"
// cli→bausteinsicht. The boundary encloses the whole page, so the edge is
// dropped there; the full container view still shows it.
func TestSyncNoConnectorLiftedOntoScopeBoundary(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeFile(t, dir+"/architecture.jsonc", `{
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
    { "from": "shop.web", "to": "shop.lib.core", "label": "calls core" }
  ],
  "views": {
    "all": { "title": "All", "scope": "shop", "include": ["shop.*"] },
    "part": { "title": "Part", "scope": "shop", "include": ["shop.web"] }
  }
}
`)
	runCLI(t, bin, dir, "sync")

	doc, err := drawio.LoadDocument(dir + "/architecture.drawio")
	if err != nil {
		t.Fatalf("LoadDocument: %v", err)
	}
	part := doc.GetPage("view-part")
	if part == nil {
		t.Fatal("view-part page missing")
	}
	for _, c := range part.FindAllConnectors() {
		if c.SelectAttrValue("source", "") == "part--shop" || c.SelectAttrValue("target", "") == "part--shop" {
			t.Errorf("connector lifted onto the scope boundary: %s", c.SelectAttrValue("id", ""))
		}
	}
	all := doc.GetPage("view-all")
	if all == nil || len(all.FindAllConnectors()) != 1 {
		t.Fatalf("full container view should still show web→lib")
	}
}
