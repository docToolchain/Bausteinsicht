package sync

import (
	"testing"

	"github.com/docToolchain/Bauteinsicht/internal/drawio"
	"github.com/docToolchain/Bauteinsicht/internal/model"
)

// helpers

func emptyState() *SyncState {
	return &SyncState{
		Elements:      make(map[string]ElementState),
		Relationships: []RelationshipState{},
	}
}

func stateWithElem(id, title, description, technology string) *SyncState {
	s := emptyState()
	s.Elements[id] = ElementState{Title: title, Description: description, Technology: technology, Kind: "container"}
	return s
}

func simpleModel(id, title, description, technology string) *model.BausteinsichtModel {
	return &model.BausteinsichtModel{
		Model: map[string]model.Element{
			id: {Kind: "container", Title: title, Description: description, Technology: technology},
		},
		Relationships: []model.Relationship{},
	}
}

func emptyModel() *model.BausteinsichtModel {
	return &model.BausteinsichtModel{
		Model:         map[string]model.Element{},
		Relationships: []model.Relationship{},
	}
}

func docWithElem(id, title, technology, description string) *drawio.Document {
	doc := drawio.NewDocument()
	page := doc.AddPage("p1", "Page 1")
	_ = page.CreateElement(drawio.ElementData{
		ID:          id,
		Kind:        "container",
		Title:       title,
		Technology:  technology,
		Description: description,
	}, "")
	return doc
}

func emptyDoc() *drawio.Document {
	doc := drawio.NewDocument()
	doc.AddPage("p1", "Page 1")
	return doc
}

// Tests

func TestDetectChanges_NoChanges(t *testing.T) {
	state := stateWithElem("app", "App", "Desc", "Go")
	m := simpleModel("app", "App", "Desc", "Go")
	doc := docWithElem("app", "App", "Go", "Desc")

	cs := DetectChanges(m, doc, state)

	if len(cs.ModelElementChanges) != 0 {
		t.Errorf("expected no model element changes, got %d", len(cs.ModelElementChanges))
	}
	if len(cs.DrawioElementChanges) != 0 {
		t.Errorf("expected no drawio element changes, got %d", len(cs.DrawioElementChanges))
	}
	if len(cs.Conflicts) != 0 {
		t.Errorf("expected no conflicts, got %d", len(cs.Conflicts))
	}
}

func TestDetectChanges_ModelAddedElement(t *testing.T) {
	state := emptyState()
	m := simpleModel("app", "App", "", "")
	doc := emptyDoc()

	cs := DetectChanges(m, doc, state)

	if len(cs.ModelElementChanges) != 1 {
		t.Fatalf("expected 1 model element change, got %d", len(cs.ModelElementChanges))
	}
	ch := cs.ModelElementChanges[0]
	if ch.ID != "app" || ch.Type != Added {
		t.Errorf("unexpected change: %+v", ch)
	}
}

func TestDetectChanges_ModelModifiedTitle(t *testing.T) {
	state := stateWithElem("app", "OldTitle", "", "")
	m := simpleModel("app", "NewTitle", "", "")
	doc := docWithElem("app", "OldTitle", "", "")

	cs := DetectChanges(m, doc, state)

	found := false
	for _, ch := range cs.ModelElementChanges {
		if ch.ID == "app" && ch.Type == Modified && ch.Field == "title" &&
			ch.OldValue == "OldTitle" && ch.NewValue == "NewTitle" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected model title modification, changes: %+v", cs.ModelElementChanges)
	}
	if len(cs.Conflicts) != 0 {
		t.Errorf("expected no conflicts, got %d", len(cs.Conflicts))
	}
}

func TestDetectChanges_ModelDeletedElement(t *testing.T) {
	state := stateWithElem("app", "App", "", "")
	m := emptyModel()
	doc := docWithElem("app", "App", "", "")

	cs := DetectChanges(m, doc, state)

	found := false
	for _, ch := range cs.ModelElementChanges {
		if ch.ID == "app" && ch.Type == Deleted {
			found = true
		}
	}
	if !found {
		t.Errorf("expected model deletion, changes: %+v", cs.ModelElementChanges)
	}
}

func TestDetectChanges_DrawioModifiedTitle(t *testing.T) {
	state := stateWithElem("app", "OldTitle", "", "")
	m := simpleModel("app", "OldTitle", "", "")
	doc := docWithElem("app", "NewTitle", "", "")

	cs := DetectChanges(m, doc, state)

	found := false
	for _, ch := range cs.DrawioElementChanges {
		if ch.ID == "app" && ch.Type == Modified && ch.Field == "title" &&
			ch.OldValue == "OldTitle" && ch.NewValue == "NewTitle" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected drawio title modification, changes: %+v", cs.DrawioElementChanges)
	}
	if len(cs.Conflicts) != 0 {
		t.Errorf("expected no conflicts, got %d", len(cs.Conflicts))
	}
}

