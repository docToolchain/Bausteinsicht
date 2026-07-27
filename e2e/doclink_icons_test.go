package e2e

// TestDocLinkIcons (#543) verifies the full validate → sync chain for the
// element→doc/ADR drilldown feature: an element's `link`, `decisions`
// (matching a Specification.Decisions record that has a `file`), and typed
// `docLinks` all produce clickable icon <object> cells in the synced
// draw.io XML, alongside a view-level DocLinks icon row — without disturbing
// the element's own drill-down/link attribute (ADR-009) or the existing
// back-nav button (#198).

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/beevik/etree"
	"github.com/docToolchain/Bausteinsicht/internal/drawio"
)

const docLinkModel = `{
  "specification": {
    "elements": {
      "system": { "notation": "System", "container": true },
      "container": { "notation": "Container" }
    },
    "relationships": {
      "uses": { "notation": "uses" }
    },
    "decisions": [
      { "id": "ADR-001", "title": "Use JSONC", "status": "active", "file": "ADRs/ADR-001-DSL-Format.html" }
    ]
  },
  "model": {
    "shop": {
      "kind": "system",
      "title": "Shop",
      "children": {
        "api": {
          "kind": "container",
          "title": "API",
          "link": "architecture.adoc#sec-api",
          "decisions": ["ADR-001"],
          "docLinks": [
            { "type": "arc42", "href": "docs/arc42/05-building-blocks.html#api", "title": "Baustein API" }
          ]
        }
      }
    }
  },
  "relationships": [],
  "views": {
    "overview": {
      "title": "Overview",
      "include": ["shop", "shop.api"],
      "layout": "layered",
      "docLinks": [
        { "type": "prd", "href": "docs/prd.html", "title": "Product Requirements" }
      ]
    },
    "detail": {
      "title": "Shop Detail",
      "scope": "shop",
      "include": ["shop.*"],
      "layout": "layered"
    }
  }
}`

