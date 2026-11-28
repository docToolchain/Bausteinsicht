package model

import "fmt"

// ConstraintRules lists the rule names a Constraint may use, sorted. It is the
// single source for `validate`, the JSON Schema enum and the lint engine's
// own test, which checks that every name is evaluated (#689).
var ConstraintRules = []string{
	"allowed-relationship",
	"max-depth",
	"no-circular-dependency",
	"no-relationship",
	"required-field",
	"technology-allowed",
}

// IsConstraintRule reports whether rule is one of ConstraintRules.
func IsConstraintRule(rule string) bool {
	for _, r := range ConstraintRules {
		if r == rule {
			return true
		}
	}
	return false
}

// ConstraintProblem is a missing or unusable part of a constraint definition.
type ConstraintProblem struct {
	Field   string // JSON name of the field, "" for the constraint as a whole
	Message string
}

// constraintFields lists, per rule, the JSON names of the fields the
// rule reads; "rule", "id" and "description" are read by every rule. It is the
// base of the "field of another rule" warning (#727).
var constraintFields = map[string][]string{
	"no-relationship":        {"from", "from-tag", "from-kind", "to", "to-tag", "to-kind"},
	"allowed-relationship":   {"from", "from-tag", "from-kind", "from-tags", "from-kinds", "to", "to-tag", "to-kind"},
	"required-field":         {"element-kind", "tag", "field"},
	"max-depth":              {"max"},
	"no-circular-dependency": {},
	"technology-allowed":     {"element-kind", "tag", "technologies"},
}

// CheckConstraint returns what is missing in the definition of c for its rule:
// a constraint without the selectors or values its rule needs matches nothing
// (or everything), so it is reported instead of evaluated (#727). An unknown
// rule is reported elsewhere.
func CheckConstraint(c Constraint) []ConstraintProblem {
	var ps []ConstraintProblem
	need := func(field, msg string) { ps = append(ps, ConstraintProblem{Field: field, Message: msg}) }
	switch c.Rule {
	case "no-relationship", "allowed-relationship":
		fromSet := c.From != "" || c.FromTag != "" || c.FromKind != ""
		if c.Rule == "allowed-relationship" {
			fromSet = fromSet || len(c.FromTags) > 0 || len(c.FromKinds) > 0
		}
		if !fromSet {
			need("", fmt.Sprintf("rule %q needs a from selector (from, from-tag or from-kind)", c.Rule))
		}
		if c.To == "" && c.ToTag == "" && c.ToKind == "" {
			need("", fmt.Sprintf("rule %q needs a to selector (to, to-tag or to-kind)", c.Rule))
		}
	case "required-field":
		if c.ElementKind == "" && c.Tag == "" {
			need("", `rule "required-field" needs an element-kind or tag selector`)
		}
		switch c.Field {
		case "description", "technology", "title":
		default:
			need("field", fmt.Sprintf("field %q must be one of description, technology, title", c.Field))
		}
	case "max-depth":
		if c.Max < 1 {
			need("max", fmt.Sprintf("max must be at least 1, got %d", c.Max))
		}
	case "technology-allowed":
		if c.ElementKind == "" && c.Tag == "" {
			need("", `rule "technology-allowed" needs an element-kind or tag selector`)
		}
		if len(c.Technologies) == 0 {
			need("technologies", "technologies must list at least one allowed technology")
		}
	}
	return ps
}

// UnusedConstraintFields returns the JSON names of the fields that are set on c
// but not read by its rule.
func UnusedConstraintFields(c Constraint) []string {
	used, known := constraintFields[c.Rule]
	if !known {
		return nil
	}
	set := map[string]bool{
		"from": c.From != "", "from-tag": c.FromTag != "", "from-kind": c.FromKind != "",
		"from-tags": len(c.FromTags) > 0, "from-kinds": len(c.FromKinds) > 0,
		"to": c.To != "", "to-tag": c.ToTag != "", "to-kind": c.ToKind != "",
		"element-kind": c.ElementKind != "", "tag": c.Tag != "", "field": c.Field != "",
		"max": c.Max != 0, "technologies": len(c.Technologies) > 0,
	}
	for _, f := range used {
		delete(set, f)
	}
	var unused []string
	for _, f := range []string{"from", "from-tag", "from-kind", "from-tags", "from-kinds", "to", "to-tag", "to-kind", "element-kind", "tag", "field", "max", "technologies"} {
		if set[f] {
			unused = append(unused, f)
		}
	}
	return unused
}
