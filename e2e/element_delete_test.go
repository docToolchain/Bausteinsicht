package e2e

import (
	"strings"
	"testing"

	"github.com/beevik/etree"
)

// deleteElementShapes does what deleting a shape in draw.io does: every cell of
// the element on every page (the object, its text cells) and the connectors
// attached to it go; it returns how many shapes were removed.
func deleteElementShapes(t *testing.T, path, elementID string) int {
	t.Helper()
	doc := etree.NewDocument()
	if err := doc.ReadFromFile(path); err != nil {
		t.Fatal(err)
	}
	removed := 0
	for _, diagram := range doc.FindElements("//diagram") {
		for _, obj := range diagram.FindElements(".//object[@bausteinsicht_id='" + elementID + "']") {
			cellID := obj.SelectAttrValue("id", "")
			for _, cell := range diagram.FindElements(".//mxCell") {
				id := cell.SelectAttrValue("id", "")
				if strings.HasPrefix(id, cellID+"-") || cell.SelectAttrValue("source", "") == cellID || cell.SelectAttrValue("target", "") == cellID || cell.SelectAttrValue("parent", "") == cellID {
					cell.Parent().RemoveChild(cell)
				}
			}
			obj.Parent().RemoveChild(obj)
			removed++
		}
	}
	if err := doc.WriteToFile(path); err != nil {
		t.Fatal(err)
	}
	return removed
}

// TestSyncAfterElementDeletedInDrawio (#720): deleting an element on every page
// in draw.io removes it from the model together with everything that refers to
// it. Before, the dynamic-view steps stayed behind and the model no longer
// validated, so validate, the exports and the next sync all failed.
func TestSyncAfterElementDeletedInDrawio(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")
	runCLI(t, bin, dir, "sync")

	if n := deleteElementShapes(t, dir+"/architecture.drawio", "onlineshop.cache"); n == 0 {
		t.Fatal("setup: the element was not found in the diagram")
	}
	out := runCLI(t, bin, dir, "sync")
	if !strings.Contains(out, `"onlineshop.cache" was deleted in draw.io and removed from model`) || !strings.Contains(out, "dynamicViews[") {
		t.Errorf("the warning must name the element and the dropped dynamic-view steps:\n%s", out)
	}

	if out := runCLI(t, bin, dir, "validate"); !strings.Contains(out, "valid") {
		t.Errorf("validate after the deletion:\n%s", out)
	}
	for i := 1; i <= 3; i++ {
		if out := runCLI(t, bin, dir, "sync"); !strings.Contains(out, "Already in sync") {
			t.Fatalf("sync %d after the deletion is not a no-op:\n%s", i, out)
		}
	}
	runCLI(t, bin, dir, "export-diagram")
	runCLI(t, bin, dir, "export-sequence")
}
