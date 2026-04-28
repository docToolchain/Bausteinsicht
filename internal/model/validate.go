package model

import (
	"fmt"
	"strings"
)

// ValidationError describes a single validation problem with its model path.
type ValidationError struct {
	Path    string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Path, e.Message)
}

// ValidationWarning describes a non-fatal issue with the model.
type ValidationWarning struct {
	Path    string
	Message string
}

// ValidationResult holds both errors and warnings from validation.
type ValidationResult struct {
	Errors   []ValidationError
	Warnings []ValidationWarning
}

// Validate checks the model for consistency and returns all found errors.
func Validate(m *BausteinsichtModel) []ValidationError {
	result := ValidateWithWarnings(m)
	return result.Errors
}

// ValidateWithWarnings checks the model for consistency and returns errors and warnings.
func ValidateWithWarnings(m *BausteinsichtModel) ValidationResult {
	var result ValidationResult
	result.Errors = append(result.Errors, validateElements(m)...)
	result.Errors = append(result.Errors, validateRelationships(m)...)
	result.Errors = append(result.Errors, validateViews(m)...)
	result.Errors = append(result.Errors, validateDynamicViews(m)...)
	result.Warnings = append(result.Warnings, validateEmptyModel(m)...)
	return result
}

// validateEmptyModel checks for models with no specification or no elements.
func validateEmptyModel(m *BausteinsichtModel) []ValidationWarning {
	var warnings []ValidationWarning
	if len(m.Specification.Elements) == 0 {
		warnings = append(warnings, ValidationWarning{
			Path:    "specification",
			Message: "no element kinds defined in specification",
		})
	}
	if len(m.Model) == 0 {
		warnings = append(warnings, ValidationWarning{
			Path:    "model",
			Message: "model is empty (no elements defined)",
		})
	}
	return warnings
}

func validateElements(m *BausteinsichtModel) []ValidationError {
	var errs []ValidationError
	for id, elem := range m.Model {
		if err := validateElementID(id); err != nil {
			errs = append(errs, ValidationError{Path: "model." + id, Message: err.Error()})
		}
		errs = append(errs, validateElement(m, "model."+id, elem, 1)...)
	}
	return errs
}

func validateElement(m *BausteinsichtModel, path string, elem Element, depth int) []ValidationError {
	var errs []ValidationError

	if depth > MaxElementDepth {
		errs = append(errs, ValidationError{
			Path:    path,
			Message: fmt.Sprintf("element nesting exceeds maximum depth of %d", MaxElementDepth),
		})
		return errs
	}

	if elem.Kind == "" {
		errs = append(errs, ValidationError{Path: path, Message: "missing required field \"kind\""})
	} else {
		kindDef, known := m.Specification.Elements[elem.Kind]
		if !known {
			errs = append(errs, ValidationError{
				Path:    path,
				Message: fmt.Sprintf("unknown kind %q", elem.Kind),
			})
		} else if len(elem.Children) > 0 && !kindDef.Container {
			errs = append(errs, ValidationError{
				Path:    path,
				Message: fmt.Sprintf("kind %q does not allow children (container: false)", elem.Kind),
			})
		}
	}

	if elem.Title == "" {
		errs = append(errs, ValidationError{Path: path, Message: "missing required field \"title\""})
	}

	for childID, child := range elem.Children {
		if err := validateElementID(childID); err != nil {
			errs = append(errs, ValidationError{Path: path + "." + childID, Message: err.Error()})
		}
		errs = append(errs, validateElement(m, path+"."+childID, child, depth+1)...)
	}

	return errs
}

func validateRelationships(m *BausteinsichtModel) []ValidationError {
	var errs []ValidationError
	// Track seen relationships keyed by "from->to->kind->label" to allow
	// multiple relationships between the same pair with different kind or label. (#142)
	type relSig struct {
		from, to, kind, label string
	}
	seen := make(map[relSig]int) // signature → first index

	for i, rel := range m.Relationships {
		path := fmt.Sprintf("relationships[%d]", i)

		if _, err := lookupElement(m, rel.From); err != nil {
			errs = append(errs, ValidationError{
				Path:    path,
				Message: fmt.Sprintf("from %q does not resolve to an existing element", rel.From),
			})
		}
		if _, err := lookupElement(m, rel.To); err != nil {
			errs = append(errs, ValidationError{
				Path:    path,
				Message: fmt.Sprintf("to %q does not resolve to an existing element", rel.To),
			})
		}
		if rel.Kind != "" {
			if _, known := m.Specification.Relationships[rel.Kind]; !known {
				errs = append(errs, ValidationError{
					Path:    path,
					Message: fmt.Sprintf("unknown relationship kind %q", rel.Kind),
				})
			}
		}

		// Detect fully duplicate relationships (same from, to, kind, and label). (#117, #142)
		// Multiple relationships between the same pair are allowed if they
		// differ in kind or label.
		sig := relSig{from: rel.From, to: rel.To, kind: rel.Kind, label: rel.Label}
		if firstIdx, exists := seen[sig]; exists {
			errs = append(errs, ValidationError{
				Path:    path,
				Message: fmt.Sprintf("duplicate relationship %s → %s (first at relationships[%d])", rel.From, rel.To, firstIdx),
			})
		} else {
			seen[sig] = i
		}
	}
	return errs
}

