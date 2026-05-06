package changelog

import (
	"testing"
	"time"

	"github.com/docToolchain/Bausteinsicht/internal/diff"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

func TestGenerate_NoChanges(t *testing.T) {
	elem := model.Element{
		Kind:  "system",
		Title: "System A",
	}
	m1 := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"system-a": elem,
		},
		Relationships: []model.Relationship{},
	}
	m2 := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"system-a": elem,
		},
		Relationships: []model.Relationship{},
	}

	from := Reference{Ref: "v1.0", Date: time.Now()}
	to := Reference{Ref: "v2.0", Date: time.Now().Add(24 * time.Hour)}

	cl := Generate(m1, m2, from, to)

	if cl.Elements.CountAdded() != 0 || cl.Elements.CountRemoved() != 0 || cl.Elements.CountChanged() != 0 {
		t.Error("Expected no changes, but got some")
	}
}

func TestGenerate_AddedElement(t *testing.T) {
	m1 := &model.BausteinsichtModel{
		Model:         map[string]model.Element{},
		Relationships: []model.Relationship{},
	}
	m2 := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"api": {
				Kind:  "container",
				Title: "API Server",
			},
		},
		Relationships: []model.Relationship{},
	}

	from := Reference{Ref: "v1.0"}
	to := Reference{Ref: "v2.0"}

	cl := Generate(m1, m2, from, to)

	if cl.Elements.CountAdded() != 1 {
		t.Errorf("Expected 1 added element, got %d", cl.Elements.CountAdded())
	}
	if len(cl.Elements.Added) > 0 && cl.Elements.Added[0].ID != "api" {
		t.Errorf("Expected added element 'api', got %q", cl.Elements.Added[0].ID)
	}
}

func TestGenerate_RemovedElement(t *testing.T) {
	m1 := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"legacy": {
				Kind:  "system",
				Title: "Legacy System",
			},
		},
		Relationships: []model.Relationship{},
	}
	m2 := &model.BausteinsichtModel{
		Model:         map[string]model.Element{},
		Relationships: []model.Relationship{},
	}

	from := Reference{Ref: "v1.0"}
	to := Reference{Ref: "v2.0"}

	cl := Generate(m1, m2, from, to)

	if cl.Elements.CountRemoved() != 1 {
		t.Errorf("Expected 1 removed element, got %d", cl.Elements.CountRemoved())
	}
}

func TestGenerate_ChangedElement(t *testing.T) {
	m1 := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"db": {
				Kind:       "storage",
				Title:      "Database",
				Technology: "PostgreSQL",
			},
		},
		Relationships: []model.Relationship{},
	}
	m2 := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"db": {
				Kind:       "storage",
				Title:      "Database",
				Technology: "MongoDB",
			},
		},
		Relationships: []model.Relationship{},
	}

	from := Reference{Ref: "v1.0"}
	to := Reference{Ref: "v2.0"}

	cl := Generate(m1, m2, from, to)

	if cl.Elements.CountChanged() != 1 {
		t.Errorf("Expected 1 changed element, got %d", cl.Elements.CountChanged())
	}
}

func TestFlattenElements_Simple(t *testing.T) {
	elems := map[string]model.Element{
		"system": {
			Kind:  "system",
			Title: "My System",
		},
	}

	flattened := flattenElements(elems, "")

	if len(flattened) != 1 {
		t.Errorf("Expected 1 element, got %d", len(flattened))
	}
	if _, ok := flattened["system"]; !ok {
		t.Error("Expected element 'system' not found")
	}
}

func TestFlattenElements_Nested(t *testing.T) {
	elems := map[string]model.Element{
		"system": {
			Kind:  "system",
			Title: "My System",
			Children: map[string]model.Element{
				"api": {
					Kind:  "container",
					Title: "API",
				},
			},
		},
	}

	flattened := flattenElements(elems, "")

	if len(flattened) != 2 {
		t.Errorf("Expected 2 elements, got %d", len(flattened))
	}
	if _, ok := flattened["system"]; !ok {
		t.Error("Expected element 'system' not found")
	}
	if _, ok := flattened["system.api"]; !ok {
		t.Error("Expected element 'system.api' not found")
	}
}

func TestFlattenElements_DeeplyNested(t *testing.T) {
	elems := map[string]model.Element{
		"system": {
			Kind: "system",
			Children: map[string]model.Element{
				"backend": {
					Kind: "subsystem",
					Children: map[string]model.Element{
						"api": {
							Kind:  "container",
							Title: "API",
						},
					},
				},
			},
		},
	}

	flattened := flattenElements(elems, "")

	if len(flattened) != 3 {
		t.Errorf("Expected 3 elements, got %d", len(flattened))
	}
	if _, ok := flattened["system.backend.api"]; !ok {
		t.Error("Expected element 'system.backend.api' not found")
	}
}

func TestElementChanges_FilterByKind(t *testing.T) {
	ec := ElementChanges{
		Added: []diff.ElementChange{
			{
				ID: "api",
				ToBe: &model.Element{Kind: "container", Title: "API"},
			},
			{
				ID: "db",
				ToBe: &model.Element{Kind: "storage", Title: "Database"},
			},
		},
		Removed: []diff.ElementChange{},
		Changed: []diff.ElementChange{},
	}

	filtered := ec.FilterByKind("container")
	if filtered.CountAdded() != 1 {
		t.Errorf("Expected 1 container, got %d", filtered.CountAdded())
	}
}

func TestRelationshipChanges_Counts(t *testing.T) {
	rc := RelationshipChanges{
		Added: []diff.RelationshipChange{
			{From: "a", To: "b", Type: diff.ChangeAdded},
			{From: "c", To: "d", Type: diff.ChangeAdded},
		},
		Removed: []diff.RelationshipChange{
			{From: "e", To: "f", Type: diff.ChangeRemoved},
		},
	}

	if rc.CountAddedRelationships() != 2 {
		t.Errorf("Expected 2 added relationships, got %d", rc.CountAddedRelationships())
	}
	if rc.CountRemovedRelationships() != 1 {
		t.Errorf("Expected 1 removed relationship, got %d", rc.CountRemovedRelationships())
	}
}