func TestDocLinkIcons(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "architecture.jsonc")
	writeFile(t, modelPath, docLinkModel)

	// Chain validate → sync, as a real user's workflow would (#512-style
	// producer→consumer coverage): the new docLinks field must validate
	// cleanly before sync ever sees it.
	runCLI(t, bin, dir, "validate")
	runCLI(t, bin, dir, "sync")

	doc, err := drawio.LoadDocument(filepath.Join(dir, "architecture.drawio"))
	if err != nil {
		t.Fatalf("LoadDocument after sync: %v", err)
	}

	overviewPage := requirePageE2E(t, doc, "view-overview")
	root := overviewPage.Root()

	// Element's own link is untouched (drill-down owns "shop"'s shape link
	// since it has a detail view; "shop.api" has no detail view so keeps its
	// own user-defined link on the shape, per ADR-009's existing priority rule).
	apiObj := findObjectByBSID(root, "shop.api")
	if apiObj == nil {
		t.Fatal("expected an <object> for shop.api on the overview page")
	}
	if got := apiObj.SelectAttrValue("link", ""); got != "architecture.adoc#sec-api" {
		t.Errorf("shop.api shape: expected unchanged link %q, got %q", "architecture.adoc#sec-api", got)
	}

	// Link icon: independent clickable icon, same href. Element cell IDs on a
	// view page are scoped ("<viewID>--<elementID>"), so the icon ID follows
	// that same on-page cell ID, not the bare model element ID.
	const apiCellID = "overview--shop.api"
	linkIcon := findObjectByID(root, "doclink-"+apiCellID+"-link")
	if linkIcon == nil {
		t.Fatalf("expected a doclink-%s-link icon", apiCellID)
	}
	if got := linkIcon.SelectAttrValue("link", ""); got != "architecture.adoc#sec-api" {
		t.Errorf("link icon: expected %q, got %q", "architecture.adoc#sec-api", got)
	}

	// Decision badge is now clickable, resolved to the ADR's file.
	decisionIcon := findObjectByID(root, "doclink-"+apiCellID+"-decision-0")
	if decisionIcon == nil {
		t.Fatalf("expected a doclink-%s-decision-0 icon", apiCellID)
	}
	if got := decisionIcon.SelectAttrValue("link", ""); got != "ADRs/ADR-001-DSL-Format.html" {
		t.Errorf("decision icon: expected %q, got %q", "ADRs/ADR-001-DSL-Format.html", got)
	}

	// Typed docLinks icon.
	docIcon := findObjectByID(root, "doclink-"+apiCellID+"-doc-0")
	if docIcon == nil {
		t.Fatalf("expected a doclink-%s-doc-0 icon", apiCellID)
	}
	if got := docIcon.SelectAttrValue("link", ""); got != "docs/arc42/05-building-blocks.html#api" {
		t.Errorf("doc icon: expected %q, got %q", "docs/arc42/05-building-blocks.html#api", got)
	}

	// View-level DocLinks icon row, page-absolute (parent=1).
	viewIcon := findObjectByID(root, "doclink-view-overview-0")
	if viewIcon == nil {
		t.Fatal("expected a doclink-view-overview-0 icon on the overview page")
	}
	if got := viewIcon.SelectAttrValue("link", ""); got != "docs/prd.html" {
		t.Errorf("view icon: expected %q, got %q", "docs/prd.html", got)
	}

	// The "detail" view (scope=shop) still gets its back-nav button — new
	// icons must coexist with, not replace, existing navigation (#198).
	detailPage := requirePageE2E(t, doc, "view-detail")
	backNavFound := false
	for _, obj := range detailPage.Root().SelectElements("object") {
		if strings.HasPrefix(obj.SelectAttrValue("id", ""), "nav-back-") {
			backNavFound = true
			break
		}
	}
	if !backNavFound {
		t.Error("expected the detail view to still have its nav-back button")
	}

	// Re-sync must not duplicate icons (idempotency).
	runCLI(t, bin, dir, "sync")
	doc2, err := drawio.LoadDocument(filepath.Join(dir, "architecture.drawio"))
	if err != nil {
		t.Fatalf("LoadDocument after re-sync: %v", err)
	}
	count := 0
	for _, obj := range requirePageE2E(t, doc2, "view-overview").Root().SelectElements("object") {
		if obj.SelectAttrValue("id", "") == "doclink-"+apiCellID+"-link" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 doclink-%s-link icon after re-sync, got %d", apiCellID, count)
	}

	// ── SVG export + link validation (requires draw.io CLI) ────────────────────
	// The .drawio-XML assertions above prove the `link` attributes are written
	// correctly; this proves they actually survive draw.io's own SVG rendering
	// and come out as clickable <a href> elements — the claim 03_data_models.adoc
	// makes for `link` ("Use SVG export ... to preserve links") extended to the
	// new doc-link icons. Mirrors bigbank_arc42_test.go's SVGExportAndLinks step.
	drawioCmd := findDrawioCmd()
	if drawioCmd == "" {
		t.Log("draw.io CLI not found — skipping SVG export and link validation")
		t.Log("To run this part: install draw.io CLI and re-run the test")
		return
	}

	svgDir := filepath.Join(dir, "svgs")
	if err := os.MkdirAll(svgDir, 0o755); err != nil {
		t.Fatalf("mkdir svgs: %v", err)
	}

	exportCmd := exec.Command(bin,
		"export",
		"--model", modelPath,
		"--image-format", "svg",
		"--output", svgDir,
	)
	if out, err := exportCmd.CombinedOutput(); err != nil {
		t.Fatalf("bausteinsicht export: %v\n%s", err, out)
	}

	expectedHrefs := map[string]string{
		"element link":    "architecture.adoc#sec-api",
		"decision":        "ADRs/ADR-001-DSL-Format.html",
		"element docLink": "docs/arc42/05-building-blocks.html#api",
		"view docLink":    "docs/prd.html",
	}
	validateSVGLinks(t, filepath.Join(svgDir, "architecture-overview.svg"), expectedHrefs)
}

func requirePageE2E(t *testing.T, doc *drawio.Document, pageID string) *drawio.Page {
	t.Helper()
	page := doc.GetPage(pageID)
	if page == nil {
		t.Fatalf("expected page %q to exist", pageID)
	}
	return page
}

func findObjectByBSID(root *etree.Element, bsID string) *etree.Element {
	for _, obj := range root.SelectElements("object") {
		if obj.SelectAttrValue("bausteinsicht_id", "") == bsID {
			return obj
		}
	}
	return nil
}

func findObjectByID(root *etree.Element, id string) *etree.Element {
	for _, obj := range root.SelectElements("object") {
		if obj.SelectAttrValue("id", "") == id {
			return obj
		}
	}
	return nil
}
