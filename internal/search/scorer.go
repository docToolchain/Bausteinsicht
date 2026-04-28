package search

import (
	"strings"
)

// fieldMatch checks whether the field value contains all query words (case-insensitive).
// Returns the weight if it matches, 0 otherwise. An exact full-string match
// (after lowercasing) returns 10× the weight to prioritise ID hits.
func fieldMatch(value string, words []string, weight int) (score int, matched bool) {
	if value == "" || weight == 0 {
		return 0, false
	}
	lower := strings.ToLower(value)
	for _, w := range words {
		if !strings.Contains(lower, w) {
			return 0, false
		}
	}
	// Exact match bonus: single-word query that equals the whole field value.
	if len(words) == 1 && lower == words[0] {
		return weight * 10, true
	}
	return weight, true
}

// scoreElement computes a relevance score for an element.
// Returns the total score and the list of field names that contributed.
func scoreElement(id, title, description, technology, kind string, tags []string, words []string) (int, []string) {
	type field struct {
		name   string
		value  string
		weight int
	}
	fields := []field{
		{"id", id, 3},
		{"title", title, 3},
		{"technology", technology, 2},
		{"kind", kind, 2},
		{"description", description, 1},
	}

	total := 0
	var matched []string
	for _, f := range fields {
		if s, ok := fieldMatch(f.value, words, f.weight); ok {
			total += s
			matched = append(matched, f.name)
		}
	}
	for _, tag := range tags {
		if s, ok := fieldMatch(tag, words, 2); ok {
			total += s
			if !contains(matched, "tags") {
				matched = append(matched, "tags")
			}
		}
	}
	return total, matched
}

// scoreRelationship computes a relevance score for a relationship.
func scoreRelationship(id, label, kind, fromTitle, toTitle string, words []string) (int, []string) {
	type field struct {
		name   string
		value  string
		weight int
	}
	fields := []field{
		{"id", id, 3},
		{"label", label, 3},
		{"kind", kind, 2},
		{"from", fromTitle, 2},
		{"to", toTitle, 2},
	}

	total := 0
	var matched []string
	for _, f := range fields {
		if s, ok := fieldMatch(f.value, words, f.weight); ok {
			total += s
			matched = append(matched, f.name)
		}
	}
	return total, matched
}

// scoreView computes a relevance score for a view.
func scoreView(key, title, description string, words []string) (int, []string) {
	type field struct {
		name   string
		value  string
		weight int
	}
	fields := []field{
		{"key", key, 3},
		{"title", title, 3},
		{"description", description, 1},
	}

	total := 0
	var matched []string
	for _, f := range fields {
		if s, ok := fieldMatch(f.value, words, f.weight); ok {
			total += s
			matched = append(matched, f.name)
		}
	}
	return total, matched
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
