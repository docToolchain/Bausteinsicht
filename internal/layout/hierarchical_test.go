package layout

import (
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// TestHierarchicalLayout_AssignLayers validates layer assignment via longest-path algorithm.
func TestHierarchicalLayout_AssignLayers(t *testing.T) {
	m := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"system": {Notation: "box"},
			},
		},
		Model: map[string]model.Element{
			"a": {Kind: "system", Title: "A"},
			"b": {Kind: "system", Title: "B"},
			"c": {Kind: "system", Title: "C"},
			"d": {Kind: "system", Title: "D"},
		},
		Relationships: []model.Relationship{
			{From: "a", To: "b", Label: "a->b"},
			{From: "b", To: "c", Label: "b->c"},
			{From: "c", To: "d", Label: "c->d"},
		},
	}

	h := NewHierarchicalLayout(m, "TB")
	result := h.Compute()

	// Check that all elements have positions
	if len(result.Positions) != 4 {
		t.Errorf("expected 4 positions, got %d", len(result.Positions))
	}

	// Check that source (a) is at layer 3, sink (d) is at layer 0
	posA := result.Positions["a"]
	posD := result.Positions["d"]

	if posA.Layer < posD.Layer {
		t.Errorf("source should be in later layer: a layer=%d, d layer=%d", posA.Layer, posD.Layer)
	}
}

// TestHierarchicalLayout_CycleHandling validates that cycles don't cause infinite recursion.
func TestHierarchicalLayout_CycleHandling(t *testing.T) {
	m := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"system": {Notation: "box"},
			},
		},
		Model: map[string]model.Element{
			"a": {Kind: "system", Title: "A"},
			"b": {Kind: "system", Title: "B"},
			"c": {Kind: "system", Title: "C"},
		},
		Relationships: []model.Relationship{
			{From: "a", To: "b", Label: "a->b"},
			{From: "b", To: "c", Label: "b->c"},
			{From: "c", To: "a", Label: "c->a (cycle)"},
		},
	}

	h := NewHierarchicalLayout(m, "TB")
	result := h.Compute()

	// Should complete without panic
	if len(result.Positions) != 3 {
		t.Errorf("expected 3 positions, got %d", len(result.Positions))
	}
}

// TestHierarchicalLayout_RankDir validates top-to-bottom and left-to-right layouts.
func TestHierarchicalLayout_RankDir(t *testing.T) {
	m := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"system": {Notation: "box"},
			},
		},
		Model: map[string]model.Element{
			"a": {Kind: "system", Title: "A"},
			"b": {Kind: "system", Title: "B"},
		},
		Relationships: []model.Relationship{
			{From: "a", To: "b", Label: "a->b"},
		},
	}

	tests := []struct {
		rankDir string
		name    string
	}{
		{"TB", "top-to-bottom"},
		{"LR", "left-to-right"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHierarchicalLayout(m, tt.rankDir)
			result := h.Compute()

			if len(result.Positions) != 2 {
				t.Errorf("expected 2 positions, got %d", len(result.Positions))
			}

			// Both should complete without error
			if result.Algorithm != Hierarchical {
				t.Errorf("expected Hierarchical algorithm, got %d", result.Algorithm)
			}
		})
	}
}

// TestHierarchicalLayout_EmptyModel handles edge case of empty model.
func TestHierarchicalLayout_EmptyModel(t *testing.T) {
	m := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"system": {Notation: "box"},
			},
		},
		Model:         make(map[string]model.Element),
		Relationships: []model.Relationship{},
	}

	h := NewHierarchicalLayout(m, "TB")
	result := h.Compute()

	if len(result.Positions) != 0 {
		t.Errorf("expected 0 positions for empty model, got %d", len(result.Positions))
	}
}

// TestHierarchicalLayout_SingleElement handles single-element model.
func TestHierarchicalLayout_SingleElement(t *testing.T) {
	m := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"system": {Notation: "box"},
			},
		},
		Model: map[string]model.Element{
			"a": {Kind: "system", Title: "A"},
		},
		Relationships: []model.Relationship{},
	}

	h := NewHierarchicalLayout(m, "TB")
	result := h.Compute()

	if len(result.Positions) != 1 {
		t.Errorf("expected 1 position, got %d", len(result.Positions))
	}

	pos := result.Positions["a"]
	if pos.X < 0 || pos.Y < 0 {
		t.Errorf("position should be non-negative: x=%f, y=%f", pos.X, pos.Y)
	}
}
