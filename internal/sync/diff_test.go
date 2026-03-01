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

// docWithScopedConnector creates a document simulating a view-based layout
// where elements have scoped cell IDs (viewID--elemID) but bausteinsicht_id
// attributes contain the un-scoped element ID.
func docWithScopedConnector(viewID, fromElemID, toElemID, label string) *drawio.Document {
	doc := drawio.NewDocument()
	page := doc.AddPage("view-"+viewID, viewID+" View")
	// Create elements with scoped cell IDs (as forward sync does)
	_ = page.CreateElement(drawio.ElementData{
		ID:     fromElemID,
		CellID: viewID + "--" + fromElemID,
		Kind:   "system",
		Title:  fromElemID,
	}, "")
	_ = page.CreateElement(drawio.ElementData{
		ID:     toElemID,
		CellID: viewID + "--" + toElemID,
		Kind:   "system",
		Title:  toElemID,
	}, "")
	// Create connector using scoped cell IDs as source/target (as forward sync does)
	page.CreateConnector(drawio.ConnectorData{
		From:      fromElemID,
		To:        toElemID,
		Label:     label,
		SourceRef: viewID + "--" + fromElemID,
		TargetRef: viewID + "--" + toElemID,
	}, "")
	return doc
}

func TestDetectChanges_ScopedConnectorsMappedToElementIDs(t *testing.T) {
	// State has a relationship with un-scoped element IDs (the correct form).
	state := emptyState()
	state.Relationships = []RelationshipState{
		{From: "customer", To: "webshop", Label: "uses"},
	}

	// Model matches state — no model-side changes.
	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"customer": {Kind: "system", Title: "Customer"},
			"webshop":  {Kind: "system", Title: "Webshop"},
		},
		Relationships: []model.Relationship{
			{From: "customer", To: "webshop", Label: "uses"},
		},
	}

	// draw.io has scoped cell IDs (e.g., context--customer) on connectors.
	doc := docWithScopedConnector("context", "customer", "webshop", "uses")

	cs := DetectChanges(m, doc, state)

	// There should be NO drawio relationship changes — the connector represents
	// the same relationship as in state, just with scoped cell IDs.
	if len(cs.DrawioRelationshipChanges) != 0 {
		t.Errorf("expected no drawio relationship changes when connectors use scoped cell IDs, got %d: %+v",
			len(cs.DrawioRelationshipChanges), cs.DrawioRelationshipChanges)
	}

	// There should be NO model relationship changes either.
	if len(cs.ModelRelationshipChanges) != 0 {
		t.Errorf("expected no model relationship changes, got %d: %+v",
			len(cs.ModelRelationshipChanges), cs.ModelRelationshipChanges)
	}
}

func TestDetectChanges_ScopedConnectorsMultipleViews(t *testing.T) {
	// Same relationship appears on multiple view pages with different scoped IDs.
	// It should be detected as a single relationship, not duplicates.
	state := emptyState()
	state.Relationships = []RelationshipState{
		{From: "customer", To: "webshop", Label: "uses"},
	}

	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"customer": {Kind: "system", Title: "Customer"},
			"webshop":  {Kind: "system", Title: "Webshop"},
		},
		Relationships: []model.Relationship{
			{From: "customer", To: "webshop", Label: "uses"},
		},
	}

	// Build a doc with the same relationship on two view pages.
	doc := drawio.NewDocument()
	for _, viewID := range []string{"context", "containers"} {
		page := doc.AddPage("view-"+viewID, viewID+" View")
		_ = page.CreateElement(drawio.ElementData{
			ID: "customer", CellID: viewID + "--customer", Kind: "system", Title: "Customer",
		}, "")
		_ = page.CreateElement(drawio.ElementData{
			ID: "webshop", CellID: viewID + "--webshop", Kind: "system", Title: "Webshop",
		}, "")
		page.CreateConnector(drawio.ConnectorData{
			From: "customer", To: "webshop", Label: "uses",
			SourceRef: viewID + "--customer", TargetRef: viewID + "--webshop",
		}, "")
	}

	cs := DetectChanges(m, doc, state)

	if len(cs.DrawioRelationshipChanges) != 0 {
		t.Errorf("expected no drawio relationship changes for multi-view scoped connectors, got %d: %+v",
			len(cs.DrawioRelationshipChanges), cs.DrawioRelationshipChanges)
	}
	if len(cs.ModelRelationshipChanges) != 0 {
		t.Errorf("expected no model relationship changes, got %d: %+v",
			len(cs.ModelRelationshipChanges), cs.ModelRelationshipChanges)
	}
}

