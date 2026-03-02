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

	// Detect direction-swap pairs (Deleted a→b + Added b→a) so we can
	// update the relationship in-place and preserve metadata (#185).
	swaps := detectRelSwaps(changes.DrawioRelationshipChanges)

	for _, ch := range changes.DrawioRelationshipChanges {
		if swaps[relSwapKey{ch.From, ch.To, ch.Type}] {
			continue // handled as part of a swap pair
		}
		applyRelationshipChange(ch, m, result)
	}

	// Apply swaps: update direction in-place, preserving kind/label/description.
	for _, sw := range collectSwapPairs(changes.DrawioRelationshipChanges) {
		applyRelSwap(sw.from, sw.to, m, result)
	}

	return result
}

type relSwapKey struct {
	from, to string
	typ      ChangeType
}

type swapPair struct {
	from, to string // new direction (from the Added change)
}

// detectRelSwaps returns a set of (from, to, type) triples that are part of
// a direction-swap pair. A swap is a Deleted(a→b) paired with Added(b→a).
func detectRelSwaps(changes []RelationshipChange) map[relSwapKey]bool {
	deleted := make(map[[2]string]bool)
	added := make(map[[2]string]bool)
	for _, ch := range changes {
		switch ch.Type {
		case Deleted:
			deleted[[2]string{ch.From, ch.To}] = true
		case Added:
			added[[2]string{ch.From, ch.To}] = true
		}
	}
	result := make(map[relSwapKey]bool)
	for pair := range deleted {
		reverse := [2]string{pair[1], pair[0]}
		if added[reverse] {
			result[relSwapKey{pair[0], pair[1], Deleted}] = true
			result[relSwapKey{pair[1], pair[0], Added}] = true
		}
	}
	return result
}

// collectSwapPairs returns the new-direction pairs for detected swaps.
func collectSwapPairs(changes []RelationshipChange) []swapPair {
	swaps := detectRelSwaps(changes)
	var pairs []swapPair
	for key := range swaps {
		if key.typ == Added {
			pairs = append(pairs, swapPair{from: key.from, to: key.to})
		}
	}
	return pairs
}

// applyRelSwap updates a relationship's direction in-place, preserving metadata.
func applyRelSwap(newFrom, newTo string, m *model.BausteinsichtModel, result *ReverseResult) {
	for i, r := range m.Relationships {
		if r.From == newTo && r.To == newFrom {
			m.Relationships[i].From = newFrom
			m.Relationships[i].To = newTo
			result.RelationshipsUpdated++
			return
		}
	}
	// Fallback: original already deleted somehow; create new.
	m.Relationships = append(m.Relationships, model.Relationship{
		From: newFrom,
		To:   newTo,
	})
	result.RelationshipsCreated++
}

func applyElementChange(ch ElementChange, m *model.BausteinsichtModel, result *ReverseResult) {
	switch ch.Type {
	case Modified:
		// Reject empty title updates from draw.io (#150).
		if ch.Field == "title" && strings.TrimSpace(ch.NewValue) == "" {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Element %q: ignoring empty title from draw.io", ch.ID))
			return
		}
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
		// Clean stale references from view include/exclude lists.
		for viewID, v := range m.Views {
			v.Include = removeFromSlice(v.Include, ch.ID)
			v.Exclude = removeFromSlice(v.Exclude, ch.ID)
			m.Views[viewID] = v
		}
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Element %q was deleted in draw.io and removed from model", ch.ID))

	case Added:
		if m.Model == nil {
			m.Model = make(map[string]model.Element)
		}
		m.Model[ch.ID] = model.Element{
			Title: ch.NewValue,
		}
		result.ElementsCreated++
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("New element %q added from draw.io — review and assign a meaningful ID if needed.", ch.ID))
	}
}

func applyRelationshipChange(ch RelationshipChange, m *model.BausteinsichtModel, result *ReverseResult) {
	switch ch.Type {
	case Modified:
		updated := false
		if ch.Index >= 0 && ch.Index < len(m.Relationships) {
			r := m.Relationships[ch.Index]
			if r.From == ch.From && r.To == ch.To {
				if ch.Field == "label" {
					m.Relationships[ch.Index].Label = ch.NewValue
				}
				updated = true
			}
		}
		// Fallback: search by from/to if index does not match.
		if !updated {
			for i, r := range m.Relationships {
				if r.From == ch.From && r.To == ch.To {
					if ch.Field == "label" {
						m.Relationships[i].Label = ch.NewValue
					}
					updated = true
					break
				}
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
		if ch.Index >= 0 && ch.Index < len(m.Relationships) {
			r := m.Relationships[ch.Index]
			if r.From == ch.From && r.To == ch.To {
				m.Relationships = append(m.Relationships[:ch.Index], m.Relationships[ch.Index+1:]...)
			} else {
				// Fallback: filter by from/to.
				m.Relationships = filterRelationships(m.Relationships, ch.From, ch.To)
			}
		} else {
			m.Relationships = filterRelationships(m.Relationships, ch.From, ch.To)
		}
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

// removeFromSlice returns a new slice with all occurrences of val removed.
func removeFromSlice(s []string, val string) []string {
	result := make([]string, 0, len(s))
	for _, v := range s {
		if v != val {
			result = append(result, v)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