func TestDetectChanges_ConflictBothModifiedSameField(t *testing.T) {
	state := stateWithElem("app", "OldTitle", "", "")
	m := simpleModel("app", "ModelTitle", "", "")
	doc := docWithElem("app", "DrawioTitle", "", "")

	cs := DetectChanges(m, doc, state)

	found := false
	for _, c := range cs.Conflicts {
		if c.ElementID == "app" && c.Field == "title" &&
			c.ModelValue == "ModelTitle" && c.DrawioValue == "DrawioTitle" &&
			c.LastSyncValue == "OldTitle" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected conflict, conflicts: %+v", cs.Conflicts)
	}
}

func TestDetectChanges_NoDuplicateConflictWhenDifferentFields(t *testing.T) {
	state := stateWithElem("app", "OldTitle", "OldDesc", "")
	// model changes title, drawio changes description → different fields, no conflict
	m := simpleModel("app", "NewTitle", "OldDesc", "")
	doc := docWithElem("app", "OldTitle", "", "NewDesc")

	cs := DetectChanges(m, doc, state)

	if len(cs.Conflicts) != 0 {
		t.Errorf("expected no conflicts for different-field changes, got: %+v", cs.Conflicts)
	}
	modelChanged := false
	for _, ch := range cs.ModelElementChanges {
		if ch.ID == "app" && ch.Field == "title" {
			modelChanged = true
		}
	}
	drawioChanged := false
	for _, ch := range cs.DrawioElementChanges {
		if ch.ID == "app" && ch.Field == "description" {
			drawioChanged = true
		}
	}
	if !modelChanged {
		t.Error("expected model title change")
	}
	if !drawioChanged {
		t.Error("expected drawio description change")
	}
}

func TestDetectChanges_RelationshipAdded(t *testing.T) {
	state := emptyState()
	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"a": {Kind: "container", Title: "A"},
			"b": {Kind: "container", Title: "B"},
		},
		Relationships: []model.Relationship{
			{From: "a", To: "b", Label: "calls"},
		},
	}
	doc := emptyDoc()

	cs := DetectChanges(m, doc, state)

	found := false
	for _, ch := range cs.ModelRelationshipChanges {
		if ch.From == "a" && ch.To == "b" && ch.Type == Added {
			found = true
		}
	}
	if !found {
		t.Errorf("expected relationship added, changes: %+v", cs.ModelRelationshipChanges)
	}
}

func TestDetectChanges_RelationshipModifiedLabel(t *testing.T) {
	state := emptyState()
	state.Relationships = []RelationshipState{
		{From: "a", To: "b", Label: "old"},
	}
	m := &model.BausteinsichtModel{
		Model:         map[string]model.Element{},
		Relationships: []model.Relationship{{From: "a", To: "b", Label: "new"}},
	}
	doc := emptyDoc()

	cs := DetectChanges(m, doc, state)

	found := false
	for _, ch := range cs.ModelRelationshipChanges {
		if ch.From == "a" && ch.To == "b" && ch.Type == Modified &&
			ch.Field == "label" && ch.OldValue == "old" && ch.NewValue == "new" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected label modification, changes: %+v", cs.ModelRelationshipChanges)
	}
}

func TestDetectChanges_RelationshipDeleted(t *testing.T) {
	state := emptyState()
	state.Relationships = []RelationshipState{
		{From: "a", To: "b", Label: "calls"},
	}
	m := &model.BausteinsichtModel{
		Model:         map[string]model.Element{},
		Relationships: []model.Relationship{},
	}
	doc := emptyDoc()

	cs := DetectChanges(m, doc, state)

	found := false
	for _, ch := range cs.ModelRelationshipChanges {
		if ch.From == "a" && ch.To == "b" && ch.Type == Deleted {
			found = true
		}
	}
	if !found {
		t.Errorf("expected relationship deleted, changes: %+v", cs.ModelRelationshipChanges)
	}
}

func TestDetectChanges_DrawioRelationshipAdded(t *testing.T) {
	state := emptyState()
	m := emptyModel()
	doc := drawio.NewDocument()
	page := doc.AddPage("p1", "Page 1")
	page.CreateConnector(drawio.ConnectorData{From: "x", To: "y", Label: "uses"}, "")

	cs := DetectChanges(m, doc, state)

	found := false
	for _, ch := range cs.DrawioRelationshipChanges {
		if ch.From == "x" && ch.To == "y" && ch.Type == Added {
			found = true
		}
	}
	if !found {
		t.Errorf("expected drawio relationship added, changes: %+v", cs.DrawioRelationshipChanges)
	}
}
