package model

import (
	"testing"
)

func TestFilterElementsByTags_NoFilters(t *testing.T) {
	elements := map[string]*Element{
		"elem1": {Kind: "system", Title: "Elem1", Tags: []string{"tag1"}},
		"elem2": {Kind: "system", Title: "Elem2", Tags: []string{"tag2"}},
	}

	result := FilterElementsByTags(elements, nil, nil)
	if len(result) != 2 {
		t.Errorf("expected 2 elements, got %d", len(result))
	}
}

func TestFilterElementsByTags_FilterTagsIntersection(t *testing.T) {
	elements := map[string]*Element{
		"elem1": {Kind: "system", Title: "Elem1", Tags: []string{"backend", "critical"}},
		"elem2": {Kind: "system", Title: "Elem2", Tags: []string{"backend"}},
		"elem3": {Kind: "system", Title: "Elem3", Tags: []string{"frontend"}},
	}

	// Filter for elements with both "backend" AND "critical"
	result := FilterElementsByTags(elements, []string{"backend", "critical"}, nil)
	if len(result) != 1 {
		t.Errorf("expected 1 element, got %d", len(result))
	}
	if _, ok := result["elem1"]; !ok {
		t.Errorf("expected elem1 in result")
	}
}

func TestFilterElementsByTags_SingleFilterTag(t *testing.T) {
	elements := map[string]*Element{
		"elem1": {Kind: "system", Title: "Elem1", Tags: []string{"backend"}},
		"elem2": {Kind: "system", Title: "Elem2", Tags: []string{"backend", "critical"}},
		"elem3": {Kind: "system", Title: "Elem3", Tags: []string{"frontend"}},
	}

	result := FilterElementsByTags(elements, []string{"backend"}, nil)
	if len(result) != 2 {
		t.Errorf("expected 2 elements, got %d", len(result))
	}
	if _, ok := result["elem1"]; !ok {
		t.Errorf("expected elem1 in result")
	}
	if _, ok := result["elem2"]; !ok {
		t.Errorf("expected elem2 in result")
	}
}

func TestFilterElementsByTags_ExcludeTagsUnion(t *testing.T) {
	elements := map[string]*Element{
		"elem1": {Kind: "system", Title: "Elem1", Tags: []string{"experimental"}},
		"elem2": {Kind: "system", Title: "Elem2", Tags: []string{"deprecated"}},
		"elem3": {Kind: "system", Title: "Elem3", Tags: []string{"stable"}},
	}

	// Exclude elements with ANY of "experimental" OR "deprecated"
	result := FilterElementsByTags(elements, nil, []string{"experimental", "deprecated"})
	if len(result) != 1 {
		t.Errorf("expected 1 element, got %d", len(result))
	}
	if _, ok := result["elem3"]; !ok {
		t.Errorf("expected elem3 in result")
	}
}

func TestFilterElementsByTags_FilterAndExclude(t *testing.T) {
	elements := map[string]*Element{
		"elem1": {Kind: "system", Title: "Elem1", Tags: []string{"backend", "stable"}},
		"elem2": {Kind: "system", Title: "Elem2", Tags: []string{"backend", "experimental"}},
		"elem3": {Kind: "system", Title: "Elem3", Tags: []string{"frontend"}},
	}

	// Include only backend elements, exclude experimental ones
	result := FilterElementsByTags(elements, []string{"backend"}, []string{"experimental"})
	if len(result) != 1 {
		t.Errorf("expected 1 element, got %d", len(result))
	}
	if _, ok := result["elem1"]; !ok {
		t.Errorf("expected elem1 in result")
	}
}

func TestFilterElementsByTags_NoMatches(t *testing.T) {
	elements := map[string]*Element{
		"elem1": {Kind: "system", Title: "Elem1", Tags: []string{"tag1"}},
		"elem2": {Kind: "system", Title: "Elem2", Tags: []string{"tag2"}},
	}

	result := FilterElementsByTags(elements, []string{"nonexistent"}, nil)
	if len(result) != 0 {
		t.Errorf("expected 0 elements, got %d", len(result))
	}
}

func TestFilterElementsByTags_EmptyElements(t *testing.T) {
	elements := make(map[string]*Element)
	result := FilterElementsByTags(elements, []string{"tag1"}, nil)
	if len(result) != 0 {
		t.Errorf("expected 0 elements, got %d", len(result))
	}
}
