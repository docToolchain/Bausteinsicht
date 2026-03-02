package sync

import (
	"fmt"
	"strings"

	"github.com/docToolchain/Bauteinsicht/internal/drawio"
	"github.com/docToolchain/Bauteinsicht/internal/model"
)

// ChangeType classifies a change.
type ChangeType int

const (
	Added    ChangeType = iota
	Modified            // nolint:deadcode
	Deleted
)

// ElementChange represents a change to a single element.
type ElementChange struct {
	ID       string
	Type     ChangeType
	Field    string // "title", "description", "technology", "" for add/delete
	OldValue string
	NewValue string
}

// RelationshipChange represents a change to a relationship.
type RelationshipChange struct {
	From     string
	To       string
	Index    int    // relationship array index for disambiguation
	Type     ChangeType
	Field    string // "label", "" for add/delete
	OldValue string
	NewValue string
}

// ChangeSet contains all detected changes from both sides.
type ChangeSet struct {
	ModelElementChanges       []ElementChange
	ModelRelationshipChanges  []RelationshipChange
	DrawioElementChanges      []ElementChange
	DrawioRelationshipChanges []RelationshipChange
	Conflicts                 []Conflict
}

// drawioElemSnapshot holds extracted data from a draw.io element.
type drawioElemSnapshot struct {
	title       string
	technology  string
	description string
	kind        string
}

// relKey returns a canonical key for a relationship.
// The index disambiguates multiple relationships between the same pair.
func relKey(from, to string, index int) string {
	return fmt.Sprintf("%s:%s:%d", from, to, index)
}

// computeVisibleElements returns the set of element IDs that should be visible
// across all views. If the model has no views, returns nil (meaning ALL elements
// are visible on the single page).
func computeVisibleElements(m *model.BausteinsichtModel) map[string]bool {
	if len(m.Views) == 0 {
		return nil // all elements visible
	}
	visible := make(map[string]bool)
	for _, view := range m.Views {
		v := view
		resolved, err := model.ResolveView(m, &v)
		if err != nil {
			continue
		}
		for _, id := range resolved {
			visible[id] = true
		}
		// The scope element itself is also visible (rendered as boundary).
		if view.Scope != "" {
			visible[view.Scope] = true
		}
	}
	return visible
}

// DetectChanges performs a three-way diff between the model, draw.io document,
// and the last known sync state.
func DetectChanges(m *model.BausteinsichtModel, doc *drawio.Document, lastState *SyncState) *ChangeSet {
	cs := &ChangeSet{}

	flatModel := model.FlattenElements(m)
	drawioElems := extractDrawioElements(doc)
	visibleElems := computeVisibleElements(m)
	detectElementChanges(cs, flatModel, drawioElems, lastState, visibleElems)

	modelRels := buildModelRelMap(m)
	drawioRels := extractDrawioRelationships(doc)
	detectRelationshipChanges(cs, modelRels, drawioRels, lastState)

	return cs
}

// extractDrawioElements gathers element data from all pages in the document.
func extractDrawioElements(doc *drawio.Document) map[string]drawioElemSnapshot {
	result := make(map[string]drawioElemSnapshot)
	for _, page := range doc.Pages() {
		for _, obj := range page.FindAllElements() {
			id := obj.SelectAttrValue("bausteinsicht_id", "")
			if id == "" {
				continue
			}
			label := obj.SelectAttrValue("label", "")
			title, technology, labelDesc := drawio.ParseLabel(label)
			tooltipDesc := obj.SelectAttrValue("tooltip", "")
			description := tooltipDesc
			if description == "" {
				description = labelDesc
			}
			result[id] = drawioElemSnapshot{
				title:       title,
				technology:  technology,
				description: description,
				kind:        obj.SelectAttrValue("bausteinsicht_kind", ""),
			}
		}
	}
	return result
}

// stripScopedPrefix removes the view prefix from a scoped cell ID.
// Scoped cell IDs have the format "viewID--elemID" where "--" is the separator.
// If the ID does not contain "--", it is returned unchanged (legacy documents).
func stripScopedPrefix(cellID string) string {
	if idx := strings.Index(cellID, "--"); idx >= 0 {
		return cellID[idx+2:]
	}
	return cellID
}

