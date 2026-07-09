package constraints_test

import (
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/constraints"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// makeModel builds a minimal BausteinsichtModel with the given elements, relationships, and constraints.
func makeModel(elements map[string]model.Element, rels []model.Relationship, cs []model.Constraint) *model.BausteinsichtModel {
	return &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"system":    {Notation: "System"},
				"container": {Notation: "Container"},
				"component": {Notation: "Component"},
				"person":    {Notation: "Person"},
			},
		},
		Model:         elements,
		Relationships: rels,
		Constraints:   cs,
	}
}

func countViolations(r constraints.Result) int {
	return r.Total
}

// ─── no-relationship ─────────────────────────────────────────────────────────

func TestNoRelationship_NoViolation(t *testing.T) {
	m := makeModel(
		map[string]model.Element{
			"a": {Kind: "system"},
			"b": {Kind: "container"},
		},
		[]model.Relationship{{From: "a", To: "b"}},
		[]model.Constraint{{
			ID: "c1", Rule: "no-relationship",
			FromKind: "container", ToKind: "system",
		}},
	)
	r := constraints.Evaluate(m)
	if countViolations(r) != 0 {
		t.Errorf("expected 0 violations, got %d: %v", r.Total, r.Violations)
	}
}

func TestNoRelationship_Violation(t *testing.T) {
	m := makeModel(
		map[string]model.Element{
			"a": {Kind: "system"},
			"b": {Kind: "container"},
		},
		[]model.Relationship{{From: "a", To: "b"}},
		[]model.Constraint{{
			ID: "c1", Rule: "no-relationship",
			FromKind: "system", ToKind: "container",
		}},
	)
	r := constraints.Evaluate(m)
	if r.Total != 1 {
		t.Errorf("expected 1 violation, got %d", r.Total)
	}
}

// TestNoRelationship_TagBased_Violation verifies the motivating case from #542:
// two elements of the SAME kind (both "container") can still be distinguished
// by tag, so "UI must not talk to datastore directly" becomes expressible.
func TestNoRelationship_TagBased_Violation(t *testing.T) {
	m := makeModel(
		map[string]model.Element{
			"adminConsole": {Kind: "container", Tags: []string{"ui"}},
			"database":     {Kind: "container", Tags: []string{"datastore"}},
		},
		[]model.Relationship{{From: "adminConsole", To: "database"}},
		[]model.Constraint{{
			ID: "c1", Rule: "no-relationship",
			FromTag: "ui", ToTag: "datastore",
		}},
	)
	r := constraints.Evaluate(m)
	if r.Total != 1 {
		t.Errorf("expected 1 violation, got %d", r.Total)
	}
}

func TestNoRelationship_TagBased_NoViolation(t *testing.T) {
	m := makeModel(
		map[string]model.Element{
			"adminConsole": {Kind: "container", Tags: []string{"ui"}},
			"api":          {Kind: "container", Tags: []string{"service"}},
		},
		[]model.Relationship{{From: "adminConsole", To: "api"}},
		[]model.Constraint{{
			ID: "c1", Rule: "no-relationship",
			FromTag: "ui", ToTag: "datastore",
		}},
	)
	r := constraints.Evaluate(m)
	if r.Total != 0 {
		t.Errorf("expected 0 violations, got %d: %v", r.Total, r.Violations)
	}
}

// TestNoRelationship_ElementID_Violation verifies exact-ID endpoint targeting.
func TestNoRelationship_ElementID_Violation(t *testing.T) {
	m := makeModel(
		map[string]model.Element{
			"legacyBatch": {Kind: "container"},
			"coreApi":     {Kind: "container"},
		},
		[]model.Relationship{{From: "legacyBatch", To: "coreApi"}},
		[]model.Constraint{{
			ID: "c1", Rule: "no-relationship",
			From: "legacyBatch", To: "coreApi",
		}},
	)
	r := constraints.Evaluate(m)
	if r.Total != 1 {
		t.Errorf("expected 1 violation, got %d", r.Total)
	}
}

