package sync

import (
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
func relKey(from, to string) string {
	return from + ":" + to
}

// DetectChanges performs a three-way diff between the model, draw.io document,
// and the last known sync state.
func DetectChanges(m *model.BausteinsichtModel, doc *drawio.Document, lastState *SyncState) *ChangeSet {
	cs := &ChangeSet{}

	flatModel := model.FlattenElements(m)
	drawioElems := extractDrawioElements(doc)
	detectElementChanges(cs, flatModel, drawioElems, lastState)

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
			title, technology := drawio.ParseLabel(label)
			result[id] = drawioElemSnapshot{
				title:       title,
				technology:  technology,
				description: obj.SelectAttrValue("tooltip", ""),
				kind:        obj.SelectAttrValue("bausteinsicht_kind", ""),
			}
		}
	}
	return result
}

// extractDrawioRelationships gathers connector data from all pages.
func extractDrawioRelationships(doc *drawio.Document) map[string]RelationshipState {
	result := make(map[string]RelationshipState)
	for _, page := range doc.Pages() {
		for _, cell := range page.FindAllConnectors() {
			from := cell.SelectAttrValue("source", "")
			to := cell.SelectAttrValue("target", "")
			if from == "" || to == "" {
				continue
			}
			result[relKey(from, to)] = RelationshipState{
				From:  from,
				To:    to,
				Label: cell.SelectAttrValue("value", ""),
			}
		}
	}
	return result
}

// buildModelRelMap converts model relationships to a map keyed by relKey.
func buildModelRelMap(m *model.BausteinsichtModel) map[string]RelationshipState {
	modelRels := make(map[string]RelationshipState, len(m.Relationships))
	for _, r := range m.Relationships {
		modelRels[relKey(r.From, r.To)] = RelationshipState{
			From:  r.From,
			To:    r.To,
			Label: r.Label,
			Kind:  r.Kind,
		}
	}
	return modelRels
}

// detectElementChanges performs three-way comparison for elements.
func detectElementChanges(
	cs *ChangeSet,
	flatModel map[string]*model.Element,
	drawioElems map[string]drawioElemSnapshot,
	lastState *SyncState,
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
		}

		// Draw.io side changes
		switch {
		case inDrawio && !inLast:
			cs.DrawioElementChanges = append(cs.DrawioElementChanges, ElementChange{ID: id, Type: Added})
		case !inDrawio && inLast:
			cs.DrawioElementChanges = append(cs.DrawioElementChanges, ElementChange{ID: id, Type: Deleted})
		case inDrawio && inLast:
			appendIfChanged(id, "title", lastElem.Title, de.title, &cs.DrawioElementChanges)
			appendIfChanged(id, "description", lastElem.Description, de.description, &cs.DrawioElementChanges)
			appendIfChanged(id, "technology", lastElem.Technology, de.technology, &cs.DrawioElementChanges)
		}

		// Conflicts: both sides modified the same field
		if inModel && inDrawio && inLast {
			checkElemConflict(cs, id, "title", lastElem.Title, me.Title, de.title)
			checkElemConflict(cs, id, "description", lastElem.Description, me.Description, de.description)
			checkElemConflict(cs, id, "technology", lastElem.Technology, me.Technology, de.technology)
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
		lastRels[relKey(r.From, r.To)] = r
	}

	allKeys := unionRelKeys(modelRels, drawioRels, lastRels)

	for k := range allKeys {
		mr, inModel := modelRels[k]
		dr, inDrawio := drawioRels[k]
		lr, inLast := lastRels[k]

		from, to := resolveRelFromTo(mr, lr, dr)

		// Model side
		switch {
		case inModel && !inLast:
			cs.ModelRelationshipChanges = append(cs.ModelRelationshipChanges, RelationshipChange{
				From: from, To: to, Type: Added,
			})
		case !inModel && inLast:
			cs.ModelRelationshipChanges = append(cs.ModelRelationshipChanges, RelationshipChange{
				From: from, To: to, Type: Deleted,
			})
		case inModel && inLast && mr.Label != lr.Label:
			cs.ModelRelationshipChanges = append(cs.ModelRelationshipChanges, RelationshipChange{
				From: from, To: to, Type: Modified, Field: "label",
				OldValue: lr.Label, NewValue: mr.Label,
			})
		}

		// Draw.io side
		switch {
		case inDrawio && !inLast:
			cs.DrawioRelationshipChanges = append(cs.DrawioRelationshipChanges, RelationshipChange{
				From: from, To: to, Type: Added,
			})
		case !inDrawio && inLast:
			cs.DrawioRelationshipChanges = append(cs.DrawioRelationshipChanges, RelationshipChange{
				From: from, To: to, Type: Deleted,
			})
		case inDrawio && inLast && dr.Label != lr.Label:
			cs.DrawioRelationshipChanges = append(cs.DrawioRelationshipChanges, RelationshipChange{
				From: from, To: to, Type: Modified, Field: "label",
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

// resolveRelFromTo returns the from/to pair from the first non-empty source.
func resolveRelFromTo(mr, lr, dr RelationshipState) (from, to string) {
	from, to = mr.From, mr.To
	if from == "" {
		from, to = lr.From, lr.To
	}
	if from == "" {
		from, to = dr.From, dr.To
	}
	return from, to
}
