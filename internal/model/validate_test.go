package model

import (
	"strings"
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

// TestValidate_MultipleRelsSamePairDifferentLabel verifies that multiple
// relationships between the same pair with different labels are allowed. (#142)
func TestValidate_MultipleRelsSamePairDifferentLabel(t *testing.T) {
	m := buildValidModel()
	m.Relationships = []Relationship{
		{From: "customer", To: "shop", Kind: "uses", Label: "browses"},
		{From: "customer", To: "shop", Kind: "uses", Label: "buys from"},
	}

	errs := Validate(m)
	for _, e := range errs {
		if contains(e.Message, "duplicate") {
			t.Errorf("should not report duplicate for different labels, got: %v", e)
		}
	}
}

// TestValidate_MultipleRelsSamePairDifferentKind verifies that multiple
// relationships between the same pair with different kinds are allowed. (#142)
func TestValidate_MultipleRelsSamePairDifferentKind(t *testing.T) {
	m := buildValidModel()
	m.Specification.Relationships["calls"] = RelationshipKind{Notation: "calls"}
	m.Relationships = []Relationship{
		{From: "customer", To: "shop", Kind: "uses"},
		{From: "customer", To: "shop", Kind: "calls"},
	}

	errs := Validate(m)
	for _, e := range errs {
		if contains(e.Message, "duplicate") {
			t.Errorf("should not report duplicate for different kinds, got: %v", e)
		}
	}
}

func TestValidate_EmptyElementID(t *testing.T) {
	m := &BausteinsichtModel{
		Specification: Specification{
			Elements: map[string]ElementKind{"system": {Notation: "System"}},
		},
		Model: map[string]Element{
			"": {Kind: "system", Title: "Empty ID"},
		},
	}
	errs := Validate(m)
	if len(errs) == 0 {
		t.Fatal("expected validation error for empty element ID")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "invalid element ID") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'invalid element ID' error, got: %v", errs)
	}
}

func TestValidate_WhitespaceElementID(t *testing.T) {
	m := &BausteinsichtModel{
		Specification: Specification{
			Elements: map[string]ElementKind{"system": {Notation: "System"}},
		},
		Model: map[string]Element{
			" ": {Kind: "system", Title: "Whitespace"},
		},
	}
	errs := Validate(m)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "invalid element ID") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'invalid element ID' error for whitespace-only ID, got: %v", errs)
	}
}

func TestValidate_EmptyChildID(t *testing.T) {
	m := &BausteinsichtModel{
		Specification: Specification{
			Elements: map[string]ElementKind{
				"system": {Notation: "System", Container: true},
			},
		},
		Model: map[string]Element{
			"parent": {Kind: "system", Title: "Parent", Children: map[string]Element{
				"": {Kind: "system", Title: "Empty Child"},
			}},
		},
	}
	errs := Validate(m)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "invalid element ID") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'invalid element ID' error for empty child ID, got: %v", errs)
	}
}

func TestValidate_ViewIncludeNonExistentElement(t *testing.T) {
	m := buildValidModel()
	m.Views["overview"] = View{
		Title:   "Overview",
		Include: []string{"nonexistent"},
	}

	errs := Validate(m)
	if !containsPath(errs, "views.overview.include") {
		t.Errorf("expected error at views.overview.include, got %v", errs)
	}
	if !containsMessage(errs, `element "nonexistent" does not exist`) {
		t.Errorf("expected error message about element not existing, got %v", errs)
	}
}

func TestValidate_ViewExcludeNonExistentElement(t *testing.T) {
	m := buildValidModel()
	m.Views["overview"] = View{
		Title:   "Overview",
		Include: []string{"customer"},
		Exclude: []string{"nonexistent"},
	}

	errs := Validate(m)
	if !containsPath(errs, "views.overview.exclude") {
		t.Errorf("expected error at views.overview.exclude, got %v", errs)
	}
	if !containsMessage(errs, `element "nonexistent" does not exist`) {
		t.Errorf("expected error message about element not existing, got %v", errs)
	}
}

