package sync

import (
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/drawio"
)

// duplicateElementOnPage copies the cell of the element the way draw.io does
// on copy and paste: same attributes, new cell ID, shifted position.
func duplicateElementOnPage(t *testing.T, page *drawio.Page, id string) {
	t.Helper()
	orig := page.FindElement(id)
	if orig == nil {
		t.Fatalf("element %q not on page %q", id, page.Name())
	}
	cp := orig.Copy()
	cp.CreateAttr("id", orig.SelectAttrValue("id", "")+"-copy")
	page.Root().AddChild(cp)
}

func countElementsWithID(page *drawio.Page, id string) int {
	n := 0
	for _, obj := range page.FindAllElements() {
		if obj.SelectAttrValue(attrBausteinsichtID, "") == id {
			n++
		}
	}
	return n
}

// #722: a copy of an element cell on the same page carries the same
// bausteinsicht_id. It is reported, and the copy becomes an unmanaged shape.
func TestSync_DuplicateBausteinsichtIDOnOnePageIsReportedAndStripped(t *testing.T) {
	m := liftingModel()
	ts := minimalTemplates(t)
	doc := shopContainersDoc()
	Run(m, doc, emptyState(), ts, nil)
	page := requirePage(t, doc, "view-containers")
	state := buildTestState(t, m, doc)
	duplicateElementOnPage(t, page, "shop.web")
	if got := countElementsWithID(page, "shop.web"); got != 2 {
		t.Fatalf("setup: %d cells with the ID, want 2", got)
	}

	res := Run(m, doc, state, ts, nil)

	if got := countElementsWithID(page, "shop.web"); got != 1 {
		t.Errorf("%d cells keep bausteinsicht_id \"shop.web\", want 1 (the copy is unmanaged)", got)
	}
	found := false
	for _, w := range res.Warnings {
		if strings.Contains(w, `duplicate bausteinsicht_id "shop.web"`) && strings.Contains(w, `"Container View"`) {
			found = true
		}
	}
	if !found {
		t.Errorf("no warning about the duplicate ID, warnings: %q", res.Warnings)
	}
	if _, ok := m.Model["shop"].Children["web"]; !ok {
		t.Error("the original element vanished from the model")
	}

	// The next run has nothing left to report.
	again := Run(m, doc, buildTestState(t, m, doc), ts, nil)
	for _, w := range again.Warnings {
		if strings.Contains(w, "duplicate bausteinsicht_id") {
			t.Errorf("the duplicate is reported again: %s", w)
		}
	}
}

