package model

import (
	"testing"
)

// buildValidModel returns a fully valid model for use as a baseline in tests.
func buildValidModel() *BausteinsichtModel {
	return &BausteinsichtModel{
		Specification: Specification{
			Elements: map[string]ElementKind{
				"actor":     {Notation: "Actor", Container: false},
				"system":    {Notation: "System", Container: true},
				"container": {Notation: "Container", Container: false},
			},
			Relationships: map[string]RelationshipKind{
				"uses": {Notation: "uses"},
			},
		},
		Model: map[string]Element{
			"customer": {
				Kind:  "actor",
				Title: "Customer",
			},
			"shop": {
				Kind:  "system",
				Title: "Online Shop",
				Children: map[string]Element{
					"api": {
						Kind:  "container",
						Title: "API",
					},
				},
			},
		},
		Relationships: []Relationship{
			{From: "customer", To: "shop", Kind: "uses"},
		},
		Views: map[string]View{
			"main": {Title: "Main View", Scope: "shop"},
		},
	}
}

func TestValidate_ValidModel(t *testing.T) {
	m := buildValidModel()
	errs := Validate(m)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestValidate_MissingElementKind(t *testing.T) {
	m := buildValidModel()
	elem := m.Model["customer"]
	elem.Kind = ""
	m.Model["customer"] = elem

	errs := Validate(m)
	if !containsPath(errs, "model.customer") {
		t.Errorf("expected error for model.customer, got %v", errs)
	}
}

func TestValidate_MissingElementTitle(t *testing.T) {
	m := buildValidModel()
	elem := m.Model["customer"]
	elem.Title = ""
	m.Model["customer"] = elem

	errs := Validate(m)
	if !containsPath(errs, "model.customer") {
		t.Errorf("expected error for model.customer, got %v", errs)
	}
}

func TestValidate_UnknownElementKind(t *testing.T) {
	m := buildValidModel()
	elem := m.Model["customer"]
	elem.Kind = "unknownkind"
	m.Model["customer"] = elem

	errs := Validate(m)
	if !containsPath(errs, "model.customer") {
		t.Errorf("expected error for model.customer, got %v", errs)
	}
}

func TestValidate_ChildrenOnNonContainerKind(t *testing.T) {
	m := buildValidModel()
	// "actor" has container:false, so children are not allowed
	elem := m.Model["customer"]
	elem.Children = map[string]Element{
		"child": {Kind: "actor", Title: "Child"},
	}
	m.Model["customer"] = elem

	errs := Validate(m)
	if !containsPath(errs, "model.customer") {
		t.Errorf("expected error for model.customer, got %v", errs)
	}
}

func TestValidate_RelationshipFromNonExistent(t *testing.T) {
	m := buildValidModel()
	m.Relationships = []Relationship{
		{From: "nonexistent", To: "shop", Kind: "uses"},
	}

	errs := Validate(m)
	if !containsMessage(errs, "nonexistent") {
		t.Errorf("expected error mentioning 'nonexistent', got %v", errs)
	}
}

func TestValidate_RelationshipToNonExistent(t *testing.T) {
	m := buildValidModel()
	m.Relationships = []Relationship{
		{From: "customer", To: "nonexistent", Kind: "uses"},
	}

	errs := Validate(m)
	if !containsMessage(errs, "nonexistent") {
		t.Errorf("expected error mentioning 'nonexistent', got %v", errs)
	}
}

func TestValidate_UnknownRelationshipKind(t *testing.T) {
	m := buildValidModel()
	m.Relationships = []Relationship{
		{From: "customer", To: "shop", Kind: "unknownrel"},
	}

	errs := Validate(m)
	if !containsMessage(errs, "unknownrel") {
		t.Errorf("expected error mentioning 'unknownrel', got %v", errs)
	}
}

func TestValidate_ViewMissingTitle(t *testing.T) {
	m := buildValidModel()
	m.Views["empty"] = View{Title: ""}

	errs := Validate(m)
	if !containsPath(errs, "views.empty") {
		t.Errorf("expected error for views.empty, got %v", errs)
	}
}

func TestValidate_ViewNonExistentScope(t *testing.T) {
	m := buildValidModel()
	m.Views["bad"] = View{Title: "Bad View", Scope: "nonexistent"}

	errs := Validate(m)
	if !containsPath(errs, "views.bad") {
		t.Errorf("expected error for views.bad, got %v", errs)
	}
}

func TestValidate_DuplicateRelationship(t *testing.T) {
	m := buildValidModel()
	// Add a duplicate relationship (same from/to as the existing one).
	m.Relationships = append(m.Relationships,
		Relationship{From: "customer", To: "shop", Kind: "uses"})

	errs := Validate(m)
	if !containsMessage(errs, "duplicate") {
		t.Errorf("expected error about duplicate relationship, got %v", errs)
	}
}

// containsPath checks whether any error has the given path.
func containsPath(errs []ValidationError, path string) bool {
	for _, e := range errs {
		if e.Path == path {
			return true
		}
	}
	return false
}

// containsMessage checks whether any error message contains the given substring.
func containsMessage(errs []ValidationError, substr string) bool {
	for _, e := range errs {
		if contains(e.Message, substr) || contains(e.Path, substr) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
