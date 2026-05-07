package graph

import (
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

func TestAnalyzerNoCycles(t *testing.T) {
	m := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"component": {Notation: "box"},
			},
		},
		Model: map[string]model.Element{
			"a": {Kind: "component", Title: "A"},
			"b": {Kind: "component", Title: "B"},
			"c": {Kind: "component", Title: "C"},
		},
		Relationships: []model.Relationship{
			{From: "a", To: "b"},
			{From: "b", To: "c"},
		},
	}

	a := NewAnalyzer(m)
	result := a.Analyze()

	if len(result.Cycles) > 0 {
		t.Errorf("expected no cycles, got %d", len(result.Cycles))
	}

	if !result.IDAGValid {
		t.Error("graph should be a valid DAG")
	}

	if result.ElementCount != 3 {
		t.Errorf("expected 3 elements, got %d", result.ElementCount)
	}

	if result.MaxDepth != 2 {
		t.Errorf("expected max depth 2, got %d", result.MaxDepth)
	}
}

func TestAnalyzerWithCycle(t *testing.T) {
	m := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"component": {Notation: "box"},
			},
		},
		Model: map[string]model.Element{
			"a": {Kind: "component", Title: "A"},
			"b": {Kind: "component", Title: "B"},
			"c": {Kind: "component", Title: "C"},
		},
		Relationships: []model.Relationship{
			{From: "a", To: "b"},
			{From: "b", To: "c"},
			{From: "c", To: "a"}, // Creates cycle
		},
	}

	a := NewAnalyzer(m)
	result := a.Analyze()

	if len(result.Cycles) == 0 {
		t.Error("expected to find a cycle")
	}

	if result.IDAGValid {
		t.Error("graph should not be a valid DAG")
	}

	if result.Cycles[0].Length != 3 {
		t.Errorf("expected cycle length 3, got %d", result.Cycles[0].Length)
	}
}

func TestAnalyzerCentrality(t *testing.T) {
	m := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"component": {Notation: "box"},
			},
		},
		Model: map[string]model.Element{
			"hub":    {Kind: "component", Title: "Hub"},
			"dep1":   {Kind: "component", Title: "Dep1"},
			"dep2":   {Kind: "component", Title: "Dep2"},
		},
		Relationships: []model.Relationship{
			{From: "hub", To: "dep1"},
			{From: "hub", To: "dep2"},
		},
	}

	a := NewAnalyzer(m)
	result := a.Analyze()

	// Find hub's centrality
	var hubCentrality *Centrality
	for i := range result.Centrality {
		if result.Centrality[i].ID == "hub" {
			hubCentrality = &result.Centrality[i]
			break
		}
	}

	if hubCentrality == nil {
		t.Fatal("hub not found in centrality results")
	}

	if hubCentrality.OutDegree != 2 {
		t.Errorf("hub should have out-degree 2, got %d", hubCentrality.OutDegree)
	}

	if hubCentrality.InDegree != 0 {
		t.Errorf("hub should have in-degree 0, got %d", hubCentrality.InDegree)
	}
}

func TestAnalyzerMaxDepth(t *testing.T) {
	m := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"component": {Notation: "box"},
			},
		},
		Model: map[string]model.Element{
			"a": {Kind: "component", Title: "A"},
			"b": {Kind: "component", Title: "B"},
			"c": {Kind: "component", Title: "C"},
			"d": {Kind: "component", Title: "D"},
		},
		Relationships: []model.Relationship{
			{From: "a", To: "b"},
			{From: "b", To: "c"},
			{From: "c", To: "d"},
		},
	}

	a := NewAnalyzer(m)
	result := a.Analyze()

	if result.MaxDepth != 3 {
		t.Errorf("expected max depth 3, got %d", result.MaxDepth)
	}
}

func TestAnalyzerStronglyConnectedComponents(t *testing.T) {
	m := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"component": {Notation: "box"},
			},
		},
		Model: map[string]model.Element{
			"a": {Kind: "component", Title: "A"},
			"b": {Kind: "component", Title: "B"},
			"c": {Kind: "component", Title: "C"},
		},
		Relationships: []model.Relationship{
			{From: "a", To: "b"},
			{From: "b", To: "c"},
			{From: "c", To: "a"}, // cycle
		},
	}

	a := NewAnalyzer(m)
	result := a.Analyze()

	// Should have one cycle component
	hasCycleComponent := false
	for _, comp := range result.Components {
		if comp.IsCycle && len(comp.Elements) == 3 {
			hasCycleComponent = true
			break
		}
	}

	if !hasCycleComponent {
		t.Error("expected to find a cycle component with 3 elements")
	}
}

func TestAnalyzerEmptyModel(t *testing.T) {
	m := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: make(map[string]model.ElementKind),
		},
		Model:         make(map[string]model.Element),
		Relationships: []model.Relationship{},
	}

	a := NewAnalyzer(m)
	result := a.Analyze()

	if result.ElementCount != 0 {
		t.Errorf("expected 0 elements, got %d", result.ElementCount)
	}

	if result.MaxDepth != 0 {
		t.Errorf("expected max depth 0, got %d", result.MaxDepth)
	}
}