// buildCellIDToElemID builds a mapping from draw.io cell IDs to bausteinsicht
// element IDs. When views are used, cell IDs are scoped (e.g., "context--customer")
// while element IDs are un-scoped (e.g., "customer").
func buildCellIDToElemID(doc *drawio.Document) map[string]string {
	m := make(map[string]string)
	for _, page := range doc.Pages() {
		for _, obj := range page.FindAllElements() {
			elemID := obj.SelectAttrValue("bausteinsicht_id", "")
			cellID := obj.SelectAttrValue("id", "")
			if elemID != "" && cellID != "" {
				m[cellID] = elemID
			}
		}
	}
	return m
}

// extractDrawioRelationships gathers connector data from all pages.
// Connector source/target cell IDs are resolved to element IDs using the
// bausteinsicht_id attributes of referenced elements.
// Lifted connectors (where an endpoint was lifted to a parent because the
// original target is not visible on a view) are excluded to avoid phantom
// reverse changes.
func extractDrawioRelationships(doc *drawio.Document) map[string]RelationshipState {
	cellToElem := buildCellIDToElemID(doc)
	result := make(map[string]RelationshipState)
	for _, page := range doc.Pages() {
		for _, cell := range page.FindAllConnectors() {
			fromCell := cell.SelectAttrValue("source", "")
			toCell := cell.SelectAttrValue("target", "")
			if fromCell == "" || toCell == "" {
				continue
			}
			// Resolve scoped cell IDs to element IDs.
			// Fall back to stripping the view prefix from scoped cell IDs
			// (e.g., "components--onlineshop.db" → "onlineshop.db") when
			// the element was deleted and is no longer in cellToElem (#166).
			// For legacy (non-view) documents the raw cell ID is used as-is.
			from := fromCell
			if elemID, ok := cellToElem[fromCell]; ok {
				from = elemID
			} else {
				from = stripScopedPrefix(fromCell)
			}
			to := toCell
			if elemID, ok := cellToElem[toCell]; ok {
				to = elemID
			} else {
				to = stripScopedPrefix(toCell)
			}
			// Extract the relationship index from the connector ID.
			cellID := cell.SelectAttrValue("id", "")
			index := parseConnectorIndex(cellID)
			key := relKey(from, to, index)
			if _, exists := result[key]; !exists {
				result[key] = RelationshipState{
					From:  from,
					To:    to,
					Index: index,
					Label: cell.SelectAttrValue("value", ""),
				}
			}
		}
	}
	return result
}

// parseConnectorIndex extracts the index from a connector ID of the form
// "rel-<from>-<to>-<index>". Returns 0 if the ID does not contain an index
// (backward compatibility with old connector IDs "rel-<from>-<to>").
func parseConnectorIndex(id string) int {
	if !strings.HasPrefix(id, "rel-") {
		return 0
	}
	// The index is the last segment after the last '-'.
	lastDash := strings.LastIndex(id, "-")
	if lastDash < 0 {
		return 0
	}
	indexStr := id[lastDash+1:]
	var index int
	if _, err := fmt.Sscanf(indexStr, "%d", &index); err != nil {
		return 0
	}
	return index
}

// buildModelRelMap converts model relationships to a map keyed by relKey.
func buildModelRelMap(m *model.BausteinsichtModel) map[string]RelationshipState {
	modelRels := make(map[string]RelationshipState, len(m.Relationships))
	for i, r := range m.Relationships {
		modelRels[relKey(r.From, r.To, i)] = RelationshipState{
			From:  r.From,
			To:    r.To,
			Index: i,
			Label: r.Label,
			Kind:  r.Kind,
		}
	}
	return modelRels
}