func TestValidate_ViewWildcardPatternsNoError(t *testing.T) {
	m := buildValidModel()
	m.Views["overview"] = View{
		Title:   "Overview",
		Include: []string{"*", "**", "foo.*"},
		Exclude: []string{"bar.**"},
	}

	errs := Validate(m)
	for _, e := range errs {
		if strings.Contains(e.Path, "views.overview.include") || strings.Contains(e.Path, "views.overview.exclude") {
			t.Errorf("wildcard patterns should not produce errors, got %v", e)
		}
	}
}

func TestValidate_ViewIncludeValidElement(t *testing.T) {
	m := buildValidModel()
	m.Views["overview"] = View{
		Title:   "Overview",
		Include: []string{"customer", "shop.api"},
	}

	errs := Validate(m)
	for _, e := range errs {
		if strings.Contains(e.Path, "views.overview.include") {
			t.Errorf("valid element references should not produce errors, got %v", e)
		}
	}
}

func TestValidateWithWarnings_EmptyJSON(t *testing.T) {
	m := &BausteinsichtModel{}
	result := ValidateWithWarnings(m)
	if len(result.Warnings) == 0 {
		t.Fatal("expected warnings for empty model, got none")
	}
	foundSpec := false
	foundModel := false
	for _, w := range result.Warnings {
		if w.Path == "specification" && strings.Contains(w.Message, "no element kinds defined") {
			foundSpec = true
		}
		if w.Path == "model" && strings.Contains(w.Message, "no elements defined") {
			foundModel = true
		}
	}
	if !foundSpec {
		t.Errorf("expected warning about empty specification, got: %v", result.Warnings)
	}
	if !foundModel {
		t.Errorf("expected warning about empty model, got: %v", result.Warnings)
	}
	// Empty model should not produce errors
	if len(result.Errors) != 0 {
		t.Errorf("expected no errors for empty model, got %v", result.Errors)
	}
}

func TestValidateWithWarnings_SpecOnly(t *testing.T) {
	m := &BausteinsichtModel{
		Specification: Specification{
			Elements: map[string]ElementKind{
				"system": {Notation: "System"},
			},
		},
	}
	result := ValidateWithWarnings(m)
	foundModel := false
	for _, w := range result.Warnings {
		if w.Path == "model" && strings.Contains(w.Message, "no elements defined") {
			foundModel = true
		}
	}
	if !foundModel {
		t.Errorf("expected warning about empty model, got: %v", result.Warnings)
	}
	// Should NOT warn about specification since it has elements defined
	for _, w := range result.Warnings {
		if w.Path == "specification" {
			t.Errorf("should not warn about specification when elements are defined, got: %v", w)
		}
	}
}

func TestValidateWithWarnings_ValidModel_NoWarnings(t *testing.T) {
	m := buildValidModel()
	result := ValidateWithWarnings(m)
	if len(result.Warnings) != 0 {
		t.Errorf("expected no warnings for valid model, got %v", result.Warnings)
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected no errors for valid model, got %v", result.Errors)
	}
}

func TestValidate_ViewInvalidLayout(t *testing.T) {
	m := buildValidModel()
	m.Views["overview"] = View{
		Title:   "Overview",
		Include: []string{"*"},
		Layout:  "invalid",
	}
	errs := Validate(m)
	if !containsMessage(errs, "invalid layout") {
		t.Errorf("expected layout validation error, got: %v", errs)
	}
}

func TestValidate_ViewValidLayouts(t *testing.T) {
	for _, layout := range []string{"", "layered", "grid", "none"} {
		t.Run(layout, func(t *testing.T) {
			m := buildValidModel()
			m.Views["overview"] = View{
				Title:   "Overview",
				Include: []string{"*"},
				Layout:  layout,
			}
			errs := Validate(m)
			if containsMessage(errs, "invalid layout") {
				t.Errorf("layout %q should be valid, got error: %v", layout, errs)
			}
		})
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