// TestNoRelationship_IDTakesPrecedenceOverTag verifies that when both an
// exact-ID selector and a tag selector are set on the same endpoint, the
// more specific ID selector wins (#542).
func TestNoRelationship_IDTakesPrecedenceOverTag(t *testing.T) {
	m := makeModel(
		map[string]model.Element{
			"adminConsole": {Kind: "container", Tags: []string{"ui"}},
			"other":        {Kind: "container", Tags: []string{"ui"}},
			"database":     {Kind: "container", Tags: []string{"datastore"}},
		},
		// "other" also has the "ui" tag but the constraint pins "from" to the
		// exact ID "adminConsole" — "other" -> "database" must NOT violate.
		[]model.Relationship{{From: "other", To: "database"}},
		[]model.Constraint{{
			ID: "c1", Rule: "no-relationship",
			From: "adminConsole", FromTag: "ui", ToTag: "datastore",
		}},
	)
	r := constraints.Evaluate(m)
	if r.Total != 0 {
		t.Errorf("expected 0 violations (ID selector should override tag selector), got %d: %v", r.Total, r.Violations)
	}
}

// ─── allowed-relationship ────────────────────────────────────────────────────

func TestAllowedRelationship_NoViolation(t *testing.T) {
	m := makeModel(
		map[string]model.Element{
			"a": {Kind: "system"},
			"b": {Kind: "container"},
		},
		[]model.Relationship{{From: "a", To: "b"}},
		[]model.Constraint{{
			ID: "c1", Rule: "allowed-relationship",
			FromKinds: []string{"system"}, ToKind: "container",
		}},
	)
	r := constraints.Evaluate(m)
	if r.Total != 0 {
		t.Errorf("expected 0 violations, got %d", r.Total)
	}
}

func TestAllowedRelationship_Violation(t *testing.T) {
	m := makeModel(
		map[string]model.Element{
			"p": {Kind: "person"},
			"b": {Kind: "container"},
		},
		[]model.Relationship{{From: "p", To: "b"}},
		[]model.Constraint{{
			ID: "c1", Rule: "allowed-relationship",
			FromKinds: []string{"system"}, ToKind: "container",
		}},
	)
	r := constraints.Evaluate(m)
	if r.Total != 1 {
		t.Errorf("expected 1 violation, got %d", r.Total)
	}
}

// TestAllowedRelationship_TagBased_Violation verifies from-tags allow-list
// targeting (#542), analogous to from-kinds but for elements of the same
// kind distinguished only by tag.
func TestAllowedRelationship_TagBased_Violation(t *testing.T) {
	m := makeModel(
		map[string]model.Element{
			"adminConsole": {Kind: "container", Tags: []string{"ui"}},
			"batchJob":     {Kind: "container", Tags: []string{"internal"}},
			"database":     {Kind: "container", Tags: []string{"datastore"}},
		},
		[]model.Relationship{{From: "batchJob", To: "database"}},
		[]model.Constraint{{
			ID: "c1", Rule: "allowed-relationship",
			FromTags: []string{"internal"}, ToTag: "datastore",
		}},
	)
	r := constraints.Evaluate(m)
	if r.Total != 0 {
		t.Errorf("expected 0 violations (batchJob is tagged 'internal', allowed), got %d: %v", r.Total, r.Violations)
	}
}

func TestAllowedRelationship_TagBased_ViolationBlocked(t *testing.T) {
	m := makeModel(
		map[string]model.Element{
			"adminConsole": {Kind: "container", Tags: []string{"ui"}},
			"database":     {Kind: "container", Tags: []string{"datastore"}},
		},
		[]model.Relationship{{From: "adminConsole", To: "database"}},
		[]model.Constraint{{
			ID: "c1", Rule: "allowed-relationship",
			FromTags: []string{"internal"}, ToTag: "datastore",
		}},
	)
	r := constraints.Evaluate(m)
	if r.Total != 1 {
		t.Errorf("expected 1 violation (adminConsole is tagged 'ui', not in allow-list), got %d", r.Total)
	}
}

