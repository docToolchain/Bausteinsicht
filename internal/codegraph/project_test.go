package codegraph

import (
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// threeTierModel mirrors the worked example from #562/ADR-013: api -> svc,
// svc -> db, each anchored to its own Java package.
func threeTierModel() *model.BausteinsichtModel {
	return &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"component": {Notation: "Component"},
			},
			Relationships: map[string]model.RelationshipKind{
				"uses": {Notation: "uses"},
			},
		},
		Model: map[string]model.Element{
			"api": {
				Kind: "component", Title: "API",
				CodeMapping: &model.CodeMapping{Packages: []string{"com.acme.backend.api"}},
			},
			"svc": {
				Kind: "component", Title: "Service",
				CodeMapping: &model.CodeMapping{Packages: []string{"com.acme.backend.svc"}},
			},
			"db": {
				Kind: "component", Title: "Repository",
				CodeMapping: &model.CodeMapping{Packages: []string{"com.acme.backend.db"}},
			},
		},
		Relationships: []model.Relationship{
			{From: "api", To: "svc", Kind: "uses"},
			{From: "svc", To: "db", Kind: "uses"},
		},
	}
}

func TestProject_MapsPackageEdgesToElements(t *testing.T) {
	cg := &CodeGraph{
		Language: "java",
		Nodes: []Node{
			{ID: "com.acme.backend.api", Kind: "package"},
			{ID: "com.acme.backend.svc", Kind: "package"},
			{ID: "com.acme.backend.db", Kind: "package"},
		},
		Edges: []Edge{
			{From: "com.acme.backend.api", To: "com.acme.backend.svc"},
			{From: "com.acme.backend.svc", To: "com.acme.backend.db"},
		},
	}

	result, err := Project(cg, threeTierModel())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Snapshot.Elements) != 3 {
		t.Errorf("expected 3 elements, got %d: %v", len(result.Snapshot.Elements), result.Snapshot.Elements)
	}
	if len(result.Snapshot.Relationships) != 2 {
		t.Fatalf("expected 2 relationships, got %d: %v", len(result.Snapshot.Relationships), result.Snapshot.Relationships)
	}
	if len(result.UnmappedPackages) != 0 {
		t.Errorf("expected no unmapped packages, got %v", result.UnmappedPackages)
	}
}

// TestProject_DriftEdgeSurfacesAsNewRelationship is #562's own worked
// example: real code adds api -> db, which the model doesn't have. That
// extra dependency must appear in the asIs snapshot so `diff` surfaces it.
func TestProject_DriftEdgeSurfacesAsNewRelationship(t *testing.T) {
	cg := &CodeGraph{
		Language: "java",
		Nodes: []Node{
			{ID: "com.acme.backend.api", Kind: "package"},
			{ID: "com.acme.backend.svc", Kind: "package"},
			{ID: "com.acme.backend.db", Kind: "package"},
		},
		Edges: []Edge{
			{From: "com.acme.backend.api", To: "com.acme.backend.svc"},
			{From: "com.acme.backend.svc", To: "com.acme.backend.db"},
			{From: "com.acme.backend.api", To: "com.acme.backend.db"}, // drift: not in the model
		},
	}

	result, err := Project(cg, threeTierModel())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, r := range result.Snapshot.Relationships {
		if r.From == "api" && r.To == "db" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected drift relationship api -> db in asIs snapshot, got %v", result.Snapshot.Relationships)
	}
}

func TestProject_IntraElementEdgeDropped(t *testing.T) {
	cg := &CodeGraph{
		Nodes: []Node{
			{ID: "com.acme.backend.api", Kind: "package"},
			{ID: "com.acme.backend.api.internal", Kind: "package"},
		},
		Edges: []Edge{
			// both packages resolve to the same element ("api") — implementation
			// detail, not architecture.
			{From: "com.acme.backend.api", To: "com.acme.backend.api.internal"},
		},
	}

	result, err := Project(cg, threeTierModel())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Snapshot.Relationships) != 0 {
		t.Errorf("expected intra-element edge to be dropped, got %v", result.Snapshot.Relationships)
	}
}

func TestProject_SubpackageMatchesElementPrefix(t *testing.T) {
	cg := &CodeGraph{
		Nodes: []Node{{ID: "com.acme.backend.api.internal.handlers", Kind: "package"}},
		Edges: []Edge{},
	}

	result, err := Project(cg, threeTierModel())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := result.Snapshot.Elements["api"]; !ok {
		t.Errorf("expected subpackage to resolve to element 'api', got elements: %v", result.Snapshot.Elements)
	}
}

// TestProject_PrefixDoesNotMatchSiblingPackage guards the dot-boundary rule:
// "com.acme.backend.api" must not match "com.acme.backend.apiextra".
func TestProject_PrefixDoesNotMatchSiblingPackage(t *testing.T) {
	cg := &CodeGraph{
		Nodes: []Node{{ID: "com.acme.backend.apiextra", Kind: "package"}},
		Edges: []Edge{},
	}

	result, err := Project(cg, threeTierModel())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Snapshot.Elements) != 0 {
		t.Errorf("expected no element match, got %v", result.Snapshot.Elements)
	}
	if len(result.UnmappedPackages) != 1 || result.UnmappedPackages[0] != "com.acme.backend.apiextra" {
		t.Errorf("expected the sibling package to be unmapped, got %v", result.UnmappedPackages)
	}
}

func TestProject_UnmappedPackagesCollectedNotDropped(t *testing.T) {
	cg := &CodeGraph{
		Nodes: []Node{
			{ID: "com.acme.backend.api", Kind: "package"},
			{ID: "com.acme.legacy.util", Kind: "package"},
		},
		Edges: []Edge{
			{From: "com.acme.backend.api", To: "com.acme.legacy.util"},
		},
	}

	result, err := Project(cg, threeTierModel())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.UnmappedPackages) != 1 || result.UnmappedPackages[0] != "com.acme.legacy.util" {
		t.Errorf("expected com.acme.legacy.util unmapped, got %v", result.UnmappedPackages)
	}
	// The edge touching an unmapped package must not appear as a relationship.
	if len(result.Snapshot.Relationships) != 0 {
		t.Errorf("expected no relationship for an edge with an unmapped endpoint, got %v", result.Snapshot.Relationships)
	}
}

func TestProject_DuplicateEdgesDeduplicated(t *testing.T) {
	cg := &CodeGraph{
		Nodes: []Node{
			{ID: "com.acme.backend.api", Kind: "package"},
			{ID: "com.acme.backend.svc", Kind: "package"},
		},
		Edges: []Edge{
			{From: "com.acme.backend.api", To: "com.acme.backend.svc"},
			{From: "com.acme.backend.api", To: "com.acme.backend.svc"}, // duplicate class-level edge
		},
	}

	result, err := Project(cg, threeTierModel())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Snapshot.Relationships) != 1 {
		t.Errorf("expected duplicate edges to collapse into 1 relationship, got %d: %v",
			len(result.Snapshot.Relationships), result.Snapshot.Relationships)
	}
}

func TestHasAnyCodeMapping(t *testing.T) {
	if has, err := HasAnyCodeMapping(threeTierModel()); err != nil || !has {
		t.Errorf("expected true, got has=%v err=%v", has, err)
	}

	empty := &model.BausteinsichtModel{Model: map[string]model.Element{
		"x": {Kind: "component", Title: "X"},
	}}
	if has, err := HasAnyCodeMapping(empty); err != nil || has {
		t.Errorf("expected false, got has=%v err=%v", has, err)
	}
}