// validLayouts is the set of allowed values for View.Layout.
var validLayouts = map[string]bool{
	"":        true,
	"layered": true,
	"grid":    true,
	"none":    true,
}

func validateViews(m *BausteinsichtModel) []ValidationError {
	var errs []ValidationError
	for id, view := range m.Views {
		path := "views." + id
		if view.Title == "" {
			errs = append(errs, ValidationError{Path: path, Message: "missing required field \"title\""})
		}
		if !validLayouts[view.Layout] {
			errs = append(errs, ValidationError{
				Path:    path,
				Message: fmt.Sprintf("invalid layout %q (must be \"layered\", \"grid\", \"none\", or empty)", view.Layout),
			})
		}
		if view.Scope != "" {
			if _, err := lookupElement(m, view.Scope); err != nil {
				errs = append(errs, ValidationError{
					Path:    path,
					Message: fmt.Sprintf("scope %q does not resolve to an existing element", view.Scope),
				})
			}
		}
		for _, entry := range view.Include {
			if strings.Contains(entry, "*") {
				continue
			}
			if _, err := lookupElement(m, entry); err != nil {
				errs = append(errs, ValidationError{
					Path:    path + ".include",
					Message: fmt.Sprintf("element %q does not exist", entry),
				})
			}
		}
		for _, entry := range view.Exclude {
			if strings.Contains(entry, "*") {
				continue
			}
			if _, err := lookupElement(m, entry); err != nil {
				errs = append(errs, ValidationError{
					Path:    path + ".exclude",
					Message: fmt.Sprintf("element %q does not exist", entry),
				})
			}
		}
	}
	return errs
}

var validStepTypes = map[StepType]bool{
	StepSync:   true,
	StepAsync:  true,
	StepReturn: true,
	"":         true, // omitted → default sync
}

func validateDynamicViews(m *BausteinsichtModel) []ValidationError {
	var errs []ValidationError
	for vi, dv := range m.DynamicViews {
		path := fmt.Sprintf("dynamicViews[%d]", vi)
		if dv.Key == "" {
			errs = append(errs, ValidationError{Path: path, Message: "missing required field \"key\""})
		}
		if dv.Title == "" {
			errs = append(errs, ValidationError{Path: path, Message: "missing required field \"title\""})
		}
		if len(dv.Steps) == 0 {
			errs = append(errs, ValidationError{Path: path, Message: "dynamic view must have at least one step"})
			continue
		}
		seenIndex := make(map[int]bool)
		for si, step := range dv.Steps {
			spath := fmt.Sprintf("%s.steps[%d]", path, si)
			if _, err := lookupElement(m, step.From); err != nil {
				errs = append(errs, ValidationError{
					Path:    spath,
					Message: fmt.Sprintf("from %q does not resolve to an existing element", step.From),
				})
			}
			if _, err := lookupElement(m, step.To); err != nil {
				errs = append(errs, ValidationError{
					Path:    spath,
					Message: fmt.Sprintf("to %q does not resolve to an existing element", step.To),
				})
			}
			if !validStepTypes[step.Type] {
				errs = append(errs, ValidationError{
					Path:    spath,
					Message: fmt.Sprintf("invalid type %q (must be \"sync\", \"async\", or \"return\")", step.Type),
				})
			}
			if seenIndex[step.Index] {
				errs = append(errs, ValidationError{
					Path:    spath,
					Message: fmt.Sprintf("duplicate step index %d", step.Index),
				})
			}
			seenIndex[step.Index] = true
		}
	}
	return errs
}

// validateElementID checks that an element ID is valid.
func validateElementID(id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("invalid element ID %q: must not be empty or whitespace", id)
	}
	return nil
}

// lookupElement resolves a dot-notation path to an Element within the model.
func lookupElement(m *BausteinsichtModel, path string) (Element, error) {
	head, rest, hasDot := strings.Cut(path, ".")
	elem, ok := m.Model[head]
	if !ok {
		return Element{}, fmt.Errorf("element %q not found", head)
	}
	if !hasDot {
		return elem, nil
	}
	return lookupChild(elem, rest)
}

func lookupChild(parent Element, path string) (Element, error) {
	head, rest, hasDot := strings.Cut(path, ".")
	child, ok := parent.Children[head]
	if !ok {
		return Element{}, fmt.Errorf("element %q not found", head)
	}
	if !hasDot {
		return child, nil
	}
	return lookupChild(child, rest)
}
