package model

import (
	"testing"
)

// Additional tests to improve code coverage for critical paths

func TestStatusColorAllStatuses(t *testing.T) {
	tests := []struct {
		status   string
		expected string
	}{
		{StatusProposed, "#fff2cc"},
		{StatusDesign, "#dae8fc"},
		{StatusImplementing, "#ffe6cc"},
		{StatusDeployed, "#d5e8d4"},
		{StatusDeprecated, "#f8cecc"},
		{StatusArchived, "#f5f5f5"},
		{"unknown", "#ffffff"},
		{"", "#ffffff"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := StatusColor(tt.status)
			if result != tt.expected {
				t.Errorf("StatusColor(%q) = %q, want %q", tt.status, result, tt.expected)
			}
		})
	}
}

func TestValidateConstraints(t *testing.T) {
	m := &BausteinsichtModel{
		Specification: Specification{
			Elements: map[string]ElementKind{
				"system": {Notation: "box"},
				"service": {Notation: "component"},
			},
		},
		Model: map[string]Element{
			"a": {Kind: "system", Title: "A"},
			"b": {Kind: "service", Title: "B"},
		},
		Constraints: []Constraint{
			{
				ID:       "no-direct-db",
				Rule:     "no-relationship",
				FromKind: "system",
				ToKind:   "database",
			},
		},
	}

	errs := Validate(m)
	if len(errs) > 0 {
		t.Errorf("expected valid model with constraints, got %v", errs)
	}
}

func TestDynamicViewStepTypes(t *testing.T) {
	tests := []struct {
		name      string
		stepType  StepType
		shouldErr bool
	}{
		{"sync", StepSync, false},
		{"async", StepAsync, false},
		{"return", StepReturn, false},
		{"empty", "", false},
		{"invalid", "callback", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &BausteinsichtModel{
				Specification: Specification{
					Elements: map[string]ElementKind{
						"service": {Notation: "component"},
					},
				},
				Model: map[string]Element{
					"a": {Kind: "service", Title: "A"},
					"b": {Kind: "service", Title: "B"},
				},
				DynamicViews: []DynamicView{
					{
						Key:   "test",
						Title: "Test",
						Steps: []SequenceStep{
							{Index: 1, From: "a", To: "b", Type: tt.stepType},
						},
					},
				},
			}

			errs := Validate(m)
			hasErr := len(errs) > 0

			if tt.shouldErr && !hasErr {
				t.Errorf("expected error for step type %q", tt.stepType)
			}
			if !tt.shouldErr && hasErr {
				t.Errorf("expected no error for step type %q, got %v", tt.stepType, errs)
			}
		})
	}
}

func TestElementKindConfig(t *testing.T) {
	tests := []struct {
		name      string
		kind      ElementKind
		canHaveChildren bool
	}{
		{"container", ElementKind{Notation: "box", Container: true}, true},
		{"leaf", ElementKind{Notation: "component", Container: false}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &BausteinsichtModel{
				Specification: Specification{
					Elements: map[string]ElementKind{
						"container": {Notation: "box", Container: true},
						"component": {Notation: "component", Container: false},
					},
				},
				Model: map[string]Element{
					"parent": {
						Kind:  "container",
						Title: "Parent",
						Children: map[string]Element{
							"child": {Kind: "component", Title: "Child"},
						},
					},
				},
			}

			errs := Validate(m)
			if len(errs) > 0 {
				t.Errorf("expected valid nested structure, got %v", errs)
			}
		})
	}
}

func TestRelationshipKindConfig(t *testing.T) {
	m := &BausteinsichtModel{
		Specification: Specification{
			Elements: map[string]ElementKind{
				"system": {Notation: "box"},
			},
			Relationships: map[string]RelationshipKind{
				"sync": {Notation: "arrow", Dashed: false},
				"async": {Notation: "arrow", Dashed: true},
			},
		},
		Model: map[string]Element{
			"a": {Kind: "system", Title: "A"},
			"b": {Kind: "system", Title: "B"},
		},
		Relationships: []Relationship{
			{From: "a", To: "b", Kind: "sync"},
			{From: "b", To: "a", Kind: "async"},
		},
	}

	errs := Validate(m)
	if len(errs) > 0 {
		t.Errorf("expected valid relationships with kinds, got %v", errs)
	}
}

func TestComplexHierarchy(t *testing.T) {
	m := &BausteinsichtModel{
		Specification: Specification{
			Elements: map[string]ElementKind{
				"system": {Notation: "box", Container: true},
				"service": {Notation: "component", Container: true},
				"module": {Notation: "box", Container: false},
			},
		},
		Model: map[string]Element{
			"platform": {
				Kind:  "system",
				Title: "Platform",
				Children: map[string]Element{
					"backend": {
						Kind:  "service",
						Title: "Backend",
						Children: map[string]Element{
							"auth": {Kind: "module", Title: "Auth"},
							"api": {Kind: "module", Title: "API"},
						},
					},
					"frontend": {
						Kind:  "service",
						Title: "Frontend",
						Children: map[string]Element{
							"ui": {Kind: "module", Title: "UI"},
						},
					},
				},
			},
		},
		Relationships: []Relationship{
			{From: "platform.backend.auth", To: "platform.backend.api"},
			{From: "platform.frontend.ui", To: "platform.backend.api"},
		},
	}

	errs := Validate(m)
	if len(errs) > 0 {
		t.Errorf("expected valid complex hierarchy, got %v", errs)
	}
}
