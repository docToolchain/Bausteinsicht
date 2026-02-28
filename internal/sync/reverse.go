package sync

import (
	"fmt"
	"strings"

	"github.com/docToolchain/Bauteinsicht/internal/model"
)

// ReverseResult summarizes the changes applied back to the model.
type ReverseResult struct {
	ElementsCreated      int
	ElementsUpdated      int
	ElementsDeleted      int
	RelationshipsCreated int
	RelationshipsUpdated int
	RelationshipsDeleted int
	Warnings             []string
}

// ApplyReverse applies draw.io-side changes back to the model.
func ApplyReverse(changes *ChangeSet, m *model.BausteinsichtModel) *ReverseResult {
	result := &ReverseResult{}

	for _, ch := range changes.DrawioElementChanges {
		applyElementChange(ch, m, result)
	}

	for _, ch := range changes.DrawioRelationshipChanges {
		applyRelationshipChange(ch, m, result)
	}

	return result
}

func applyElementChange(ch ElementChange, m *model.BausteinsichtModel, result *ReverseResult) {
	switch ch.Type {
	case Modified:
		err := modifyElement(m, ch.ID, func(e *model.Element) {
			switch ch.Field {
			case "title":
				e.Title = ch.NewValue
			case "description":
				e.Description = ch.NewValue
			case "technology":
				e.Technology = ch.NewValue
			}
		})
		if err != nil {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Element %q not found in model: %v", ch.ID, err))
			return
		}
		result.ElementsUpdated++

	case Deleted:
		err := deleteElement(m, ch.ID)
		if err != nil {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Element %q could not be deleted: %v", ch.ID, err))
			return
		}
		result.ElementsDeleted++
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Element %q was deleted in draw.io and removed from model", ch.ID))

	case Added:
		result.Warnings = append(result.Warnings,
			"New element detected in draw.io (no bausteinsicht_id). Please add it to the model manually.")
	}
}

func applyRelationshipChange(ch RelationshipChange, m *model.BausteinsichtModel, result *ReverseResult) {
	switch ch.Type {
	case Modified:
		updated := false
		for i, r := range m.Relationships {
			if r.From == ch.From && r.To == ch.To {
				if ch.Field == "label" {
					m.Relationships[i].Label = ch.NewValue
				}
				updated = true
				break
			}
		}
		if updated {
			result.RelationshipsUpdated++
		} else {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Relationship %q->%q not found in model", ch.From, ch.To))
		}

	case Deleted:
		before := len(m.Relationships)
		m.Relationships = filterRelationships(m.Relationships, ch.From, ch.To)
		if len(m.Relationships) < before {
			result.RelationshipsDeleted++
		}

	case Added:
		m.Relationships = append(m.Relationships, model.Relationship{
			From:  ch.From,
			To:    ch.To,
			Label: ch.NewValue,
		})
		result.RelationshipsCreated++
	}
}

// filterRelationships returns all relationships except those matching from/to.
func filterRelationships(rels []model.Relationship, from, to string) []model.Relationship {
	result := make([]model.Relationship, 0, len(rels))
	for _, r := range rels {
		if r.From != from || r.To != to {
			result = append(result, r)
		}
	}
	return result
}

// modifyElement finds an element by dot-notation ID and applies fn to it.
func modifyElement(m *model.BausteinsichtModel, id string, fn func(*model.Element)) error {
	parts := strings.Split(id, ".")
	return modifyInMap(m.Model, parts, id, fn)
}

// modifyInMap recursively walks the map to find and modify the target element.
func modifyInMap(elems map[string]model.Element, parts []string, fullID string, fn func(*model.Element)) error {
	if len(parts) == 0 {
		return fmt.Errorf("empty path")
	}
	key := parts[0]
	elem, ok := elems[key]
	if !ok {
		return fmt.Errorf("element %q not found", fullID)
	}
	if len(parts) == 1 {
		fn(&elem)
		elems[key] = elem
		return nil
	}
	if elem.Children == nil {
		return fmt.Errorf("element %q not found: no children at this level", fullID)
	}
	if err := modifyInMap(elem.Children, parts[1:], fullID, fn); err != nil {
		return err
	}
	elems[key] = elem
	return nil
}

// deleteElement removes an element by dot-notation ID from the model hierarchy.
func deleteElement(m *model.BausteinsichtModel, id string) error {
	parts := strings.Split(id, ".")
	return deleteFromMap(m.Model, parts, id)
}

// deleteFromMap recursively walks to find the parent map and deletes the element.
func deleteFromMap(elems map[string]model.Element, parts []string, fullID string) error {
	if len(parts) == 0 {
		return fmt.Errorf("empty path")
	}
	key := parts[0]
	if len(parts) == 1 {
		if _, ok := elems[key]; !ok {
			return fmt.Errorf("element %q not found", fullID)
		}
		delete(elems, key)
		return nil
	}
	elem, ok := elems[key]
	if !ok {
		return fmt.Errorf("element %q not found", fullID)
	}
	if elem.Children == nil {
		return fmt.Errorf("element %q not found: no children at this level", fullID)
	}
	if err := deleteFromMap(elem.Children, parts[1:], fullID); err != nil {
		return err
	}
	elems[key] = elem
	return nil
}
