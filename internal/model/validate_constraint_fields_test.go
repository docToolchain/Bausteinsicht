package model

import (
	"strings"
	"testing"
)

func constraintErrors(t *testing.T, c Constraint) []ValidationError {
	t.Helper()
	m := buildValidModel()
	m.Constraints = []Constraint{c}
	return Validate(m)
}

// #727: a constraint without the selectors or values its rule needs matched
// nothing (or everything) and lint stayed silent.
func TestValidate_ConstraintNeedsItsFields(t *testing.T) {
	for name, tc := range map[string]struct {
		c    Constraint
		path string
		msg  string
	}{
		"no-relationship without from":    {Constraint{ID: "c", Rule: "no-relationship", ToKind: "container"}, "constraints[0]", "from selector"},
		"no-relationship without to":      {Constraint{ID: "c", Rule: "no-relationship", FromKind: "actor"}, "constraints[0]", "to selector"},
		"allowed-relationship without to": {Constraint{ID: "c", Rule: "allowed-relationship", FromKinds: []string{"actor"}}, "constraints[0]", "to selector"},
		"max-depth without max":           {Constraint{ID: "c", Rule: "max-depth"}, "constraints[0].max", "at least 1"},
		"required-field without field":    {Constraint{ID: "c", Rule: "required-field", ElementKind: "actor"}, "constraints[0].field", "description, technology, title"},
		"required-field bad field":        {Constraint{ID: "c", Rule: "required-field", ElementKind: "actor", Field: "colour"}, "constraints[0].field", `"colour"`},
		"required-field without selector": {Constraint{ID: "c", Rule: "required-field", Field: "title"}, "constraints[0]", "element-kind or tag"},
		"technology-allowed no list":      {Constraint{ID: "c", Rule: "technology-allowed", ElementKind: "actor"}, "constraints[0].technologies", "at least one"},
		"technology-allowed no selector":  {Constraint{ID: "c", Rule: "technology-allowed", Technologies: []string{"Go"}}, "constraints[0]", "element-kind or tag"},
	} {
		t.Run(name, func(t *testing.T) {
			errs := constraintErrors(t, tc.c)
			if len(errs) != 1 || errs[0].Path != tc.path || !strings.Contains(errs[0].Message, tc.msg) {
				t.Errorf("errors: %+v, want one at %s containing %q", errs, tc.path, tc.msg)
			}
		})
	}
}

func TestValidate_ConstraintWithItsFieldsIsValid(t *testing.T) {
	for _, c := range []Constraint{
		{ID: "a", Rule: "no-relationship", FromTag: "x", ToKind: "container"},
		{ID: "b", Rule: "allowed-relationship", FromKinds: []string{"actor"}, To: "shop"},
		{ID: "c", Rule: "required-field", Tag: "x", Field: "description"},
		{ID: "d", Rule: "max-depth", Max: 1},
		{ID: "e", Rule: "technology-allowed", Tag: "x", Technologies: []string{"Go"}},
		{ID: "f", Rule: "no-circular-dependency"},
	} {
		if errs := constraintErrors(t, c); len(errs) != 0 {
			t.Errorf("%s: %v", c.ID, errs)
		}
	}
}

func TestValidate_ConstraintFieldOfAnotherRuleIsAWarning(t *testing.T) {
	m := buildValidModel()
	m.Constraints = []Constraint{{ID: "c", Rule: "required-field", ElementKind: "actor", Field: "title", Max: 3}}
	res := ValidateWithWarnings(m)
	if len(res.Errors) != 0 {
		t.Fatalf("errors: %v", res.Errors)
	}
	found := false
	for _, w := range res.Warnings {
		if w.Path == "constraints[0].max" && strings.Contains(w.Message, `not used by rule "required-field"`) {
			found = true
		}
	}
	if !found {
		t.Errorf("warnings: %v", res.Warnings)
	}
}
