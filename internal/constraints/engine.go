package constraints

import (
	"fmt"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// ErrUnknownRule is returned when a constraint specifies an unsupported rule type.
type ErrUnknownRule struct {
	Rule string
}

func (e ErrUnknownRule) Error() string {
	return fmt.Sprintf("unknown constraint rule %q", e.Rule)
}

// Evaluate runs all constraints against the model and returns the aggregated result.
// Unknown rule types are reported as violations so they don't silently pass.
func Evaluate(m *model.BausteinsichtModel) Result {
	var all []Violation
	for _, c := range m.Constraints {
		vs, err := evaluate(c, m)
		if err != nil {
			all = append(all, Violation{
				ConstraintID: c.ID,
				Message:      err.Error(),
			})
			continue
		}
		all = append(all, vs...)
	}
	return Result{Violations: all, Total: len(all)}
}

func evaluate(c model.Constraint, m *model.BausteinsichtModel) ([]Violation, error) {
	switch c.Rule {
	case "no-relationship":
		return noRelationship(c, m), nil
	case "allowed-relationship":
		return allowedRelationship(c, m), nil
	case "required-field":
		return requiredField(c, m), nil
	case "max-depth":
		return maxDepth(c, m), nil
	case "no-circular-dependency":
		return noCircularDependency(c, m), nil
	case "technology-allowed":
		return technologyAllowed(c, m), nil
	default:
		return nil, ErrUnknownRule{Rule: c.Rule}
	}
}
