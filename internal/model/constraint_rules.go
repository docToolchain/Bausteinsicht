package model

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
