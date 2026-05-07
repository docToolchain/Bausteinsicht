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

	// Expand elements
	elements := make([]Element, len(pattern.Elements))
	for i, tmpl := range pattern.Elements {
		elements[i] = Element{
			Kind:        tmpl.Kind,
			Title:       replaceVars(tmpl.Title, vars),
			Description: replaceVars(tmpl.Description, vars),
			Technology:  replaceVars(tmpl.Technology, vars),
			Tags:        tmpl.Tags,
		}
		// Element ID is set separately by caller
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

// ExpandPatternIDs applies variable substitution to element and relationship IDs
func ExpandPatternIDs(pattern PatternDefinition, baseID string) ([]string, []string, error) {
	vars := map[string]string{
		"{base}": baseID,
		"{BASE}": strings.ToUpper(baseID),
	}

	elemIDs := make([]string, len(pattern.Elements))
	for i, tmpl := range pattern.Elements {
		elemIDs[i] = replaceVars(tmpl.ID, vars)
	}

	relIDs := make([]string, len(pattern.Relationships))
	for i, tmpl := range pattern.Relationships {
		relIDs[i] = replaceVars(tmpl.ID, vars)
	}

	return elemIDs, relIDs, nil
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

// CheckPatternConflicts checks if any generated IDs already exist in the model
func CheckPatternConflicts(m *BausteinsichtModel, pattern PatternDefinition, baseID string) ([]string, error) {
	elemIDs, _, err := ExpandPatternIDs(pattern, baseID)
	if err != nil {
		return nil, err
	}

	flat, _ := FlattenElements(m)
	var conflicts []string

	for _, id := range elemIDs {
		if _, exists := flat[id]; exists {
			conflicts = append(conflicts, id)
		}
	}

	return conflicts, nil
}
