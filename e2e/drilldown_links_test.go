package e2e

// TestDrillDownLinks (#503) verifies that elements with a matching scope-view
// receive a "data:page/id,view-..." drill-down link in draw.io, and that
// elements without a deeper view do NOT receive such a link.
// Also verifies that detail views (views with a scope) get a back-nav button.

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/drawio"
)

func TestDrillDownLinks(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	dslSrc := filepath.Join(findModuleRoot(t), "e2e/testdata/bigbank/workspace.dsl")
	dslDst := filepath.Join(dir, "workspace.dsl")
	copyTestFile(t, dslSrc, dslDst)

	runCLI(t, bin, dir, "import", "--from", "structurizr", "workspace.dsl")
	runCLI(t, bin, dir, "sync")

	drawioPath := filepath.Join(dir, "architecture.drawio")
	doc, err := drawio.LoadDocument(drawioPath)
	if err != nil {
		t.Fatalf("LoadDocument after sync: %v", err)
	}

	// ── Drill-down link assertions ────────────────────────────────────────
	// internetBankingSystem has a Container view → must carry a data:page/id link.
	// customer is a plain person with no deeper view → must NOT carry a link.
	drillDownFound := false
	customerHasLink := false

	for _, page := range doc.Pages() {
		xmlRoot := page.Root()

		// Check <object> elements for the link attribute (drill-down links are stored
		// in "link", not "tooltip" — tooltip holds the element description).
		for _, obj := range xmlRoot.SelectElements("object") {
			bsID := obj.SelectAttrValue("bausteinsicht_id", "")
			link := obj.SelectAttrValue("link", "")

			if strings.Contains(bsID, "internetBankingSystem") && !strings.Contains(bsID, ".") {
				if strings.Contains(link, "data:page/id") {
					drillDownFound = true
				}
			}
			if bsID == "customer" && strings.Contains(link, "data:page/id") {
				customerHasLink = true
			}
		}
	}

	if !drillDownFound {
		t.Error("internetBankingSystem: expected data:page/id drill-down link, found none")
	}
	if customerHasLink {
		t.Error("customer: unexpectedly has a data:page/id drill-down link (no detail view)")
	}

	// ── Back-nav button assertion ─────────────────────────────────────────
	// At least one page should contain a back-nav button (detail view pages).
	// Back-nav buttons are <object id="nav-back-..."> elements, not bare <mxCell>.
	backNavFound := false
	for _, page := range doc.Pages() {
		xmlRoot := page.Root()
		for _, obj := range xmlRoot.SelectElements("object") {
			id := obj.SelectAttrValue("id", "")
			if strings.Contains(id, "nav-back") {
				backNavFound = true
				break
			}
		}
		if backNavFound {
			break
		}
	}
	if !backNavFound {
		t.Error("no back-nav button (<object id='nav-back-...'>) found on any detail view page")
	}

	t.Logf("drill-down link check: internetBankingSystem has link=%v, customer has link=%v",
		drillDownFound, customerHasLink)
}
