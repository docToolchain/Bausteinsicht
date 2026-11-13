package sync

import (
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/drawio"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// TestAncestorContainersWithinScope covers the nested-layout helper that lists
// the intermediate containers between a scope and a deep element, used so a
// scoped element included without its parents still gets them placed.
func TestAncestorContainersWithinScope(t *testing.T) {
	flat := map[string]*model.Element{
		"a":       {},
		"a.b":     {},
		"a.b.c":   {},
		"a.b.c.d": {},
	}

	tests := []struct {
		name    string
		id      string
		scopeID string
		flat    map[string]*model.Element
		want    []string
	}{
		{
			name:    "deep element returns outer-to-inner containers",
			id:      "a.b.c.d",
			scopeID: "a",
			flat:    flat,
			want:    []string{"a.b", "a.b.c"},
		},
		{
			name:    "direct child has no intermediate containers",
			id:      "a.b",
			scopeID: "a",
			flat:    flat,
			want:    nil,
		},
		{
			name:    "intermediate missing from flat is skipped",
			id:      "a.b.c.d",
			scopeID: "a",
			flat: map[string]*model.Element{
				"a":       {},
				"a.b.c":   {},
				"a.b.c.d": {},
			},
			want: []string{"a.b.c"},
		},
		{
			name:    "empty scope returns nil",
			id:      "a.b.c",
			scopeID: "",
			flat:    flat,
			want:    nil,
		},
		{
			name:    "element outside the scope returns nil",
			id:      "x.y.z",
			scopeID: "a",
			flat:    flat,
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ancestorContainersWithinScope(tt.id, tt.scopeID, tt.flat)
			if !equalStringSlices(got, tt.want) {
				t.Errorf("ancestorContainersWithinScope(%q, %q) = %v, want %v",
					tt.id, tt.scopeID, got, tt.want)
			}
		})
	}
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestSetCellParent verifies the nested-layout re-parenting helper rewrites the
// mxCell "parent" attribute, and is a safe no-op for an unknown element id.
func TestSetCellParent(t *testing.T) {
	doc := drawio.NewDocument()
	page := doc.AddPage("p1", "Page 1")
	_ = page.CreateElement(drawio.ElementData{
		ID:    "shop.api",
		Kind:  "container",
		Title: "API",
	}, "")

	setCellParent(page, "shop.api", "shop-boundary")

	cell := page.FindElement("shop.api").FindElement("mxCell")
	if got := cell.SelectAttrValue("parent", ""); got != "shop-boundary" {
		t.Errorf("parent = %q, want %q", got, "shop-boundary")
	}

	// Unknown element: must not panic and must leave the page untouched.
	setCellParent(page, "does.not.exist", "whatever")
}

// TestSetCellSize verifies the nested-layout sizing helper rewrites the
// mxGeometry width/height, and is a safe no-op for an unknown element id.
func TestSetCellSize(t *testing.T) {
	doc := drawio.NewDocument()
	page := doc.AddPage("p1", "Page 1")
	_ = page.CreateElement(drawio.ElementData{
		ID:    "shop.api",
		Kind:  "container",
		Title: "API",
	}, "")

	setCellSize(page, "shop.api", 320, 240)

	geo := page.FindElement("shop.api").FindElement("mxCell").FindElement("mxGeometry")
	if got := geo.SelectAttrValue("width", ""); got != "320" {
		t.Errorf("width = %q, want %q", got, "320")
	}
	if got := geo.SelectAttrValue("height", ""); got != "240" {
		t.Errorf("height = %q, want %q", got, "240")
	}

	// Unknown element: must not panic and must leave the page untouched.
	setCellSize(page, "does.not.exist", 10, 10)
}

// TestApplyForward_NestedLayout_PlacesAncestorsAndReparents drives the nested
// layout through the real ApplyForward pipeline. It covers the two blocks the
// nested feature adds to the forward engine: Block A (applyForwardToView closes
// the resolved set under missing ancestor containers) and Block B
// (placeElementsWithLayout re-parents each element to its immediate container's
// scoped cell instead of the scope boundary).
func TestApplyForward_NestedLayout_PlacesAncestorsAndReparents(t *testing.T) {
	doc := drawio.NewDocument()
	doc.AddPage("view-nested", "Nested View")
	ts := minimalTemplates(t)

	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"ordering": {Kind: "container", Title: "Ordering", Children: map[string]model.Element{
				"inner": {Kind: "container", Title: "Inner", Children: map[string]model.Element{
					"leaf": {Kind: "container", Title: "Leaf"},
				}},
			}},
		},
		Views: map[string]model.View{
			"nested": {
				Title:  "Nested View",
				Scope:  "ordering",
				Layout: "nested",
				// Deliberately include only the DEEP leaf, omitting its
				// intermediate container, to exercise Block A.
				Include: []string{"ordering.inner.leaf"},
			},
		},
	}

	cs := &ChangeSet{
		ModelElementChanges: []ElementChange{
			{ID: "ordering", Type: Added},
			{ID: "ordering.inner", Type: Added},
			{ID: "ordering.inner.leaf", Type: Added},
		},
	}

	ApplyForward(cs, doc, ts, m)

	page := requirePage(t, doc, "view-nested")

	// Block A: the intermediate container, though NOT in the include list, is
	// placed because it is an ancestor of the included deep leaf.
	inner := page.FindElement("ordering.inner")
	if inner == nil {
		t.Fatal("intermediate container 'ordering.inner' was not placed (Block A ancestor expansion)")
	}

	leaf := page.FindElement("ordering.inner.leaf")
	if leaf == nil {
		t.Fatal("'ordering.inner.leaf' was not placed")
	}

	// Block B: the leaf is re-parented to its immediate container's scoped cell,
	// not to the scope boundary.
	gotLeafParent := leaf.FindElement("mxCell").SelectAttrValue("parent", "")
	if want := "nested--ordering.inner"; gotLeafParent != want {
		t.Errorf("leaf cell parent = %q, want %q (nested re-parenting)", gotLeafParent, want)
	}

	// The intermediate container is itself contained by the scope boundary cell.
	gotInnerParent := inner.FindElement("mxCell").SelectAttrValue("parent", "")
	if want := "nested--ordering"; gotInnerParent != want {
		t.Errorf("inner container cell parent = %q, want %q", gotInnerParent, want)
	}
}

