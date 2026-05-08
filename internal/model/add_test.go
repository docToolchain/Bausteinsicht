package model

import (
	"testing"
)

func TestAddView_Success(t *testing.T) {
	m := &BausteinsichtModel{
		Model: map[string]Element{
			"system": {Kind: "system", Title: "My System"},
			"user":   {Kind: "actor", Title: "User"},
		},
		Views: make(map[string]View),
	}

	err := m.AddView("context", View{
		Title:   "Context View",
		Scope:   "system",
		Include: []string{"system", "user"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	view, exists := m.Views["context"]
	if !exists {
		t.Fatal("view not found in model")
	}
	if view.Title != "Context View" {
		t.Errorf("expected title 'Context View', got %q", view.Title)
	}
	if view.Scope != "system" {
		t.Errorf("expected scope 'system', got %q", view.Scope)
	}
}

func TestAddView_EmptyKey(t *testing.T) {
	m := &BausteinsichtModel{}
	err := m.AddView("", View{Title: "Empty Key"})

	if err == nil {
		t.Error("expected error for empty key")
	}
}

func TestAddView_ScopeNotFound(t *testing.T) {
	m := &BausteinsichtModel{
		Model: make(map[string]Element),
		Views: make(map[string]View),
	}

	err := m.AddView("test", View{
		Scope: "nonexistent",
	})

	if err == nil {
		t.Error("expected error for nonexistent scope")
	}
}

func TestAddView_IncludeNotFound(t *testing.T) {
	m := &BausteinsichtModel{
		Model: map[string]Element{
			"system": {Kind: "system", Title: "System"},
		},
		Views: make(map[string]View),
	}

	err := m.AddView("test", View{
		Include: []string{"system", "nonexistent"},
	})

	if err == nil {
		t.Error("expected error for nonexistent include")
	}
}

func TestAddView_WithWildcards(t *testing.T) {
	m := &BausteinsichtModel{
		Model: map[string]Element{
			"system": {Kind: "system", Title: "System"},
		},
		Views: make(map[string]View),
	}

	err := m.AddView("test", View{
		Include: []string{"system.*", "another.*"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddView_MergeTitle(t *testing.T) {
	m := &BausteinsichtModel{
		Model: map[string]Element{
			"system": {Kind: "system", Title: "System"},
		},
		Views: map[string]View{
			"context": {
				Title:   "Old Title",
				Include: []string{"system"},
			},
		},
	}

	err := m.AddView("context", View{
		Title: "New Title",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	view := m.Views["context"]
	if view.Title != "New Title" {
		t.Errorf("expected title 'New Title', got %q", view.Title)
	}
	if len(view.Include) != 1 || view.Include[0] != "system" {
		t.Errorf("expected include [system], got %v", view.Include)
	}
}

func TestAddView_MergeInclude(t *testing.T) {
	m := &BausteinsichtModel{
		Model: map[string]Element{
			"system": {Kind: "system", Title: "System"},
			"user":   {Kind: "actor", Title: "User"},
		},
		Views: map[string]View{
			"context": {
				Title:   "Context",
				Include: []string{"system"},
			},
		},
	}

	err := m.AddView("context", View{
		Include: []string{"user"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	view := m.Views["context"]
	if len(view.Include) != 2 {
		t.Errorf("expected 2 includes, got %d: %v", len(view.Include), view.Include)
	}
}

func TestAddView_MergeIncludeDeduplicate(t *testing.T) {
	m := &BausteinsichtModel{
		Model: map[string]Element{
			"system": {Kind: "system", Title: "System"},
		},
		Views: map[string]View{
			"context": {
				Title:   "Context",
				Include: []string{"system"},
			},
		},
	}

	// Add same element again
	err := m.AddView("context", View{
		Include: []string{"system"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	view := m.Views["context"]
	if len(view.Include) != 1 {
		t.Errorf("expected 1 include (deduplicated), got %d: %v", len(view.Include), view.Include)
	}
}

func TestAddSpecificationElement_Success(t *testing.T) {
	m := &BausteinsichtModel{
		Specification: Specification{
			Elements: make(map[string]ElementKind),
		},
	}

	err := m.AddSpecificationElement("custom", ElementKind{
		Notation:    "Custom Box",
		Description: "A custom element type",
		Container:   true,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	elem, exists := m.Specification.Elements["custom"]
	if !exists {
		t.Fatal("element not found in specification")
	}
	if elem.Notation != "Custom Box" {
		t.Errorf("expected notation 'Custom Box', got %q", elem.Notation)
	}
	if !elem.Container {
		t.Error("expected Container=true")
	}
}

func TestAddSpecificationElement_Duplicate(t *testing.T) {
	m := &BausteinsichtModel{
		Specification: Specification{
			Elements: map[string]ElementKind{
				"system": {Notation: "System"},
			},
		},
	}

	err := m.AddSpecificationElement("system", ElementKind{
		Notation: "Another System",
	})

	if err == nil {
		t.Error("expected error for duplicate element")
	}
}

func TestAddSpecificationElement_EmptyNotation(t *testing.T) {
	m := &BausteinsichtModel{
		Specification: Specification{
			Elements: make(map[string]ElementKind),
		},
	}

	err := m.AddSpecificationElement("custom", ElementKind{
		Notation: "",
	})

	if err == nil {
		t.Error("expected error for empty notation")
	}
}

func TestAddSpecificationRelationship_Success(t *testing.T) {
	m := &BausteinsichtModel{
		Specification: Specification{
			Relationships: make(map[string]RelationshipKind),
		},
	}

	err := m.AddSpecificationRelationship("depends_on", RelationshipKind{
		Notation: "depends on",
		Dashed:   true,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rel, exists := m.Specification.Relationships["depends_on"]
	if !exists {
		t.Fatal("relationship not found in specification")
	}
	if rel.Notation != "depends on" {
		t.Errorf("expected notation 'depends on', got %q", rel.Notation)
	}
	if !rel.Dashed {
		t.Error("expected Dashed=true")
	}
}

func TestAddSpecificationRelationship_Duplicate(t *testing.T) {
	m := &BausteinsichtModel{
		Specification: Specification{
			Relationships: map[string]RelationshipKind{
				"uses": {Notation: "uses"},
			},
		},
	}

	err := m.AddSpecificationRelationship("uses", RelationshipKind{
		Notation: "another uses",
	})

	if err == nil {
		t.Error("expected error for duplicate relationship")
	}
}
