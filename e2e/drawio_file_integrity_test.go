package e2e

// TestDrawioFileIntegrity (#519) closes E2E-Test-Plan.adoc section 9 (5
// tests, previously zero e2e coverage): structural invariants of the
// generated draw.io XML across repeated sync cycles — well-formedness,
// no orphaned connectors after element deletion, consistent per-kind
// styling, and base mxGraphModel/mxCell structure staying intact.
//
// Well-formedness is checked by reparsing with the same etree-based parser
// the CLI itself uses (internal/drawio.LoadDocument); xmllint is used
// additionally when available on PATH for an independent, stricter check,
// but isn't a hard requirement (mirrors this suite's existing pattern of
// gracefully degrading when optional external tools are absent).

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/drawio"
)

// assertWellFormedXML reparses the draw.io file with etree, and additionally
// with xmllint if it's on PATH, failing the test if either rejects it.
func assertWellFormedXML(t *testing.T, drawioPath string) {
	t.Helper()
	if _, err := drawio.LoadDocument(drawioPath); err != nil {
		t.Fatalf("draw.io file is not well-formed XML: %v", err)
	}
	if xmllintPath, err := exec.LookPath("xmllint"); err == nil {
		out, err := exec.Command(xmllintPath, "--noout", drawioPath).CombinedOutput()
		if err != nil {
			t.Fatalf("xmllint rejected draw.io file: %v\n%s", err, out)
		}
	}
}

func TestDrawioFileIntegrity(t *testing.T) {
	t.Run("9.1_WellFormedAfter10Cycles", test9_1WellFormedAfter10Cycles)
	t.Run("9.2_NoOrphanedConnectors", test9_2NoOrphanedConnectors)
	t.Run("9.3_StyleConsistency", test9_3StyleConsistency)
	t.Run("9.4_PageStructureBaseCells", test9_4PageStructureBaseCells)
	t.Run("9.5_MxGraphModelAttributes", test9_5MxGraphModelAttributes)
}

// test9_1WellFormedAfter10Cycles covers 9.1: 10 rounds of add+sync, checking
// well-formedness after every single cycle (not just at the end).
func test9_1WellFormedAfter10Cycles(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	for i := 1; i <= 10; i++ {
		id := fmt.Sprintf("cyclemember%d", i)
		runCLI(t, bin, dir, "add", "element", "--parent", "onlineshop", "--id", id, "--kind", "ui", "--title", fmt.Sprintf("Cycle Member %d", i))
		runCLI(t, bin, dir, "sync")
		assertWellFormedXML(t, dir+"/architecture.drawio")
	}
}

// test9_2NoOrphanedConnectors covers 9.2: removing an element that has
// relationships (and the relationships themselves, as a valid model must)
// leaves no dangling connector referencing the deleted element's cell.
func test9_2NoOrphanedConnectors(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	writeFile(t, dir+"/architecture.jsonc", `{
  "specification": {
    "elements": { "system": { "notation": "System" } },
    "relationships": { "uses": { "notation": "uses" } }
  },
  "model": {
    "nodeA": { "kind": "system", "title": "A" },
    "nodeB": { "kind": "system", "title": "B" },
    "nodeC": { "kind": "system", "title": "C" }
  },
  "relationships": [
    { "from": "nodeA", "to": "nodeB", "label": "r1", "kind": "uses" },
    { "from": "nodeB", "to": "nodeC", "label": "r2", "kind": "uses" }
  ],
  "config": { "metadata": false, "legend": false },
  "views": {
    "main": { "title": "Main", "include": ["*"] }
  }
}
`)
	runCLI(t, bin, dir, "sync")

	doc, err := drawio.LoadDocument(dir + "/architecture.drawio")
	if err != nil {
		t.Fatalf("LoadDocument: %v", err)
	}
	page := mustGetPage(t, doc, "main")
	if len(page.FindAllConnectors()) != 2 {
		t.Fatalf("expected 2 connectors before deletion, got %d", len(page.FindAllConnectors()))
	}

	// Remove the middle element AND its relationships (a model referencing
	// relationships to a nonexistent element fails validation — see section
	// 2 test 2.5 — so a clean removal must drop both together).
	writeFile(t, dir+"/architecture.jsonc", `{
  "specification": {
    "elements": { "system": { "notation": "System" } },
    "relationships": { "uses": { "notation": "uses" } }
  },
  "model": {
    "nodeA": { "kind": "system", "title": "A" },
    "nodeC": { "kind": "system", "title": "C" }
  },
  "relationships": [],
  "config": { "metadata": false, "legend": false },
  "views": {
    "main": { "title": "Main", "include": ["*"] }
  }
}
`)
	runCLI(t, bin, dir, "sync")

	doc2, err := drawio.LoadDocument(dir + "/architecture.drawio")
	if err != nil {
		t.Fatalf("LoadDocument after deletion: %v", err)
	}
	page2 := mustGetPage(t, doc2, "main")
	assertAbsent(t, page2, "nodeB")
	assertPresent(t, page2, "nodeA", "nodeC")

	for _, conn := range page2.FindAllConnectors() {
		src := conn.SelectAttrValue("source", "")
		tgt := conn.SelectAttrValue("target", "")
		if strings.Contains(src, "nodeB") || strings.Contains(tgt, "nodeB") {
			t.Errorf("orphaned connector references deleted nodeB: source=%q target=%q", src, tgt)
		}
	}
	if got := len(page2.FindAllConnectors()); got != 0 {
		t.Errorf("expected 0 connectors after removing nodeB and both its relationships, got %d", got)
	}
}

