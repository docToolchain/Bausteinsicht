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

// Validate checks the model for consistency and returns all found errors.
func Validate(m *BausteinsichtModel) []ValidationError {
	var errs []ValidationError
	errs = append(errs, validateElements(m)...)
	errs = append(errs, validateRelationships(m)...)
	errs = append(errs, validateViews(m)...)
	return errs
}

func validateElements(m *BausteinsichtModel) []ValidationError {
	var errs []ValidationError
	for id, elem := range m.Model {
		if err := validateElementID(id); err != nil {
			errs = append(errs, ValidationError{Path: "model." + id, Message: err.Error()})
		}
		errs = append(errs, validateElement(m, "model."+id, elem)...)
	}
	return errs
}

func validateElement(m *BausteinsichtModel, path string, elem Element) []ValidationError {
	var errs []ValidationError

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
		errs = append(errs, validateElement(m, path+"."+childID, child)...)
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

func validateViews(m *BausteinsichtModel) []ValidationError {
	var errs []ValidationError
	for id, view := range m.Views {
		path := "views." + id
		if view.Title == "" {
			errs = append(errs, ValidationError{Path: path, Message: "missing required field \"title\""})
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

// validateElementID checks that an element ID is valid.
func validateElementID(id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("invalid element ID %q: must not be empty or whitespace", id)
	}
	return nil
}

// lookupElement resolves a dot-notation path to an Element within the model.
func lookupElement(m *BausteinsichtModel, path string) (Element, error) {
	parts := strings.SplitN(path, ".", 2)
	elem, ok := m.Model[parts[0]]
	if !ok {
		return Element{}, fmt.Errorf("element %q not found", parts[0])
	}
	if len(parts) == 1 {
		return elem, nil
	}
	return lookupChild(elem, parts[1])
}

func lookupChild(parent Element, path string) (Element, error) {
	parts := strings.SplitN(path, ".", 2)
	child, ok := parent.Children[parts[0]]
	if !ok {
		return Element{}, fmt.Errorf("element %q not found", parts[0])
	}
	if len(parts) == 1 {
		return child, nil
	}
	return lookupChild(child, parts[1])
}
