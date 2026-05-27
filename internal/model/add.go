package model

import (
	"fmt"
	"sort"
)

// AddView creates a new view or merges fields into an existing view.
// For existing views:
// - --include elements are merged (deduplicated)
// - --title, --scope, --description are updated (if specified)
// Returns an error if:
// - View key is empty
// - Scope element doesn't exist (if specified)
// - Include elements don't exist (if specified)
func (m *BausteinsichtModel) AddView(key string, view View) error {
	if key == "" {
		return fmt.Errorf("view key must not be empty")
	}

	// Initialize views map if needed
	if m.Views == nil {
		m.Views = make(map[string]View)
	}

	// Validate scope exists (if specified)
	if view.Scope != "" {
		if _, err := Resolve(m, view.Scope); err != nil {
			return fmt.Errorf("scope %q not found: %w", view.Scope, err)
		}
	}

	// Validate all include elements exist
	for _, include := range view.Include {
		// Skip wildcard patterns — they're validated at render time
		if include == "" || (len(include) > 0 && include[len(include)-1] == '*') {
			continue
		}
		if _, err := Resolve(m, include); err != nil {
			return fmt.Errorf("include %q not found: %w", include, err)
		}
	}

	// Check if view already exists
	existingView, exists := m.Views[key]
	if exists {
		// Merge: keep existing fields, override with new values, merge include lists
		if view.Title != "" {
			existingView.Title = view.Title
		}
		if view.Scope != "" {
			existingView.Scope = view.Scope
		}
		if view.Description != "" {
			existingView.Description = view.Description
		}
		if view.Layout != "" {
			existingView.Layout = view.Layout
		}
		// Merge include lists (deduplicate, sort for deterministic output)
		if len(view.Include) > 0 {
			includedSet := make(map[string]bool)
			for _, elem := range existingView.Include {
				includedSet[elem] = true
			}
			for _, elem := range view.Include {
				includedSet[elem] = true
			}
			merged := make([]string, 0, len(includedSet))
			for elem := range includedSet {
				merged = append(merged, elem)
			}
			sort.Strings(merged)
			existingView.Include = merged
		}
		m.Views[key] = existingView
	} else {
		// New view: use as-is
		m.Views[key] = view
	}

	return nil
}

// AddSpecificationElement adds an element kind to the specification.
// Returns an error if the element kind already exists.
func (m *BausteinsichtModel) AddSpecificationElement(key string, kind ElementKind) error {
	if key == "" {
		return fmt.Errorf("element key must not be empty")
	}

	if kind.Notation == "" {
		return fmt.Errorf("notation must not be empty")
	}

	// Initialize elements map if needed
	if m.Specification.Elements == nil {
		m.Specification.Elements = make(map[string]ElementKind)
	}

	// Check for duplicate
	if _, exists := m.Specification.Elements[key]; exists {
		return fmt.Errorf("element kind %q already exists in specification", key)
	}

	m.Specification.Elements[key] = kind

	return nil
}

// AddSpecificationRelationship adds a relationship kind to the specification.
// Returns an error if the relationship kind already exists.
func (m *BausteinsichtModel) AddSpecificationRelationship(key string, kind RelationshipKind) error {
	if key == "" {
		return fmt.Errorf("relationship key must not be empty")
	}

	if kind.Notation == "" {
		return fmt.Errorf("notation must not be empty")
	}

	// Check for duplicate
	if _, exists := m.Specification.Relationships[key]; exists {
		return fmt.Errorf("relationship kind %q already exists in specification", key)
	}

	// Initialize relationships map if needed
	if m.Specification.Relationships == nil {
		m.Specification.Relationships = make(map[string]RelationshipKind)
	}

	// Add relationship kind
	m.Specification.Relationships[key] = kind

	return nil
}
