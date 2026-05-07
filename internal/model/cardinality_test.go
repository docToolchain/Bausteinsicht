package model

import (
	"testing"
)

func TestValidateCardinality(t *testing.T) {
	tests := []struct {
		name          string
		cardinality   string
		shouldBeValid bool
	}{
		{"OneToOne", "1:1", true},
		{"OneToMany", "1:N", true},
		{"ManyToMany", "N:N", true},
		{"Empty", "", true}, // cardinality is optional
		{"Invalid", "1:0", false},
		{"Invalid", "M:N", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &BausteinsichtModel{
				Specification: Specification{
					Elements: map[string]ElementKind{
						"system": {Notation: "box"},
					},
				},
				Model: map[string]Element{
					"a": {Kind: "system", Title: "A"},
					"b": {Kind: "system", Title: "B"},
				},
				Relationships: []Relationship{
					{From: "a", To: "b", Cardinality: tt.cardinality},
				},
			}

			errs := Validate(m)
			hasError := len(errs) > 0

			if tt.shouldBeValid && hasError {
				t.Errorf("expected valid cardinality %q, got error: %v", tt.cardinality, errs)
			}
			if !tt.shouldBeValid && !hasError {
				t.Errorf("expected invalid cardinality %q to have error", tt.cardinality)
			}
		})
	}
}

func TestValidateDataFlow(t *testing.T) {
	tests := []struct {
		name          string
		dataFlow      string
		shouldBeValid bool
	}{
		{"Sync", "sync", true},
		{"Async", "async", true},
		{"RequestResponse", "request/response", true},
		{"PublishSubscribe", "publish/subscribe", true},
		{"Empty", "", true}, // dataFlow is optional
		{"Invalid", "callback", false},
		{"Invalid", "stream", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &BausteinsichtModel{
				Specification: Specification{
					Elements: map[string]ElementKind{
						"system": {Notation: "box"},
					},
				},
				Model: map[string]Element{
					"a": {Kind: "system", Title: "A"},
					"b": {Kind: "system", Title: "B"},
				},
				Relationships: []Relationship{
					{From: "a", To: "b", DataFlow: tt.dataFlow},
				},
			}

			errs := Validate(m)
			hasError := len(errs) > 0

			if tt.shouldBeValid && hasError {
				t.Errorf("expected valid dataFlow %q, got error: %v", tt.dataFlow, errs)
			}
			if !tt.shouldBeValid && !hasError {
				t.Errorf("expected invalid dataFlow %q to have error", tt.dataFlow)
			}
		})
	}
}

func TestCardinalityAnnotations(t *testing.T) {
	m := &BausteinsichtModel{
		Specification: Specification{
			Elements: map[string]ElementKind{
				"service": {Notation: "component"},
			},
		},
		Model: map[string]Element{
			"api":      {Kind: "service", Title: "API"},
			"database": {Kind: "service", Title: "Database"},
		},
		Relationships: []Relationship{
			{From: "api", To: "database", Cardinality: "1:N", DataFlow: "sync", Label: "queries"},
		},
	}

	errs := Validate(m)
	if len(errs) > 0 {
		t.Fatalf("expected valid model, got errors: %v", errs)
	}

	rel := m.Relationships[0]
	if rel.Cardinality != "1:N" {
		t.Errorf("expected cardinality 1:N, got %q", rel.Cardinality)
	}
	if rel.DataFlow != "sync" {
		t.Errorf("expected dataFlow sync, got %q", rel.DataFlow)
	}
}