// detectElementChanges performs three-way comparison for elements.
// visibleElems is the set of element IDs visible across all views. If nil,
// all elements are considered visible (no views defined).
func detectElementChanges(
	cs *ChangeSet,
	flatModel map[string]*model.Element,
	drawioElems map[string]drawioElemSnapshot,
	lastState *SyncState,
	visibleElems map[string]bool,
) {
	allIDs := unionElementIDs(flatModel, drawioElems, lastState)

	for id := range allIDs {
		me, inModel := flatModel[id]
		de, inDrawio := drawioElems[id]
		lastElem, inLast := lastState.Elements[id]

		// Model side changes
		switch {
		case inModel && !inLast:
			cs.ModelElementChanges = append(cs.ModelElementChanges, ElementChange{ID: id, Type: Added})
		case !inModel && inLast:
			cs.ModelElementChanges = append(cs.ModelElementChanges, ElementChange{ID: id, Type: Deleted})
		case inModel && inLast:
			appendIfChanged(id, "title", lastElem.Title, me.Title, &cs.ModelElementChanges)
			appendIfChanged(id, "description", lastElem.Description, me.Description, &cs.ModelElementChanges)
			appendIfChanged(id, "technology", lastElem.Technology, me.Technology, &cs.ModelElementChanges)
			appendIfChanged(id, "kind", lastElem.Kind, me.Kind, &cs.ModelElementChanges)
		}

		// Draw.io side changes
		switch {
		case inDrawio && !inLast:
			cs.DrawioElementChanges = append(cs.DrawioElementChanges, ElementChange{ID: id, Type: Added})
		case !inDrawio && inLast:
			// Only treat as deleted if the element should be visible on at least one
			// view page. Elements not in any view's resolved set are simply filtered
			// out and their absence from draw.io is expected, not a deletion. (#108, #118)
			if visibleElems == nil || visibleElems[id] {
				cs.DrawioElementChanges = append(cs.DrawioElementChanges, ElementChange{ID: id, Type: Deleted})
			}
		case inDrawio && inLast:
			appendIfChanged(id, "title", lastElem.Title, de.title, &cs.DrawioElementChanges)
			appendIfChanged(id, "description", lastElem.Description, de.description, &cs.DrawioElementChanges)
			appendIfChanged(id, "technology", lastElem.Technology, de.technology, &cs.DrawioElementChanges)
			// Note: kind is not compared on the draw.io side because scope
			// boundary elements have a derived kind (e.g. "system_boundary")
			// that legitimately differs from the model kind ("system").
		}

		// Conflicts: both sides modified the same field
		if inModel && inDrawio && inLast {
			checkElemConflict(cs, id, "title", lastElem.Title, me.Title, de.title)
			checkElemConflict(cs, id, "description", lastElem.Description, me.Description, de.description)
			checkElemConflict(cs, id, "technology", lastElem.Technology, me.Technology, de.technology)
			// Note: kind conflicts are not checked because kind is
			// model-authoritative and draw.io boundary kinds are derived.
		}
	}
}

// unionElementIDs returns the union of IDs across all three sources.
func unionElementIDs(
	flatModel map[string]*model.Element,
	drawioElems map[string]drawioElemSnapshot,
	lastState *SyncState,
) map[string]struct{} {
	all := make(map[string]struct{})
	for id := range flatModel {
		all[id] = struct{}{}
	}
	for id := range lastState.Elements {
		all[id] = struct{}{}
	}
	for id := range drawioElems {
		all[id] = struct{}{}
	}
	return all
}

// appendIfChanged adds a Modified ElementChange if newValue differs from lastValue.
func appendIfChanged(id, field, lastValue, newValue string, changes *[]ElementChange) {
	if newValue != lastValue {
		*changes = append(*changes, ElementChange{
			ID:       id,
			Type:     Modified,
			Field:    field,
			OldValue: lastValue,
			NewValue: newValue,
		})
	}
}

// checkElemConflict adds a Conflict when both model and draw.io changed the same field.
func checkElemConflict(cs *ChangeSet, id, field, last, modelVal, drawioVal string) {
	if modelVal != last && drawioVal != last {
		cs.Conflicts = append(cs.Conflicts, Conflict{
			ElementID:     id,
			Field:         field,
			ModelValue:    modelVal,
			DrawioValue:   drawioVal,
			LastSyncValue: last,
		})
	}
}

