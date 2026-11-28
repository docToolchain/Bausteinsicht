package model

import (
	"strings"
	"testing"
)

// #689: validate said "Model is valid." for a constraint with an unknown rule
// and for two constraints with the same id; lint then failed.
func TestValidate_UnknownConstraintRule(t *testing.T) {
	m := buildValidModel()
	m.Constraints = []Constraint{
		{ID: "c1", Rule: "bogus-rule"},
		{ID: "c2", Rule: "no-cycles"},
		{ID: "c3", Rule: "required-field", ElementKind: "actor", Field: "description"},
	}

	errs := Validate(m)

	if len(errs) != 2 {
		t.Fatalf("want 2 errors (c1, c2), got %v", errs)
	}
	if errs[0].Path != "constraints[0].rule" || !strings.Contains(errs[0].Message, `unknown constraint rule "bogus-rule"`) {
		t.Errorf("first error: %+v", errs[0])
	}
	if !strings.Contains(errs[0].Message, "no-relationship") {
		t.Errorf("the message should list the valid rules: %q", errs[0].Message)
	}
	if errs[1].Path != "constraints[1].rule" {
		t.Errorf("second error path %q", errs[1].Path)
	}
}

func TestValidate_EmptyConstraintRule(t *testing.T) {
	m := buildValidModel()
	m.Constraints = []Constraint{{ID: "c1"}}
	errs := Validate(m)
	if len(errs) != 1 || errs[0].Path != "constraints[0].rule" {
		t.Fatalf("want one error for the missing rule, got %v", errs)
	}
}

func TestValidate_DuplicateConstraintID(t *testing.T) {
	m := buildValidModel()
	m.Constraints = []Constraint{
		{ID: "c1", Rule: "max-depth", Max: 3},
		{ID: "c1", Rule: "no-circular-dependency"},
	}
	errs := Validate(m)
	if len(errs) != 1 {
		t.Fatalf("want 1 error, got %v", errs)
	}
	if errs[0].Path != "constraints[1].id" || !strings.Contains(errs[0].Message, `duplicate constraint id "c1"`) {
		t.Errorf("error: %+v", errs[0])
	}
}

func TestValidate_KnownConstraintsAreValid(t *testing.T) {
	m := buildValidModel()
	m.Constraints = []Constraint{
		{ID: "a", Rule: "no-relationship", FromKind: "actor", ToKind: "container"},
		{ID: "b", Rule: "allowed-relationship", FromKind: "actor", ToKind: "system"},
		{ID: "c", Rule: "required-field", ElementKind: "actor", Field: "description"},
		{ID: "d", Rule: "max-depth", Max: 3},
		{ID: "e", Rule: "no-circular-dependency"},
		{ID: "f", Rule: "technology-allowed", ElementKind: "container", Technologies: []string{"Go"}},
	}
	if errs := Validate(m); len(errs) != 0 {
		t.Errorf("valid constraints rejected: %v", errs)
	}
}