func TestDetectChanges_LiftedConnectorIgnored(t *testing.T) {
	// The model has customer → shop.frontend. On the context view,
	// the relationship is lifted to customer → shop (because shop.frontend
	// is not on the context view). The lifted connector should NOT
	// be treated as a new drawio relationship.
	state := emptyState()
	state.Relationships = []RelationshipState{
		{From: "customer", To: "shop.frontend", Label: "uses"},
	}

	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"customer": {Kind: "system", Title: "Customer"},
			"shop": {Kind: "system", Title: "Shop", Children: map[string]model.Element{
				"frontend": {Kind: "container", Title: "Frontend"},
			}},
		},
		Relationships: []model.Relationship{
			{From: "customer", To: "shop.frontend", Label: "uses"},
		},
	}

	// Create a doc where the connector uses lifted endpoints.
	doc := drawio.NewDocument()
	page := doc.AddPage("view-ctx", "Context")
	_ = page.CreateElement(drawio.ElementData{
		ID: "customer", CellID: "ctx--customer", Kind: "system", Title: "Customer",
	}, "")
	_ = page.CreateElement(drawio.ElementData{
		ID: "shop", CellID: "ctx--shop", Kind: "system", Title: "Shop",
	}, "")
	// This is a lifted connector: the original relationship is customer → shop.frontend
	// but the connector endpoints were lifted to customer → shop.
	page.CreateConnector(drawio.ConnectorData{
		From: "customer", To: "shop",
		Label:     "uses",
		SourceRef: "ctx--customer",
		TargetRef: "ctx--shop",
	}, "")

	cs := DetectChanges(m, doc, state)

	// The lifted connector (customer → shop) should NOT appear as a new
	// drawio relationship, because it's a visual representation of an
	// existing model relationship (customer → shop.frontend).
	for _, ch := range cs.DrawioRelationshipChanges {
		if ch.From == "customer" && ch.To == "shop" && ch.Type == Added {
			t.Errorf("lifted connector customer→shop should not be detected as a new drawio relationship, got: %+v",
				cs.DrawioRelationshipChanges)
		}
	}
}

func TestDetectChanges_TooltipChangeDetected(t *testing.T) {
	// Sync state has element with description "Old Desc"
	state := stateWithElem("app", "App", "Old Desc", "Go")

	// Model has same description (no model-side change)
	m := simpleModel("app", "App", "Old Desc", "Go")

	// Create draw.io doc where label still has "Old Desc" in description,
	// but the tooltip attribute has been changed to "New Desc" by the user
	doc := docWithElem("app", "App", "Go", "Old Desc")
	// Manually update the tooltip attribute to simulate user editing tooltip in draw.io
	for _, page := range doc.Pages() {
		obj := page.FindElement("app")
		if obj != nil {
			attr := obj.SelectAttr("tooltip")
			if attr != nil {
				attr.Value = "New Desc"
			} else {
				obj.CreateAttr("tooltip", "New Desc")
			}
		}
	}

	cs := DetectChanges(m, doc, state)

	// Should detect a drawio-side description change from "Old Desc" to "New Desc"
	found := false
	for _, ch := range cs.DrawioElementChanges {
		if ch.ID == "app" && ch.Type == Modified && ch.Field == "description" &&
			ch.OldValue == "Old Desc" && ch.NewValue == "New Desc" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected drawio description modification from tooltip change, got: %+v", cs.DrawioElementChanges)
	}
}

func simpleModelWithKind(id, title, description, technology, kind string) *model.BausteinsichtModel {
	return &model.BausteinsichtModel{
		Model: map[string]model.Element{
			id: {Kind: kind, Title: title, Description: description, Technology: technology},
		},
		Relationships: []model.Relationship{},
	}
}

func TestDetectChanges_ModelModifiedKind(t *testing.T) {
	state := stateWithElem("app", "App", "Desc", "Go")
	m := simpleModelWithKind("app", "App", "Desc", "Go", "component")
	doc := docWithElem("app", "App", "Go", "Desc")

	cs := DetectChanges(m, doc, state)

	found := false
	for _, ch := range cs.ModelElementChanges {
		if ch.ID == "app" && ch.Type == Modified && ch.Field == "kind" &&
			ch.OldValue == "container" && ch.NewValue == "component" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected model kind modification, changes: %+v", cs.ModelElementChanges)
	}
	if len(cs.Conflicts) != 0 {
		t.Errorf("expected no conflicts, got %d", len(cs.Conflicts))
	}
}

func TestDetectChanges_ScopedConnectorLabelChange(t *testing.T) {
	// Relationship exists in state with label "uses". draw.io connector (scoped)
	// has label "calls". Should detect a drawio-side label modification.
	state := emptyState()
	state.Relationships = []RelationshipState{
		{From: "a", To: "b", Label: "uses"},
	}

	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"a": {Kind: "system", Title: "A"},
			"b": {Kind: "system", Title: "B"},
		},
		Relationships: []model.Relationship{
			{From: "a", To: "b", Label: "uses"},
		},
	}

	doc := docWithScopedConnector("ctx", "a", "b", "calls")

	cs := DetectChanges(m, doc, state)

	found := false
	for _, ch := range cs.DrawioRelationshipChanges {
		if ch.From == "a" && ch.To == "b" && ch.Type == Modified &&
			ch.Field == "label" && ch.OldValue == "uses" && ch.NewValue == "calls" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected drawio label modification for scoped connector, got: %+v",
			cs.DrawioRelationshipChanges)
	}
}