// TestComputeNestedLayout_NoScopeFallsBackToGrid covers the BR-014a safety net:
// nested mode without a scope is unreachable through validation, so the layout
// falls back to the grid layout rather than producing nothing.
func TestComputeNestedLayout_NoScopeFallsBackToGrid(t *testing.T) {
	m := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"box": {Notation: "Box"},
			},
		},
		Model: map[string]model.Element{
			"a": {Kind: "box", Title: "A"},
			"b": {Kind: "box", Title: "B"},
		},
	}
	flat, err := model.FlattenElements(m)
	if err != nil {
		t.Fatalf("FlattenElements: %v", err)
	}
	ts := loadTestTemplates(t)

	lr := computeNestedLayout([]string{"a", "b"}, flat, ts, "")

	// Grid fallback positions every element and parents none to a container.
	for _, id := range []string{"a", "b"} {
		if _, ok := lr.Positions[id]; !ok {
			t.Errorf("grid fallback did not position %q", id)
		}
	}
	if len(lr.ParentOf) != 0 {
		t.Errorf("grid fallback should not set ParentOf, got %v", lr.ParentOf)
	}
}

// TestComputeNestedLayout_MultipleInnerRings covers the inner-ring stacking
// branch: a container with more than one sub-container accumulates their
// heights with a gap between siblings.
func TestComputeNestedLayout_MultipleInnerRings(t *testing.T) {
	m := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"ctx":  {Notation: "Context", Container: true},
				"zone": {Notation: "Zone", Container: true},
				"leaf": {Notation: "Leaf"},
			},
		},
		Model: map[string]model.Element{
			"ordering": {Kind: "ctx", Title: "Ordering", Children: map[string]model.Element{
				"ring1": {Kind: "zone", Title: "Ring 1", Children: map[string]model.Element{
					"a": {Kind: "leaf", Title: "A"},
				}},
				"ring2": {Kind: "zone", Title: "Ring 2", Children: map[string]model.Element{
					"b": {Kind: "leaf", Title: "B"},
				}},
			}},
		},
	}
	flat, err := model.FlattenElements(m)
	if err != nil {
		t.Fatalf("FlattenElements: %v", err)
	}
	ts := loadTestTemplates(t)
	ids := []string{
		"ordering.ring1", "ordering.ring1.a",
		"ordering.ring2", "ordering.ring2.b",
	}

	lr := computeNestedLayout(ids, flat, ts, "ordering")

	// Both sibling rings are parented to the scope and sized.
	for _, ring := range []string{"ordering.ring1", "ordering.ring2"} {
		if lr.ParentOf[ring] != "ordering" {
			t.Errorf("ParentOf[%s] = %q, want %q", ring, lr.ParentOf[ring], "ordering")
		}
		if sz, ok := lr.ContainerSizes[ring]; !ok || sz.W <= 0 || sz.H <= 0 {
			t.Errorf("ContainerSizes[%s] missing or non-positive: %+v (ok=%v)", ring, sz, ok)
		}
	}

	// Stacked vertically: the two rings must not overlap on the Y axis.
	p1, p2 := lr.Positions["ordering.ring1"], lr.Positions["ordering.ring2"]
	s1 := lr.ContainerSizes["ordering.ring1"]
	s2 := lr.ContainerSizes["ordering.ring2"]
	if p1.Y < p2.Y {
		if p1.Y+s1.H > p2.Y {
			t.Errorf("rings overlap: ring1 %+v size %+v, ring2 %+v", p1, s1, p2)
		}
	} else if p2.Y+s2.H > p1.Y {
		t.Errorf("rings overlap: ring2 %+v size %+v, ring1 %+v", p2, s2, p1)
	}
}
