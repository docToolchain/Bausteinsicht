package layout

import (
	"testing"

	"github.com/beevik/etree"
	"github.com/docToolchain/Bausteinsicht/internal/drawio"
)

// TestApply_UpdatesRealElementStructure reproduces the bug where Apply silently
// updates nothing on a real Bausteinsicht diagram. drawio.Page.CreateElement (used
// by sync for every element it creates) nests <mxGeometry> two levels deep —
// <object bausteinsicht_id="..."><mxCell><mxGeometry/></mxCell></object> — but
// updateElementPosition only checked the object's *direct* children for an
// mxGeometry tag, so it never found one and silently left every element
// untouched (Apply()'s caller swallows the "not found" error). Regression test
// for #604.
func TestApply_UpdatesRealElementStructure(t *testing.T) {
	doc := drawio.NewDocument()
	page := doc.AddPage("page1", "Page 1")
	if err := page.CreateElement(drawio.ElementData{
		ID:     "svc",
		Kind:   "container",
		Title:  "Service",
		X:      10,
		Y:      20,
		Width:  180,
		Height: 60,
	}, "rounded=1"); err != nil {
		t.Fatalf("CreateElement failed: %v", err)
	}

	result := LayoutResult{
		Positions: map[string]ElementPosition{
			"svc": {ID: "svc", X: 999, Y: 888, Width: 200, Height: 100},
		},
		Algorithm: Hierarchical,
	}

	if err := Apply(doc, result, false); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	geom := findGeometry(t, page, "svc")
	if x := geom.SelectAttrValue("x", ""); x != "999" {
		t.Errorf("x = %q, want \"999\" — Apply did not update the real object>mxCell>mxGeometry structure", x)
	}
	if y := geom.SelectAttrValue("y", ""); y != "888" {
		t.Errorf("y = %q, want \"888\"", y)
	}
}

func findGeometry(t *testing.T, page *drawio.Page, bausteinsichtID string) *etree.Element {
	t.Helper()
	root := page.Root()
	if root == nil {
		t.Fatal("page has no root")
	}
	var found *etree.Element
	walkElements(root, func(elem *etree.Element) {
		if found != nil {
			return
		}
		for _, attr := range elem.Attr {
			if attr.Key == "bausteinsicht_id" && attr.Value == bausteinsichtID {
				for _, child := range elem.ChildElements() {
					if g := findGeometryRecursive(child); g != nil {
						found = g
						return
					}
				}
			}
		}
	})
	if found == nil {
		t.Fatalf("no mxGeometry found for %s", bausteinsichtID)
	}
	return found
}

func findGeometryRecursive(elem *etree.Element) *etree.Element {
	if elem.Tag == "mxGeometry" {
		return elem
	}
	for _, child := range elem.ChildElements() {
		if g := findGeometryRecursive(child); g != nil {
			return g
		}
	}
	return nil
}