// detectRelationshipChanges performs three-way comparison for relationships.
func detectRelationshipChanges(
	cs *ChangeSet,
	modelRels map[string]RelationshipState,
	drawioRels map[string]RelationshipState,
	lastState *SyncState,
) {
	lastRels := make(map[string]RelationshipState, len(lastState.Relationships))
	for _, r := range lastState.Relationships {
		lastRels[relKey(r.From, r.To, r.Index)] = r
	}

	allKeys := unionRelKeys(modelRels, drawioRels, lastRels)

	for k := range allKeys {
		mr, inModel := modelRels[k]
		dr, inDrawio := drawioRels[k]
		lr, inLast := lastRels[k]

		from, to, index := resolveRelFromTo(mr, lr, dr)

		// Model side
		switch {
		case inModel && !inLast:
			cs.ModelRelationshipChanges = append(cs.ModelRelationshipChanges, RelationshipChange{
				From: from, To: to, Index: index, Type: Added, NewValue: mr.Label,
			})
		case !inModel && inLast:
			cs.ModelRelationshipChanges = append(cs.ModelRelationshipChanges, RelationshipChange{
				From: from, To: to, Index: index, Type: Deleted,
			})
		case inModel && inLast && mr.Label != lr.Label:
			cs.ModelRelationshipChanges = append(cs.ModelRelationshipChanges, RelationshipChange{
				From: from, To: to, Index: index, Type: Modified, Field: "label",
				OldValue: lr.Label, NewValue: mr.Label,
			})
		}

		// Draw.io side
		switch {
		case inDrawio && !inLast:
			// Skip lifted connectors: when a view lifts a relationship
			// endpoint to a parent (e.g., A→B.child becomes A→B),
			// the lifted connector should not be treated as a new relationship.
			if isLiftedRelationship(from, to, modelRels) {
				continue
			}
			cs.DrawioRelationshipChanges = append(cs.DrawioRelationshipChanges, RelationshipChange{
				From: from, To: to, Index: index, Type: Added,
			})
		case !inDrawio && inLast:
			cs.DrawioRelationshipChanges = append(cs.DrawioRelationshipChanges, RelationshipChange{
				From: from, To: to, Index: index, Type: Deleted,
			})
		case inDrawio && inLast && dr.Label != lr.Label:
			cs.DrawioRelationshipChanges = append(cs.DrawioRelationshipChanges, RelationshipChange{
				From: from, To: to, Index: index, Type: Modified, Field: "label",
				OldValue: lr.Label, NewValue: dr.Label,
			})
		}
	}
}

// unionRelKeys returns the union of relationship keys from all three sources.
func unionRelKeys(
	modelRels, drawioRels, lastRels map[string]RelationshipState,
) map[string]struct{} {
	all := make(map[string]struct{})
	for k := range modelRels {
		all[k] = struct{}{}
	}
	for k := range lastRels {
		all[k] = struct{}{}
	}
	for k := range drawioRels {
		all[k] = struct{}{}
	}
	return all
}

// isLiftedRelationship returns true if the relationship from→to is a "lifted"
// version of an existing model relationship. A relationship is lifted when a
// view shows a connector between parent elements because the original endpoint
// is not visible. For example, model has A→B.child but the view only shows A
// and B, so the connector is lifted to A→B.
func isLiftedRelationship(from, to string, modelRels map[string]RelationshipState) bool {
	for _, mr := range modelRels {
		// Same from, model to is more specific (to is ancestor of mr.To)
		if mr.From == from && mr.To != to && strings.HasPrefix(mr.To, to+".") {
			return true
		}
		// Same to, model from is more specific
		if mr.To == to && mr.From != from && strings.HasPrefix(mr.From, from+".") {
			return true
		}
		// Both endpoints lifted
		if mr.From != from && mr.To != to &&
			strings.HasPrefix(mr.From, from+".") && strings.HasPrefix(mr.To, to+".") {
			return true
		}
	}
	return false
}

// resolveRelFromTo returns the from/to/index from the first non-empty source.
func resolveRelFromTo(mr, lr, dr RelationshipState) (from, to string, index int) {
	from, to, index = mr.From, mr.To, mr.Index
	if from == "" {
		from, to, index = lr.From, lr.To, lr.Index
	}
	if from == "" {
		from, to, index = dr.From, dr.To, dr.Index
	}
	return from, to, index
}