// test9_3StyleConsistency covers 9.3: elements of the same kind get
// identical styles.
func test9_3StyleConsistency(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	var ids []string
	for i := 1; i <= 5; i++ {
		id := fmt.Sprintf("styletest%d", i)
		ids = append(ids, id)
		runCLI(t, bin, dir, "add", "element", "--parent", "onlineshop", "--id", id, "--kind", "ui", "--title", fmt.Sprintf("Style Test %d", i))
	}
	runCLI(t, bin, dir, "sync")

	doc, err := drawio.LoadDocument(dir + "/architecture.drawio")
	if err != nil {
		t.Fatalf("LoadDocument: %v", err)
	}
	var page *drawio.Page
	for _, p := range doc.Pages() {
		if p.FindElement("onlineshop."+ids[0]) != nil {
			page = p
			break
		}
	}
	if page == nil {
		t.Fatal("styletest elements not found on any page")
	}

	var styles []string
	for _, id := range ids {
		obj := page.FindElement("onlineshop." + id)
		if obj == nil {
			t.Fatalf("element onlineshop.%s not found on page", id)
		}
		cell := obj.SelectElement("mxCell")
		if cell == nil {
			t.Fatalf("onlineshop.%s has no mxCell child", id)
		}
		styles = append(styles, cell.SelectAttrValue("style", ""))
	}
	for i := 1; i < len(styles); i++ {
		if styles[i] != styles[0] {
			t.Errorf("style mismatch: %s[0]=%q vs [%d]=%q", ids[0], styles[0], i, styles[i])
		}
	}
}

// test9_4PageStructureBaseCells covers 9.4: after 10 cycles, every page
// still has exactly one mxCell id="0" and one id="1" (the draw.io base
// layer cells), not duplicated by repeated forward-sync passes.
func test9_4PageStructureBaseCells(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	for i := 1; i <= 10; i++ {
		id := fmt.Sprintf("basecelltest%d", i)
		runCLI(t, bin, dir, "add", "element", "--parent", "onlineshop", "--id", id, "--kind", "ui", "--title", fmt.Sprintf("Base Cell Test %d", i))
		runCLI(t, bin, dir, "sync")
	}

	doc, err := drawio.LoadDocument(dir + "/architecture.drawio")
	if err != nil {
		t.Fatalf("LoadDocument: %v", err)
	}
	for _, page := range doc.Pages() {
		var zeroCount, oneCount int
		for _, cell := range page.Root().SelectElements("mxCell") {
			switch cell.SelectAttrValue("id", "") {
			case "0":
				zeroCount++
			case "1":
				oneCount++
			}
		}
		if zeroCount != 1 {
			t.Errorf("page %q: mxCell id=0 count = %d, want 1", page.ID(), zeroCount)
		}
		if oneCount != 1 {
			t.Errorf("page %q: mxCell id=1 count = %d, want 1", page.ID(), oneCount)
		}
	}
}

// test9_5MxGraphModelAttributes covers 9.5: the mxGraphModel element's
// canvas attributes (dx, dy, grid, gridSize, page, pageWidth, pageHeight,
// background) survive 10 sync cycles unchanged/present, not stripped by
// repeated forward-sync passes.
func test9_5MxGraphModelAttributes(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	for i := 1; i <= 10; i++ {
		id := fmt.Sprintf("mxgraphtest%d", i)
		runCLI(t, bin, dir, "add", "element", "--parent", "onlineshop", "--id", id, "--kind", "ui", "--title", fmt.Sprintf("MxGraph Test %d", i))
		runCLI(t, bin, dir, "sync")
	}

	raw, err := os.ReadFile(dir + "/architecture.drawio")
	if err != nil {
		t.Fatalf("read draw.io: %v", err)
	}
	doc, err := drawio.LoadDocument(dir + "/architecture.drawio")
	if err != nil {
		t.Fatalf("LoadDocument: %v", err)
	}
	requiredAttrs := []string{"grid", "gridSize", "page", "pageWidth", "pageHeight", "background"}
	for _, page := range doc.Pages() {
		mgm := page.Root().Parent()
		if mgm == nil || mgm.Tag != "mxGraphModel" {
			t.Fatalf("page %q: root's parent is not mxGraphModel", page.ID())
		}
		for _, attr := range requiredAttrs {
			if mgm.SelectAttrValue(attr, "") == "" {
				t.Errorf("page %q: mxGraphModel missing/empty attribute %q\nfull file follows:\n%s", page.ID(), attr, string(raw))
			}
		}
	}
}
