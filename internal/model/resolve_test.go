package model

import (
	"testing"
)

func buildTestModel() *BausteinsichtModel {
	return &BausteinsichtModel{
		Model: map[string]Element{
			"customer": {
				Kind:  "actor",
				Title: "Customer",
			},
			"onlineshop": {
				Kind:  "system",
				Title: "Online Shop",
				Children: map[string]Element{
					"frontend": {
						Kind:  "container",
						Title: "Frontend",
					},
					"api": {
						Kind:  "container",
						Title: "API",
						Children: map[string]Element{
							"catalog": {
								Kind:  "component",
								Title: "Catalog",
							},
							"orders": {
								Kind:  "component",
								Title: "Orders",
							},
						},
					},
					"db": {
						Kind:  "container",
						Title: "Database",
					},
				},
			},
		},
	}
}

func TestResolve_SimpleID(t *testing.T) {
	m := buildTestModel()
	elem, err := Resolve(m, "customer")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if elem.Kind != "actor" {
		t.Errorf("expected kind 'actor', got %q", elem.Kind)
	}
}

func TestResolve_NestedID(t *testing.T) {
	m := buildTestModel()
	tests := []struct {
		id        string
		wantKind  string
		wantTitle string
	}{
		{"onlineshop", "system", "Online Shop"},
		{"onlineshop.api", "container", "API"},
		{"onlineshop.api.catalog", "component", "Catalog"},
		{"onlineshop.api.orders", "component", "Orders"},
		{"onlineshop.db", "container", "Database"},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			elem, err := Resolve(m, tt.id)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if elem.Kind != tt.wantKind {
				t.Errorf("expected kind %q, got %q", tt.wantKind, elem.Kind)
			}
			if elem.Title != tt.wantTitle {
				t.Errorf("expected title %q, got %q", tt.wantTitle, elem.Title)
			}
		})
	}
}

func TestResolve_NonExistent(t *testing.T) {
	m := buildTestModel()
	tests := []string{
		"nonexistent",
		"onlineshop.nonexistent",
		"onlineshop.api.nonexistent",
	}
	for _, id := range tests {
		t.Run(id, func(t *testing.T) {
			_, err := Resolve(m, id)
			if err == nil {
				t.Errorf("expected error for id %q, got nil", id)
			}
		})
	}
}

func TestFlattenElements(t *testing.T) {
	m := buildTestModel()
	flat := FlattenElements(m)

	expected := []string{
		"customer",
		"onlineshop",
		"onlineshop.frontend",
		"onlineshop.api",
		"onlineshop.api.catalog",
		"onlineshop.api.orders",
		"onlineshop.db",
	}

	if len(flat) != len(expected) {
		t.Errorf("expected %d elements, got %d", len(expected), len(flat))
	}

	for _, id := range expected {
		if _, ok := flat[id]; !ok {
			t.Errorf("expected element %q not found in flat map", id)
		}
	}
}

func TestMatchPattern_Wildcard(t *testing.T) {
	m := buildTestModel()
	flat := FlattenElements(m)

	tests := []struct {
		pattern  string
		wantIDs  []string
		notWant  []string
	}{
		{
			pattern: "onlineshop.*",
			wantIDs: []string{"onlineshop.frontend", "onlineshop.api", "onlineshop.db"},
			notWant: []string{"onlineshop.api.catalog", "onlineshop.api.orders", "customer"},
		},
		{
			pattern: "onlineshop.api.*",
			wantIDs: []string{"onlineshop.api.catalog", "onlineshop.api.orders"},
			notWant: []string{"onlineshop.api", "onlineshop.frontend"},
		},
		{
			pattern: "customer",
			wantIDs: []string{"customer"},
			notWant: []string{"onlineshop"},
		},
		{
			pattern: "*",
			wantIDs: []string{"customer", "onlineshop"},
			notWant: []string{"onlineshop.frontend", "onlineshop.api"},
		},
		{
			pattern: "**",
			wantIDs: []string{"customer", "onlineshop", "onlineshop.frontend",
				"onlineshop.api", "onlineshop.api.catalog", "onlineshop.api.orders",
				"onlineshop.db"},
		},
		{
			pattern: "onlineshop.**",
			wantIDs: []string{"onlineshop.frontend", "onlineshop.api",
				"onlineshop.api.catalog", "onlineshop.api.orders", "onlineshop.db"},
			notWant: []string{"customer", "onlineshop"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			matches := MatchPattern(flat, tt.pattern)
			matchSet := make(map[string]bool)
			for _, m := range matches {
				matchSet[m] = true
			}
			for _, want := range tt.wantIDs {
				if !matchSet[want] {
					t.Errorf("pattern %q: expected match %q not found", tt.pattern, want)
				}
			}
			for _, notWant := range tt.notWant {
				if matchSet[notWant] {
					t.Errorf("pattern %q: unexpected match %q found", tt.pattern, notWant)
				}
			}
		})
	}
}

func TestResolveView(t *testing.T) {
	m := buildTestModel()

	tests := []struct {
		name    string
		view    View
		wantIDs []string
		notWant []string
	}{
		{
			name: "include all onlineshop children",
			view: View{
				Title:   "Shop View",
				Include: []string{"onlineshop.*"},
			},
			wantIDs: []string{"onlineshop.frontend", "onlineshop.api", "onlineshop.db"},
			notWant: []string{"onlineshop.api.catalog"},
		},
		{
			name: "include with exclude",
			view: View{
				Title:   "Shop No DB",
				Include: []string{"onlineshop.*"},
				Exclude: []string{"onlineshop.db"},
			},
			wantIDs: []string{"onlineshop.frontend", "onlineshop.api"},
			notWant: []string{"onlineshop.db", "onlineshop.api.catalog"},
		},
		{
			name: "empty include returns empty",
			view: View{
				Title:   "Empty View",
				Include: []string{},
			},
			wantIDs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ids, err := ResolveView(m, &tt.view)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			idSet := make(map[string]bool)
			for _, id := range ids {
				idSet[id] = true
			}
			for _, want := range tt.wantIDs {
				if !idSet[want] {
					t.Errorf("expected ID %q not found in result", want)
				}
			}
			for _, notWant := range tt.notWant {
				if idSet[notWant] {
					t.Errorf("unexpected ID %q found in result", notWant)
				}
			}
		})
	}
}
