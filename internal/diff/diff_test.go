package diff

import (
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// Alias for testing to avoid type mismatch
type ModelSnapshot = model.ModelSnapshot

func TestCompare_NoSnapshots(t *testing.T) {
	result := Compare(nil, nil)
	if result == nil {
		t.Fatal("Compare returned nil")
	}
	if len(result.Elements) != 0 || len(result.Relationships) != 0 {
		t.Error("Expected empty diff for nil snapshots")
	}
}

func TestCompare_AddedElement(t *testing.T) {
	asIs := &ModelSnapshot{
		Elements:      make(map[string]model.Element),
		Relationships: []model.Relationship{},
	}

	toBe := &ModelSnapshot{
		Elements: map[string]model.Element{
			"system": {
				Title: "System",
				Kind:  "system",
			},
		},
		Relationships: []model.Relationship{},
	}

	result := Compare(asIs, toBe)

	if len(result.Elements) != 1 {
		t.Fatalf("Expected 1 element change, got %d", len(result.Elements))
	}

	change := result.Elements[0]
	if change.Type != ChangeAdded {
		t.Errorf("Expected ChangeAdded, got %s", change.Type)
	}
	if change.ID != "system" {
		t.Errorf("Expected id 'system', got %s", change.ID)
	}
	if result.Summary.AddedElements != 1 {
		t.Errorf("Expected 1 added element in summary, got %d", result.Summary.AddedElements)
	}
}

func TestCompare_RemovedElement(t *testing.T) {
	asIs := &ModelSnapshot{
		Elements: map[string]model.Element{
			"legacy": {
				Title: "Legacy System",
				Kind:  "system",
			},
		},
		Relationships: []model.Relationship{},
	}

	toBe := &ModelSnapshot{
		Elements:      make(map[string]model.Element),
		Relationships: []model.Relationship{},
	}

	result := Compare(asIs, toBe)

	if len(result.Elements) != 1 {
		t.Fatalf("Expected 1 element change, got %d", len(result.Elements))
	}

	change := result.Elements[0]
	if change.Type != ChangeRemoved {
		t.Errorf("Expected ChangeRemoved, got %s", change.Type)
	}
	if result.Summary.RemovedElements != 1 {
		t.Errorf("Expected 1 removed element in summary, got %d", result.Summary.RemovedElements)
	}
}

func TestCompare_ChangedElement(t *testing.T) {
	asIs := &ModelSnapshot{
		Elements: map[string]model.Element{
			"api": {
				Title:      "REST API",
				Kind:       "container",
				Technology: "Go",
			},
		},
		Relationships: []model.Relationship{},
	}

	toBe := &ModelSnapshot{
		Elements: map[string]model.Element{
			"api": {
				Title:      "REST API v2",
				Kind:       "container",
				Technology: "Rust",
			},
		},
		Relationships: []model.Relationship{},
	}

	result := Compare(asIs, toBe)

	if len(result.Elements) != 1 {
		t.Fatalf("Expected 1 element change, got %d", len(result.Elements))
	}

	change := result.Elements[0]
	if change.Type != ChangeChanged {
		t.Errorf("Expected ChangeChanged, got %s", change.Type)
	}
	if result.Summary.ChangedElements != 1 {
		t.Errorf("Expected 1 changed element in summary, got %d", result.Summary.ChangedElements)
	}
}

func TestCompare_AddedRelationship(t *testing.T) {
	asIs := &ModelSnapshot{
		Elements: map[string]model.Element{
			"client": {Title: "Client"},
			"api":    {Title: "API"},
		},
		Relationships: []model.Relationship{},
	}

	toBe := &ModelSnapshot{
		Elements: map[string]model.Element{
			"client": {Title: "Client"},
			"api":    {Title: "API"},
		},
		Relationships: []model.Relationship{
			{From: "client", To: "api", Label: "calls"},
		},
	}

	result := Compare(asIs, toBe)

	if len(result.Relationships) != 1 {
		t.Fatalf("Expected 1 relationship change, got %d", len(result.Relationships))
	}

	change := result.Relationships[0]
	if change.Type != ChangeAdded {
		t.Errorf("Expected ChangeAdded, got %s", change.Type)
	}
	if result.Summary.AddedRelationships != 1 {
		t.Errorf("Expected 1 added relationship in summary, got %d", result.Summary.AddedRelationships)
	}
}

func TestCompare_RemovedRelationship(t *testing.T) {
	asIs := &ModelSnapshot{
		Elements: map[string]model.Element{
			"client": {Title: "Client"},
			"api":    {Title: "API"},
		},
		Relationships: []model.Relationship{
			{From: "client", To: "api", Label: "calls"},
		},
	}

	toBe := &ModelSnapshot{
		Elements: map[string]model.Element{
			"client": {Title: "Client"},
			"api":    {Title: "API"},
		},
		Relationships: []model.Relationship{},
	}

	result := Compare(asIs, toBe)

	if len(result.Relationships) != 1 {
		t.Fatalf("Expected 1 relationship change, got %d", len(result.Relationships))
	}

	change := result.Relationships[0]
	if change.Type != ChangeRemoved {
		t.Errorf("Expected ChangeRemoved, got %s", change.Type)
	}
	if result.Summary.RemovedRelationships != 1 {
		t.Errorf("Expected 1 removed relationship in summary, got %d", result.Summary.RemovedRelationships)
	}
}
