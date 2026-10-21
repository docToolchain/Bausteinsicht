package sync

import (
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// TestComputeNestedLayout_MultiLevelContainment verifies that the "nested"
// layout parents each element to its IMMEDIATE container (not the scope) and
// sizes the intermediate containers — the basis for concentric-ring rendering.
func TestComputeNestedLayout_MultiLevelContainment(t *testing.T) {
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
				"outer": {Kind: "zone", Title: "Outer", Children: map[string]model.Element{
					"a": {Kind: "leaf", Title: "A", Metadata: map[string]string{"side": "top"}},
					"inner": {Kind: "zone", Title: "Inner", Children: map[string]model.Element{
						"b": {Kind: "leaf", Title: "B", Metadata: map[string]string{"side": "bottom"}},
					}},
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
		"ordering.outer",
		"ordering.outer.a",
		"ordering.outer.inner",
		"ordering.outer.inner.b",
	}

	lr := computeNestedLayout(ids, flat, ts, "ordering")

	// Each element is parented to its immediate container, not the scope.
	want := map[string]string{
		"ordering.outer":         "ordering",
		"ordering.outer.a":       "ordering.outer",
		"ordering.outer.inner":   "ordering.outer",
		"ordering.outer.inner.b": "ordering.outer.inner",
	}
	for id, exp := range want {
		if lr.ParentOf[id] != exp {
			t.Errorf("ParentOf[%s] = %q, want %q", id, lr.ParentOf[id], exp)
		}
	}

	// Intermediate containers get a computed size.
	for _, c := range []string{"ordering.outer", "ordering.outer.inner"} {
		if sz, ok := lr.ContainerSizes[c]; !ok || sz.W <= 0 || sz.H <= 0 {
			t.Errorf("ContainerSizes[%s] missing or non-positive: %+v (ok=%v)", c, sz, ok)
		}
	}

	// The scope is sized via BoundaryWidth/Height, not ContainerSizes.
	if _, ok := lr.ContainerSizes["ordering"]; ok {
		t.Errorf("scope should be sized via Boundary*, not ContainerSizes")
	}
	if lr.BoundaryWidth <= 0 || lr.BoundaryHeight <= 0 {
		t.Errorf("scope boundary not sized: w=%v h=%v", lr.BoundaryWidth, lr.BoundaryHeight)
	}

	// A container fully contains its inner ring: the inner container plus its
	// relative offset must fit inside the outer container's box.
	outer := lr.ContainerSizes["ordering.outer"]
	innerPos := lr.Positions["ordering.outer.inner"]
	inner := lr.ContainerSizes["ordering.outer.inner"]
	if innerPos.X+inner.W > outer.W || innerPos.Y+inner.H > outer.H {
		t.Errorf("inner ring overflows outer container: inner at %+v size %+v, outer %+v", innerPos, inner, outer)
	}
}