// --- View-aware deletion tests (#108, #118) ---

// modelWithCustomViews creates a model with the given elements and views.
func modelWithCustomViews(elements map[string]model.Element, views map[string]model.View) *model.BausteinsichtModel {
	return &model.BausteinsichtModel{
		Model:         elements,
		Views:         views,
		Relationships: []model.Relationship{},
	}
}

// docWithElems creates a draw.io document with multiple elements by ID.
func docWithElems(ids ...string) *drawio.Document {
	doc := drawio.NewDocument()
	page := doc.AddPage("p1", "Page 1")
	for _, id := range ids {
		_ = page.CreateElement(drawio.ElementData{
			ID:    id,
			Kind:  "container",
			Title: id,
		}, "")
	}
	return doc
}

// stateWithElems creates a SyncState with multiple elements by ID.
func stateWithElems(ids ...string) *SyncState {
	s := emptyState()
	for _, id := range ids {
		s.Elements[id] = ElementState{Title: id, Kind: "container"}
	}
	return s
}

func TestDetectChanges_ElementNotOnViewNotTreatedAsDeleted(t *testing.T) {
	// Model has elements a, b, c.
	// View only includes a and b.
	// Sync state has a, b, c (from a previous sync without views).
	// draw.io only has a, b (because c is filtered out by the view).
	// c should NOT appear as a DrawioElementChange Deleted.
	m := modelWithCustomViews(
		map[string]model.Element{
			"a": {Kind: "container", Title: "A"},
			"b": {Kind: "container", Title: "B"},
			"c": {Kind: "container", Title: "C"},
		},
		map[string]model.View{
			"main": {Title: "Main", Include: []string{"a", "b"}},
		},
	)
	state := stateWithElems("a", "b", "c")
	doc := docWithElems("a", "b")

	cs := DetectChanges(m, doc, state)

	for _, ch := range cs.DrawioElementChanges {
		if ch.ID == "c" && ch.Type == Deleted {
			t.Errorf("element 'c' is not in any view and should NOT be treated as deleted from draw.io, got: %+v",
				cs.DrawioElementChanges)
		}
	}
}

func TestDetectChanges_ExcludedElementNotTreatedAsDeleted(t *testing.T) {
	// Model has elements a and b.
	// View includes a and b but explicitly excludes b.
	// Sync state has a and b.
	// draw.io only has a (b excluded from view).
	// b should NOT appear as a DrawioElementChange Deleted.
	m := modelWithCustomViews(
		map[string]model.Element{
			"a": {Kind: "container", Title: "A"},
			"b": {Kind: "container", Title: "B"},
		},
		map[string]model.View{
			"main": {Title: "Main", Include: []string{"a", "b"}, Exclude: []string{"b"}},
		},
	)
	state := stateWithElems("a", "b")
	doc := docWithElems("a")

	cs := DetectChanges(m, doc, state)

	for _, ch := range cs.DrawioElementChanges {
		if ch.ID == "b" && ch.Type == Deleted {
			t.Errorf("element 'b' is excluded from all views and should NOT be treated as deleted from draw.io, got: %+v",
				cs.DrawioElementChanges)
		}
	}
}

func TestDetectChanges_TrueDeletionStillDetected(t *testing.T) {
	// Model has elements a and b.
	// View includes both a and b (both visible).
	// Sync state has a and b.
	// draw.io only has a (user actually deleted b from draw.io).
	// b SHOULD appear as a DrawioElementChange Deleted.
	m := modelWithCustomViews(
		map[string]model.Element{
			"a": {Kind: "container", Title: "A"},
			"b": {Kind: "container", Title: "B"},
		},
		map[string]model.View{
			"main": {Title: "Main", Include: []string{"a", "b"}},
		},
	)
	state := stateWithElems("a", "b")
	doc := docWithElems("a")

	cs := DetectChanges(m, doc, state)

	found := false
	for _, ch := range cs.DrawioElementChanges {
		if ch.ID == "b" && ch.Type == Deleted {
			found = true
		}
	}
	if !found {
		t.Errorf("element 'b' is visible on a view and was removed from draw.io; should be detected as deleted, got: %+v",
			cs.DrawioElementChanges)
	}
}

func TestDetectChanges_NoViewsAllDeletionsDetected(t *testing.T) {
	// No views (legacy mode / backward compatibility).
	// Sync state has a and b.
	// draw.io only has a.
	// b SHOULD appear as a DrawioElementChange Deleted.
	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"a": {Kind: "container", Title: "A"},
			"b": {Kind: "container", Title: "B"},
		},
		Relationships: []model.Relationship{},
		// No Views field → nil map
	}
	state := stateWithElems("a", "b")
	doc := docWithElems("a")

	cs := DetectChanges(m, doc, state)

	found := false
	for _, ch := range cs.DrawioElementChanges {
		if ch.ID == "b" && ch.Type == Deleted {
			found = true
		}
	}
	if !found {
		t.Errorf("with no views, element 'b' absent from draw.io should be detected as deleted, got: %+v",
			cs.DrawioElementChanges)
	}
}