// ─── required-field ──────────────────────────────────────────────────────────

func TestRequiredField_MissingDescription(t *testing.T) {
	m := makeModel(
		map[string]model.Element{
			"a": {Kind: "system", Title: "A"},
			"b": {Kind: "system", Title: "B", Description: "ok"},
		},
		nil,
		[]model.Constraint{{
			ID: "c1", Rule: "required-field",
			ElementKind: "system", Field: "description",
		}},
	)
	r := constraints.Evaluate(m)
	if r.Total != 1 {
		t.Errorf("expected 1 violation, got %d", r.Total)
	}
	if len(r.Violations[0].Elements) != 1 {
		t.Errorf("expected 1 offending element, got %d", len(r.Violations[0].Elements))
	}
}

func TestRequiredField_NoViolation(t *testing.T) {
	m := makeModel(
		map[string]model.Element{
			"a": {Kind: "system", Title: "A", Description: "desc"},
		},
		nil,
		[]model.Constraint{{
			ID: "c1", Rule: "required-field",
			ElementKind: "system", Field: "description",
		}},
	)
	r := constraints.Evaluate(m)
	if r.Total != 0 {
		t.Errorf("expected 0 violations, got %d", r.Total)
	}
}

func TestRequiredField_Technology(t *testing.T) {
	m := makeModel(
		map[string]model.Element{
			"a": {Kind: "container"},
		},
		nil,
		[]model.Constraint{{
			ID: "c1", Rule: "required-field",
			ElementKind: "container", Field: "technology",
		}},
	)
	r := constraints.Evaluate(m)
	if r.Total != 1 {
		t.Errorf("expected 1 violation, got %d", r.Total)
	}
}

// TestRequiredField_TagBased verifies element selection by Tag instead of
// ElementKind (#542) — e.g. requiring a description on all "public-api"
// tagged elements regardless of their kind.
func TestRequiredField_TagBased(t *testing.T) {
	m := makeModel(
		map[string]model.Element{
			"a": {Kind: "container", Tags: []string{"public-api"}},
			"b": {Kind: "component", Tags: []string{"public-api"}, Description: "documented"},
			"c": {Kind: "container"}, // no tag — not selected, would otherwise fail
		},
		nil,
		[]model.Constraint{{
			ID: "c1", Rule: "required-field",
			Tag: "public-api", Field: "description",
		}},
	)
	r := constraints.Evaluate(m)
	if r.Total != 1 {
		t.Errorf("expected 1 violation (only 'a' is public-api tagged and missing description), got %d: %v", r.Total, r.Violations)
	}
	if len(r.Violations) > 0 && len(r.Violations[0].Elements) != 1 {
		t.Errorf("expected 1 offending element, got %d", len(r.Violations[0].Elements))
	}
}

// ─── max-depth ───────────────────────────────────────────────────────────────

func TestMaxDepth_NoViolation(t *testing.T) {
	m := makeModel(
		map[string]model.Element{
			"a": {Kind: "system", Children: map[string]model.Element{
				"b": {Kind: "container"},
			}},
		},
		nil,
		[]model.Constraint{{ID: "c1", Rule: "max-depth", Max: 2}},
	)
	r := constraints.Evaluate(m)
	if r.Total != 0 {
		t.Errorf("expected 0 violations, got %d", r.Total)
	}
}

func TestMaxDepth_Violation(t *testing.T) {
	m := makeModel(
		map[string]model.Element{
			"a": {Kind: "system", Children: map[string]model.Element{
				"b": {Kind: "container", Children: map[string]model.Element{
					"c": {Kind: "component"},
				}},
			}},
		},
		nil,
		[]model.Constraint{{ID: "c1", Rule: "max-depth", Max: 2}},
	)
	r := constraints.Evaluate(m)
	if r.Total != 1 {
		t.Errorf("expected 1 violation, got %d", r.Total)
	}
}

// ─── no-circular-dependency ──────────────────────────────────────────────────

