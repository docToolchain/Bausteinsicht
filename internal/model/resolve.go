package model

import (
	"fmt"
	"strings"
)

// Resolve traverses the model hierarchy using dot notation (e.g., "webshop.api.auth").
func Resolve(m *BausteinsichtModel, id string) (*Element, error) {
	parts := strings.Split(id, ".")
	root := parts[0]

	elem, ok := m.Model[root]
	if !ok {
		return nil, fmt.Errorf("element %q not found", id)
	}

	for _, part := range parts[1:] {
		if elem.Children == nil {
			return nil, fmt.Errorf("element %q not found: no children at this level", id)
		}
		child, ok := elem.Children[part]
		if !ok {
			return nil, fmt.Errorf("element %q not found", id)
		}
		elem = child
	}

	return &elem, nil
}

// flattenInto recursively adds elements to the map with their full dot-notation path.
func flattenInto(children map[string]Element, prefix string, result map[string]*Element) {
	for key, elem := range children {
		fullID := prefix + key
		e := elem
		result[fullID] = &e
		if elem.Children != nil {
			flattenInto(elem.Children, fullID+".", result)
		}
	}
}

// FlattenElements returns all elements keyed by full dot-notation ID path.
func FlattenElements(m *BausteinsichtModel) map[string]*Element {
	result := make(map[string]*Element)
	flattenInto(m.Model, "", result)
	return result
}

// MatchPattern matches elements in the flat map against a pattern.
// Supported patterns:
//   - "id"         — exact match
//   - "prefix.*"   — direct children of prefix (one level deep)
//   - "prefix.**"  — all descendants of prefix (recursive)
//   - "*"          — all top-level elements (no dots in ID)
//   - "**"         — all elements
func MatchPattern(flatMap map[string]*Element, pattern string) []string {
	var matches []string

	switch {
	case pattern == "**":
		// Match all elements.
		for id := range flatMap {
			matches = append(matches, id)
		}

	case pattern == "*":
		// Match top-level elements only (no dots in ID).
		for id := range flatMap {
			if !strings.Contains(id, ".") {
				matches = append(matches, id)
			}
		}

	case strings.HasSuffix(pattern, ".**"):
		// Match all descendants of prefix (recursive).
		prefix := strings.TrimSuffix(pattern, "**")
		for id := range flatMap {
			if strings.HasPrefix(id, prefix) {
				matches = append(matches, id)
			}
		}

	case strings.HasSuffix(pattern, ".*"):
		// Match direct children only (one level deep).
		prefix := strings.TrimSuffix(pattern, "*")
		depth := strings.Count(prefix, ".")
		for id := range flatMap {
			if !strings.HasPrefix(id, prefix) {
				continue
			}
			rest := id[len(prefix):]
			if !strings.Contains(rest, ".") && strings.Count(id, ".") == depth {
				matches = append(matches, id)
			}
		}

	default:
		// Exact match.
		if _, ok := flatMap[pattern]; ok {
			matches = append(matches, pattern)
		}
	}

	return matches
}

// ResolveView resolves view includes/excludes to a list of element IDs.
// Starts with include patterns, then removes exclude patterns.
func ResolveView(m *BausteinsichtModel, view *View) ([]string, error) {
	if len(view.Include) == 0 {
		return []string{}, nil
	}

	flatMap := FlattenElements(m)

	included := make(map[string]bool)
	for _, pattern := range view.Include {
		for _, id := range MatchPattern(flatMap, pattern) {
			included[id] = true
		}
	}

	for _, pattern := range view.Exclude {
		for _, id := range MatchPattern(flatMap, pattern) {
			delete(included, id)
		}
	}

	result := make([]string, 0, len(included))
	for id := range included {
		result = append(result, id)
	}
	return result, nil
}
