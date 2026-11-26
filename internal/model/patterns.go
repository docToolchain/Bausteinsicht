package model

import (
	"strings"
	"unicode"
)

// ExpandPattern takes a pattern definition and applies variable substitution
// to generate concrete elements and relationships.
// baseID is used for {base}, title is used for {Title} and {BASE}.
func ExpandPattern(pattern PatternDefinition, baseID, title string) ([]Element, []Relationship, error) {
	if title == "" {
		title = baseID
	}

	vars := map[string]string{
		"{base}":  baseID,
		"{Title}": toTitleCase(title),
		"{BASE}":  strings.ToUpper(baseID),
	}

	// Expand elements (including nested children)
	elements := make([]Element, len(pattern.Elements))
	for i, tmpl := range pattern.Elements {
		elements[i] = expandPatternElement(tmpl, vars)
	}

	// Expand relationships
	relationships := make([]Relationship, len(pattern.Relationships))
	for i, tmpl := range pattern.Relationships {
		relationships[i] = Relationship{
			From:        replaceVars(tmpl.From, vars),
			To:          replaceVars(tmpl.To, vars),
			Label:       replaceVars(tmpl.Label, vars),
			Kind:        tmpl.Kind,
			Description: replaceVars(tmpl.Description, vars),
		}
	}

	return elements, relationships, nil
}

// expandPatternElement recursively expands an element template, including children
func expandPatternElement(tmpl PatternElement, vars map[string]string) Element {
	elem := Element{
		Kind:        tmpl.Kind,
		Title:       replaceVars(tmpl.Title, vars),
		Description: replaceVars(tmpl.Description, vars),
		Technology:  replaceVars(tmpl.Technology, vars),
		Tags:        tmpl.Tags,
	}

	// Recursively expand children if present
	if len(tmpl.Children) > 0 {
		elem.Children = make(map[string]Element, len(tmpl.Children))
		for _, childTmpl := range tmpl.Children {
			childID := replaceVars(childTmpl.ID, vars)
			elem.Children[childID] = expandPatternElement(childTmpl, vars)
		}
	}

	return elem
}

// ExpandPatternIDs applies variable substitution to element and relationship IDs
func ExpandPatternIDs(pattern PatternDefinition, baseID string) ([]string, []string, error) {
	vars := map[string]string{
		"{base}": baseID,
		"{BASE}": strings.ToUpper(baseID),
	}

	var elemIDs []string
	for _, tmpl := range pattern.Elements {
		elemIDs = append(elemIDs, expandPatternElementIDs(tmpl, vars)...)
	}

	relIDs := make([]string, len(pattern.Relationships))
	for i, tmpl := range pattern.Relationships {
		relIDs[i] = replaceVars(tmpl.ID, vars)
	}

	return elemIDs, relIDs, nil
}

// expandPatternElementIDs recursively extracts all element IDs from a pattern element
func expandPatternElementIDs(tmpl PatternElement, vars map[string]string) []string {
	ids := []string{replaceVars(tmpl.ID, vars)}
	for _, childTmpl := range tmpl.Children {
		ids = append(ids, expandPatternElementIDs(childTmpl, vars)...)
	}
	return ids
}

// replaceVars substitutes template variables in a string
func replaceVars(s string, vars map[string]string) string {
	result := s
	for k, v := range vars {
		result = strings.ReplaceAll(result, k, v)
	}
	return result
}

// toTitleCase converts "order" to "Order"
func toTitleCase(s string) string {
	if len(s) == 0 {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// ExpandPatternTopLevelIDs returns the IDs of the top-level elements of the
// expanded pattern, in the order ExpandPattern returns the elements (#702).
// ExpandPatternIDs lists every ID including the children, so it cannot be
// paired with the result of ExpandPattern.
func ExpandPatternTopLevelIDs(pattern PatternDefinition, baseID string) []string {
	vars := map[string]string{"{base}": baseID, "{BASE}": strings.ToUpper(baseID)}
	ids := make([]string, len(pattern.Elements))
	for i, tmpl := range pattern.Elements {
		ids[i] = replaceVars(tmpl.ID, vars)
	}
	return ids
}

// CheckPatternConflicts reports the top-level IDs of the expanded pattern that
// already exist in the model. The children of the pattern live below the new
// top-level elements, so an unrelated element with the ID of a child is no
// conflict.
func CheckPatternConflicts(m *BausteinsichtModel, pattern PatternDefinition, baseID string) ([]string, error) {
	var conflicts []string
	for _, id := range ExpandPatternTopLevelIDs(pattern, baseID) {
		if _, exists := m.Model[id]; exists {
			conflicts = append(conflicts, id)
		}
	}
	return conflicts, nil
}