func TestNoCircularDependency_NoCycle(t *testing.T) {
	m := makeModel(
		map[string]model.Element{
			"a": {Kind: "system"},
			"b": {Kind: "system"},
			"c": {Kind: "system"},
		},
		[]model.Relationship{
			{From: "a", To: "b"},
			{From: "b", To: "c"},
		},
		[]model.Constraint{{ID: "c1", Rule: "no-circular-dependency", Description: "no cycles"}},
	)
	r := constraints.Evaluate(m)
	if r.Total != 0 {
		t.Errorf("expected 0 violations, got %d: %v", r.Total, r.Violations)
	}
}

func TestNoCircularDependency_Cycle(t *testing.T) {
	m := makeModel(
		map[string]model.Element{
			"a": {Kind: "system"},
			"b": {Kind: "system"},
		},
		[]model.Relationship{
			{From: "a", To: "b"},
			{From: "b", To: "a"},
		},
		[]model.Constraint{{ID: "c1", Rule: "no-circular-dependency", Description: "no cycles"}},
	)
	r := constraints.Evaluate(m)
	if r.Total != 1 {
		t.Errorf("expected 1 violation, got %d", r.Total)
	}
	if len(r.Violations[0].Elements) == 0 {
		t.Error("expected cycle elements to be listed")
	}
}

// ─── technology-allowed ──────────────────────────────────────────────────────

func TestTechnologyAllowed_NoViolation(t *testing.T) {
	m := makeModel(
		map[string]model.Element{
			"a": {Kind: "container", Technology: "Go"},
			"b": {Kind: "container", Technology: "go"},
		},
		nil,
		[]model.Constraint{{
			ID: "c1", Rule: "technology-allowed",
			ElementKind: "container", Technologies: []string{"Go", "Java"},
		}},
	)
	r := constraints.Evaluate(m)
	if r.Total != 0 {
		t.Errorf("expected 0 violations, got %d", r.Total)
	}
}

func TestTechnologyAllowed_Violation(t *testing.T) {
	m := makeModel(
		map[string]model.Element{
			"a": {Kind: "container", Technology: "PHP"},
		},
		nil,
		[]model.Constraint{{
			ID: "c1", Rule: "technology-allowed",
			ElementKind: "container", Technologies: []string{"Go", "Java"},
		}},
	)
	r := constraints.Evaluate(m)
	if r.Total != 1 {
		t.Errorf("expected 1 violation, got %d", r.Total)
	}
}

func TestTechnologyAllowed_EmptyTechnologySkipped(t *testing.T) {
	m := makeModel(
		map[string]model.Element{
			"a": {Kind: "container"},
		},
		nil,
		[]model.Constraint{{
			ID: "c1", Rule: "technology-allowed",
			ElementKind: "container", Technologies: []string{"Go"},
		}},
	)
	r := constraints.Evaluate(m)
	if r.Total != 0 {
		t.Errorf("expected 0 violations for unset technology, got %d", r.Total)
	}
}

// ─── unknown rule ────────────────────────────────────────────────────────────

func TestUnknownRule(t *testing.T) {
	m := makeModel(
		map[string]model.Element{"a": {Kind: "system"}},
		nil,
		[]model.Constraint{{ID: "c1", Rule: "does-not-exist"}},
	)
	r := constraints.Evaluate(m)
	if r.Total != 1 {
		t.Errorf("expected 1 violation for unknown rule, got %d", r.Total)
	}
}

// ─── multiple constraints ────────────────────────────────────────────────────

func TestMultipleConstraints(t *testing.T) {
	m := makeModel(
		map[string]model.Element{
			"a": {Kind: "system"},
			"b": {Kind: "container"},
		},
		[]model.Relationship{{From: "a", To: "b"}},
		[]model.Constraint{
			{ID: "c1", Rule: "required-field", ElementKind: "system", Field: "description"},
			{ID: "c2", Rule: "required-field", ElementKind: "container", Field: "technology"},
		},
	)
	r := constraints.Evaluate(m)
	if r.Total != 2 {
		t.Errorf("expected 2 violations, got %d", r.Total)
	}
}
