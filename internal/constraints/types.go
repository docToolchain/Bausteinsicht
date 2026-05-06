// Package constraints evaluates architectural rules defined in the model's
// constraints section and reports violations.
package constraints

// Violation describes a single rule failure.
type Violation struct {
	ConstraintID string   `json:"constraint_id"`
	Message      string   `json:"message"`
	Elements     []string `json:"elements,omitempty"`
}

// Result holds all violations found during a lint run.
type Result struct {
	Violations []Violation `json:"violations"`
	Total      int         `json:"total"`
}
