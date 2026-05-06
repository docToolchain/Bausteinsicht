package diff

import (
	"fmt"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// Compare generates a diff between two architecture snapshots (asIs vs toBe)
func Compare(asIs, toBe *model.ModelSnapshot) *DiffResult {
	result := &DiffResult{
		Elements:      []ElementChange{},
		Relationships: []RelationshipChange{},
		Summary:       Summary{},
	}

	if asIs == nil || toBe == nil {
		return result
	}

	compareElements(asIs.Elements, toBe.Elements, result)
	compareRelationships(asIs.Relationships, toBe.Relationships, result)

	calculateSummary(result)

	return result
}

func compareElements(asIsElems, toBeElems map[string]model.Element, result *DiffResult) {
	// Mark as-is elements
	seenAsIs := make(map[string]bool)
	for id, asIsElem := range asIsElems {
		seenAsIs[id] = true

		toBeElem, exists := toBeElems[id]
		if !exists {
			// Element removed
			result.Elements = append(result.Elements, ElementChange{
				ID:     id,
				Type:   ChangeRemoved,
				AsIs:   &asIsElem,
				Reason: "removed from to-be state",
			})
			continue
		}

		// Check if element changed
		if hasElementChanged(&asIsElem, &toBeElem) {
			result.Elements = append(result.Elements, ElementChange{
				ID:     id,
				Type:   ChangeChanged,
				AsIs:   &asIsElem,
				ToBe:   &toBeElem,
				Reason: "element properties changed",
			})
		}
	}

	// Find added elements
	for id, toBeElem := range toBeElems {
		if !seenAsIs[id] {
			result.Elements = append(result.Elements, ElementChange{
				ID:     id,
				Type:   ChangeAdded,
				ToBe:   &toBeElem,
				Reason: "new element in to-be state",
			})
		}
	}
}

func compareRelationships(asIsRels, toBeRels []model.Relationship, result *DiffResult) {
	// Build maps for easier comparison
	asIsMap := relationshipMap(asIsRels)
	toBeMap := relationshipMap(toBeRels)

	// Find removed and changed relationships
	for key, asIsRel := range asIsMap {
		toBeRel, exists := toBeMap[key]
		if !exists {
			// Relationship removed
			result.Relationships = append(result.Relationships, RelationshipChange{
				From:   asIsRel.From,
				To:     asIsRel.To,
				Type:   ChangeRemoved,
				AsIs:   &asIsRel,
			})
			continue
		}

		// Check if changed (e.g., label changed)
		if asIsRel.Label != toBeRel.Label {
			result.Relationships = append(result.Relationships, RelationshipChange{
				From:   asIsRel.From,
				To:     asIsRel.To,
				Type:   ChangeChanged,
				AsIs:   &asIsRel,
				ToBe:   &toBeRel,
			})
		}
	}

	// Find added relationships
	for key, toBeRel := range toBeMap {
		if _, exists := asIsMap[key]; !exists {
			result.Relationships = append(result.Relationships, RelationshipChange{
				From:   toBeRel.From,
				To:     toBeRel.To,
				Type:   ChangeAdded,
				ToBe:   &toBeRel,
			})
		}
	}
}

func relationshipMap(rels []model.Relationship) map[string]model.Relationship {
	m := make(map[string]model.Relationship)
	for _, rel := range rels {
		key := fmt.Sprintf("%s->%s", rel.From, rel.To)
		m[key] = rel
	}
	return m
}

func hasElementChanged(asIs, toBe *model.Element) bool {
	if asIs == nil || toBe == nil {
		return true
	}

	// Compare relevant fields (excluding layout properties)
	return asIs.Title != toBe.Title ||
		asIs.Kind != toBe.Kind ||
		asIs.Technology != toBe.Technology ||
		asIs.Description != toBe.Description ||
		asIs.Status != toBe.Status
}

func calculateSummary(result *DiffResult) {
	for _, change := range result.Elements {
		switch change.Type {
		case ChangeAdded:
			result.Summary.AddedElements++
			result.Summary.TotalAddedElements++
		case ChangeRemoved:
			result.Summary.RemovedElements++
			result.Summary.TotalRemovedElements++
		case ChangeChanged:
			result.Summary.ChangedElements++
		}
	}

	for _, change := range result.Relationships {
		switch change.Type {
		case ChangeAdded:
			result.Summary.AddedRelationships++
		case ChangeRemoved:
			result.Summary.RemovedRelationships++
		}
	}
}
