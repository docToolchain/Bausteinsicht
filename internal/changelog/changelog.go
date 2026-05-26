package changelog

import (
	"github.com/docToolchain/Bausteinsicht/internal/diff"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// Generate creates a changelog by comparing two model versions
func Generate(from, to *model.BausteinsichtModel, fromRef, toRef Reference) *Changelog {
	// Convert models to snapshots for diff comparison
	fromSnap := modelToSnapshot(from)
	toSnap := modelToSnapshot(to)

	// Use diff package to compute changes
	result := diff.Compare(fromSnap, toSnap)

	// Organize changes by type
	return &Changelog{
		From: fromRef,
		To:   toRef,
		Elements: ElementChanges{
			Added:   filterElementsByType(result.Elements, diff.ChangeAdded),
			Removed: filterElementsByType(result.Elements, diff.ChangeRemoved),
			Changed: filterElementsByType(result.Elements, diff.ChangeChanged),
		},
		Relationships: RelationshipChanges{
			Added:   filterRelationshipsByType(result.Relationships, diff.ChangeAdded),
			Removed: filterRelationshipsByType(result.Relationships, diff.ChangeRemoved),
		},
	}
}

// modelToSnapshot converts a BausteinsichtModel to a ModelSnapshot for diff computation
func modelToSnapshot(m *model.BausteinsichtModel) *model.ModelSnapshot {
	if m == nil {
		return &model.ModelSnapshot{
			Elements:      make(map[string]model.Element),
			Relationships: []model.Relationship{},
		}
	}

	// Flatten nested elements into a single-level map using dot notation
	elements := flattenElements(m.Model, "")

	return &model.ModelSnapshot{
		Elements:      elements,
		Relationships: m.Relationships,
	}
}

// flattenElements recursively flattens nested element maps into a single-level map
// with dot-separated keys (e.g., "system.backend.api")
func flattenElements(elems map[string]model.Element, prefix string) map[string]model.Element {
	result := make(map[string]model.Element)

	for key, elem := range elems {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		// Add this element
		result[fullKey] = elem

		// Recursively flatten child elements if they exist
		if len(elem.Children) > 0 {
			children := flattenElements(elem.Children, fullKey)
			for k, v := range children {
				result[k] = v
			}
		}
	}

	return result
}

// filterElementsByType returns only elements with the specified change type
func filterElementsByType(changes []diff.ElementChange, changeType diff.ChangeType) []diff.ElementChange {
	var result []diff.ElementChange
	for _, c := range changes {
		if c.Type == changeType {
			result = append(result, c)
		}
	}
	return result
}

// filterRelationshipsByType returns only relationships with the specified change type
func filterRelationshipsByType(changes []diff.RelationshipChange, changeType diff.ChangeType) []diff.RelationshipChange {
	var result []diff.RelationshipChange
	for _, c := range changes {
		if c.Type == changeType {
			result = append(result, c)
		}
	}
	return result
}
