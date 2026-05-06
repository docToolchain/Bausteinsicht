package model

import (
	"testing"
)

func TestValidateLifecycleStatus_ArchivedWithRelationships_Warning(t *testing.T) {
	m := &BausteinsichtModel{
		Specification: Specification{
			Elements: map[string]ElementKind{
				"system": {Notation: "System"},
			},
		},
		Model: map[string]Element{
			"old": {Kind: "system", Title: "Old System", Status: StatusArchived},
			"new": {Kind: "system", Title: "New System", Status: StatusDeployed},
		},
		Relationships: []Relationship{
			{From: "old", To: "new"},
		},
	}

	result := ValidateWithWarnings(m)
	if len(result.Warnings) == 0 {
		t.Fatal("expected warning for archived element with outgoing relationships")
	}

	found := false
	for _, w := range result.Warnings {
		if w.Path == "model.old" && containsStr(w.Message, "outgoing relationships") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning about outgoing relationships, got: %v", result.Warnings)
	}
}

func TestValidateLifecycleStatus_DeprecatedWithoutSuccessor_Warning(t *testing.T) {
	m := &BausteinsichtModel{
		Specification: Specification{
			Elements: map[string]ElementKind{
				"service": {Notation: "Service"},
			},
		},
		Model: map[string]Element{
			"payment-v1": {Kind: "service", Title: "Payment v1", Status: StatusDeprecated},
			"other":      {Kind: "service", Title: "Other Service", Status: StatusDeployed},
		},
		Relationships: []Relationship{},
	}

	result := ValidateWithWarnings(m)
	found := false
	for _, w := range result.Warnings {
		if w.Path == "model.payment-v1" && containsStr(w.Message, "deployed successor") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning for deprecated without successor, got: %v", result.Warnings)
	}
}

func TestValidateLifecycleStatus_DeprecatedWithSuccessor_NoWarning(t *testing.T) {
	m := &BausteinsichtModel{
		Specification: Specification{
			Elements: map[string]ElementKind{
				"service": {Notation: "Service"},
			},
		},
		Model: map[string]Element{
			"payment-v1": {Kind: "service", Title: "Payment v1", Status: StatusDeprecated},
			"payment-v2": {Kind: "service", Title: "Payment v2", Status: StatusDeployed},
		},
		Relationships: []Relationship{
			{From: "payment-v1", To: "payment-v2"},
		},
	}

	result := ValidateWithWarnings(m)
	for _, w := range result.Warnings {
		if w.Path == "model.payment-v1" && containsStr(w.Message, "deployed successor") {
			t.Errorf("unexpected warning for deprecated with successor: %v", w)
		}
	}
}

func TestValidateLifecycleStatus_InvalidStatus_Warning(t *testing.T) {
	m := &BausteinsichtModel{
		Specification: Specification{
			Elements: map[string]ElementKind{
				"service": {Notation: "Service"},
			},
		},
		Model: map[string]Element{
			"api": {Kind: "service", Title: "API", Status: "invalid-status"},
		},
		Relationships: []Relationship{},
	}

	result := ValidateWithWarnings(m)
	found := false
	for _, w := range result.Warnings {
		if w.Path == "model.api" && containsStr(w.Message, "unknown status") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning for unknown status, got: %v", result.Warnings)
	}
}

func TestValidateLifecycleStatus_NoStatus_NoWarning(t *testing.T) {
	m := &BausteinsichtModel{
		Specification: Specification{
			Elements: map[string]ElementKind{
				"service": {Notation: "Service"},
			},
		},
		Model: map[string]Element{
			"api": {Kind: "service", Title: "API"},
		},
		Relationships: []Relationship{},
	}

	result := ValidateWithWarnings(m)
	for _, w := range result.Warnings {
		if w.Path == "model.api" {
			t.Errorf("unexpected warning for element with no status: %v", w)
		}
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
