package sync

import (
	"strings"
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

	cs := DetectChanges(m, doc, state, nil)

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

	cs := DetectChanges(m, doc, state, nil)

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

	cs := DetectChanges(m, doc, state, nil)

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

	cs := DetectChanges(m, doc, state, nil)

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

	cs := DetectChanges(m, doc, state, nil)

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

	cs := DetectChanges(m, doc, state, nil)

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

	cs := DetectChanges(m, doc, state, nil)

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

	cs := DetectChanges(m, doc, state, nil)

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

	cs := DetectChanges(m, doc, state, nil)

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

	cs := DetectChanges(m, doc, state, nil)

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

	cs := DetectChanges(m, doc, state, nil)

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

	cs := DetectChanges(m, doc, state, nil)

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

	cs := DetectChanges(m, doc, state, nil)

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

	cs := DetectChanges(m, doc, state, nil)

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

// TestDetectChanges_LiftedConnectorNotDeletedFromModel verifies that when a
// relationship exists in the model with deep endpoints (e.g., cli → model.loader)
// but the draw.io connector uses lifted endpoints (cli → model), the reverse
// sync does NOT treat the original relationship as deleted (#223).
func TestDetectChanges_LiftedConnectorNotDeletedFromModel(t *testing.T) {
	// Model has a deep relationship: cli → model.loader (index 5)
	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"cli": {Kind: "container", Title: "CLI"},
			"model": {Kind: "container", Title: "Model", Children: map[string]model.Element{
				"loader": {Kind: "component", Title: "Loader"},
			}},
		},
		Relationships: []model.Relationship{
			{From: "cli", To: "model.loader", Label: "loads JSONC", Kind: "uses"},
		},
		Views: map[string]model.View{
			"containers": {
				Title:   "Container View",
				Scope:   "",
				Include: []string{"cli", "model"},
			},
		},
	}

	// Sync state records the original relationship.
	state := emptyState()
	state.Elements["cli"] = ElementState{Title: "CLI", Kind: "container"}
	state.Elements["model"] = ElementState{Title: "Model", Kind: "container"}
	state.Elements["model.loader"] = ElementState{Title: "Loader", Kind: "component"}
	state.Relationships = []RelationshipState{
		{From: "cli", To: "model.loader", Index: 0, Label: "loads JSONC", Kind: "uses"},
	}

	// Draw.io has the connector with LIFTED endpoints (cli → model)
	// because the containers view doesn't include model.loader.
	doc := drawio.NewDocument()
	page := doc.AddPage("view-containers", "Container View")
	_ = page.CreateElement(drawio.ElementData{
		ID: "cli", CellID: "containers--cli", Kind: "container", Title: "CLI",
	}, "")
	_ = page.CreateElement(drawio.ElementData{
		ID: "model", CellID: "containers--model", Kind: "container", Title: "Model",
	}, "")
	page.CreateConnector(drawio.ConnectorData{
		From: "cli", To: "model",
		Label:     "loads JSONC",
		SourceRef: "containers--cli",
		TargetRef: "containers--model",
		Index:     0,
	}, "")

	cs := DetectChanges(m, doc, state, nil)

	// The original relationship (cli → model.loader) must NOT be marked as
	// deleted from draw.io. The lifted connector (cli → model) represents it.
	for _, ch := range cs.DrawioRelationshipChanges {
		if ch.From == "cli" && ch.To == "model.loader" && ch.Type == Deleted {
			t.Errorf("CRITICAL: relationship cli→model.loader should not be deleted when a lifted connector cli→model exists (#223), got: %+v",
				cs.DrawioRelationshipChanges)
		}
	}
}

// TestDetectChanges_RealDeletionStillWorksWithLiftedCheck ensures that a genuine
// user deletion (connector removed from draw.io, no lifted version exists) is
// still detected after the #223 fix.
func TestDetectChanges_RealDeletionStillWorksWithLiftedCheck(t *testing.T) {
	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"cli":   {Kind: "container", Title: "CLI"},
			"model": {Kind: "container", Title: "Model"},
		},
		Relationships: []model.Relationship{
			{From: "cli", To: "model", Label: "uses", Kind: "uses"},
		},
		Views: map[string]model.View{
			"containers": {
				Title:   "Container View",
				Include: []string{"cli", "model"},
			},
		},
	}

	state := emptyState()
	state.Elements["cli"] = ElementState{Title: "CLI", Kind: "container"}
	state.Elements["model"] = ElementState{Title: "Model", Kind: "container"}
	state.Relationships = []RelationshipState{
		{From: "cli", To: "model", Index: 0, Label: "uses", Kind: "uses"},
	}

	// Draw.io has the elements but the user deleted the connector.
	doc := drawio.NewDocument()
	page := doc.AddPage("view-containers", "Container View")
	_ = page.CreateElement(drawio.ElementData{
		ID: "cli", CellID: "containers--cli", Kind: "container", Title: "CLI",
	}, "")
	_ = page.CreateElement(drawio.ElementData{
		ID: "model", CellID: "containers--model", Kind: "container", Title: "Model",
	}, "")
	// No connector — user deleted it.

	cs := DetectChanges(m, doc, state, nil)

	found := false
	for _, ch := range cs.DrawioRelationshipChanges {
		if ch.From == "cli" && ch.To == "model" && ch.Type == Deleted {
			found = true
		}
	}
	if !found {
		t.Error("expected real user deletion cli→model to be detected, but it was not")
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

	cs := DetectChanges(m, doc, state, nil)

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

	cs := DetectChanges(m, doc, state, nil)

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

	cs := DetectChanges(m, doc, state, nil)

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

	cs := DetectChanges(m, doc, state, nil)

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

	cs := DetectChanges(m, doc, state, nil)

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

	cs := DetectChanges(m, doc, state, nil)

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

	cs := DetectChanges(m, doc, state, nil)

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

// --- Multiple relationships between same pair (#142) ---

func TestDetectChanges_MultipleRelsSamePairBothAdded(t *testing.T) {
	// Model has two relationships from A to B with different labels.
	// State is empty (first sync). Both should be detected as Added.
	state := emptyState()
	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"a": {Kind: "container", Title: "A"},
			"b": {Kind: "container", Title: "B"},
		},
		Relationships: []model.Relationship{
			{From: "a", To: "b", Label: "uses"},
			{From: "a", To: "b", Label: "calls"},
		},
	}
	doc := emptyDoc()

	cs := DetectChanges(m, doc, state, nil)

	addedCount := 0
	for _, ch := range cs.ModelRelationshipChanges {
		if ch.From == "a" && ch.To == "b" && ch.Type == Added {
			addedCount++
		}
	}
	if addedCount != 2 {
		t.Errorf("expected 2 Added relationship changes for a→b, got %d: %+v",
			addedCount, cs.ModelRelationshipChanges)
	}
}

func TestDetectChanges_MultipleRelsSamePairLabelModified(t *testing.T) {
	// State has two rels from A to B. Model modifies label of the second.
	state := emptyState()
	state.Relationships = []RelationshipState{
		{From: "a", To: "b", Index: 0, Label: "uses"},
		{From: "a", To: "b", Index: 1, Label: "calls"},
	}
	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"a": {Kind: "container", Title: "A"},
			"b": {Kind: "container", Title: "B"},
		},
		Relationships: []model.Relationship{
			{From: "a", To: "b", Label: "uses"},
			{From: "a", To: "b", Label: "invokes"}, // changed from "calls"
		},
	}
	doc := emptyDoc()

	cs := DetectChanges(m, doc, state, nil)

	found := false
	for _, ch := range cs.ModelRelationshipChanges {
		if ch.From == "a" && ch.To == "b" && ch.Index == 1 &&
			ch.Type == Modified && ch.Field == "label" &&
			ch.OldValue == "calls" && ch.NewValue == "invokes" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected label modification for a→b index=1, got: %+v",
			cs.ModelRelationshipChanges)
	}

	// The first relationship (index 0) should NOT be modified.
	for _, ch := range cs.ModelRelationshipChanges {
		if ch.From == "a" && ch.To == "b" && ch.Index == 0 && ch.Type == Modified {
			t.Errorf("first relationship (index=0) should not be modified, got: %+v", ch)
		}
	}
}

// --- Scoped cell ID leak on element deletion (#166) ---

func TestDetectChanges_DeletedElementConnectorUsesCanonicalIDs(t *testing.T) {
	// Scenario: element "db" (scoped cell ID "components--onlineshop.db") was
	// deleted from the draw.io document, but a connector referencing it via the
	// scoped cell ID still exists. The connector's source/target should be
	// resolved to canonical element IDs, not leaked as scoped cell IDs.
	//
	// Setup:
	// - Model has elements "webapp" and "onlineshop.db" with a relationship.
	// - State recorded the relationship from previous sync.
	// - draw.io doc: "onlineshop.db" element was deleted, but a connector from
	//   "components--webapp" to "components--onlineshop.db" remains.
	//   "webapp" element still exists with bausteinsicht_id="webapp".
	state := emptyState()
	state.Elements["webapp"] = ElementState{Title: "WebApp", Kind: "container"}
	state.Elements["onlineshop.db"] = ElementState{Title: "Database", Kind: "container"}
	state.Relationships = []RelationshipState{
		{From: "webapp", To: "onlineshop.db", Label: "reads"},
	}

	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"webapp": {Kind: "container", Title: "WebApp"},
			"onlineshop": {Kind: "system", Title: "Onlineshop", Children: map[string]model.Element{
				"db": {Kind: "container", Title: "Database"},
			}},
		},
		Relationships: []model.Relationship{
			{From: "webapp", To: "onlineshop.db", Label: "reads"},
		},
	}

	// Build draw.io doc manually: webapp exists (scoped), db is gone, connector remains.
	doc := drawio.NewDocument()
	page := doc.AddPage("view-components", "Components View")
	// webapp element with scoped cell ID
	_ = page.CreateElement(drawio.ElementData{
		ID:     "webapp",
		CellID: "components--webapp",
		Kind:   "container",
		Title:  "WebApp",
	}, "")
	// Note: onlineshop.db element is NOT in the document (deleted).
	// But the connector still references its scoped cell ID.
	page.CreateConnector(drawio.ConnectorData{
		From:      "webapp",
		To:        "onlineshop.db",
		SourceRef: "components--webapp",
		TargetRef: "components--onlineshop.db",
	}, "")

	cs := DetectChanges(m, doc, state, nil)

	// Check all drawio relationship changes: none should contain scoped cell IDs.
	for _, ch := range cs.DrawioRelationshipChanges {
		if strings.Contains(ch.From, "--") {
			t.Errorf("DrawioRelationshipChange.From contains scoped cell ID %q, expected canonical element ID", ch.From)
		}
		if strings.Contains(ch.To, "--") {
			t.Errorf("DrawioRelationshipChange.To contains scoped cell ID %q, expected canonical element ID", ch.To)
		}
	}

	// Also verify via ApplyReverse: if changes are applied, the model should
	// not contain relationships with scoped cell IDs.
	ApplyReverse(cs, m)
	for _, rel := range m.Relationships {
		if strings.Contains(rel.From, "--") {
			t.Errorf("Model relationship From contains scoped cell ID %q after reverse sync", rel.From)
		}
		if strings.Contains(rel.To, "--") {
			t.Errorf("Model relationship To contains scoped cell ID %q after reverse sync", rel.To)
		}
	}
}

func TestExtractDrawioRelationships_FallbackStripsViewPrefix(t *testing.T) {
	// When a connector references a scoped cell ID that is NOT in the
	// cellToElem map (e.g., because the element was deleted), the fallback
	// should strip the view prefix ("viewID--") to recover the element ID.
	doc := drawio.NewDocument()
	page := doc.AddPage("view-ctx", "Context View")
	// Only "a" exists as an element. "b" was deleted.
	_ = page.CreateElement(drawio.ElementData{
		ID:     "a",
		CellID: "ctx--a",
		Kind:   "system",
		Title:  "A",
	}, "")
	// Connector still references "ctx--b" which has no element in the doc.
	page.CreateConnector(drawio.ConnectorData{
		From:      "a",
		To:        "b",
		SourceRef: "ctx--a",
		TargetRef: "ctx--b",
	}, "")

	rels := extractDrawioRelationships(doc)

	for key, rel := range rels {
		if strings.Contains(rel.From, "--") {
			t.Errorf("relationship key=%s: From=%q contains scoped cell ID prefix", key, rel.From)
		}
		if strings.Contains(rel.To, "--") {
			t.Errorf("relationship key=%s: To=%q contains scoped cell ID prefix", key, rel.To)
		}
	}
}

// TestDetectChanges_TechnologyFromXMLAttribute verifies that when a user adds
// a technology XML attribute to a draw.io element (without changing the label),
// the reverse sync detects the change (#186).
func TestDetectChanges_TechnologyFromXMLAttribute(t *testing.T) {
	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"customer": {Kind: "actor", Title: "Customer"},
		},
	}

	doc := drawio.NewDocument()
	doc.AddPage("1", "Page 1")
	page := requirePage(t, doc, "1")
	if err := page.CreateElement(drawio.ElementData{
		ID:     "customer",
		CellID: "customer",
		Kind:   "actor",
		Title:  "Customer",
	}, "shape=actor;"); err != nil {
		t.Fatal(err)
	}
	// Simulate user adding technology attribute directly in XML.
	elem := page.FindElement("customer")
	if elem == nil {
		t.Fatal("element not found")
	}
	elem.CreateAttr("technology", "Human")

	lastState := &SyncState{
		Elements: map[string]ElementState{
			"customer": {Title: "Customer", Kind: "actor"},
		},
	}

	cs := DetectChanges(m, doc, lastState, nil)

	// Should detect a drawio-side technology change.
	found := false
	for _, ch := range cs.DrawioElementChanges {
		if ch.ID == "customer" && ch.Field == "technology" && ch.NewValue == "Human" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected drawio technology change for 'customer', got changes: %+v", cs.DrawioElementChanges)
	}
}

// TestExtractDrawioRelationships_IgnoresNavBackConnectors verifies that
// connectors targeting navigation buttons (nav-back-*) are excluded from
// relationship extraction to prevent phantom relationships (#205).
func TestExtractDrawioRelationships_IgnoresNavBackConnectors(t *testing.T) {
	doc := drawio.NewDocument()
	doc.AddPage("view-ctx", "Context")
	page := requirePage(t, doc, "view-ctx")

	// Create a managed element.
	if err := page.CreateElement(drawio.ElementData{
		ID: "svc", CellID: "ctx--svc",
		Kind: "system", Title: "Service",
	}, "shape=test;"); err != nil {
		t.Fatal(err)
	}

	// Create a nav-back button (no bausteinsicht_id).
	root := page.Root()
	navObj := root.CreateElement("object")
	navObj.CreateAttr("label", "&larr; Overview")
	navObj.CreateAttr("id", "nav-back-ctx")
	navObj.CreateAttr("link", "data:page/id,view-overview")
	navCell := navObj.CreateElement("mxCell")
	navCell.CreateAttr("style", "rounded=1;")
	navCell.CreateAttr("vertex", "1")
	navCell.CreateAttr("parent", "1")

	// Create a connector from svc to the nav-back button.
	conn := root.CreateElement("mxCell")
	conn.CreateAttr("id", "user-edge-1")
	conn.CreateAttr("edge", "1")
	conn.CreateAttr("source", "ctx--svc")
	conn.CreateAttr("target", "nav-back-ctx")
	conn.CreateAttr("parent", "1")

	rels := extractDrawioRelationships(doc)

	// Should NOT contain any relationship targeting nav-back-ctx.
	for key, rel := range rels {
		if strings.Contains(rel.From, "nav-back") || strings.Contains(rel.To, "nav-back") {
			t.Errorf("expected nav-back connector to be excluded, got relationship %q: %+v", key, rel)
		}
	}
}

// TestDetectChanges_NewRelationshipFromDrawioHasLabel verifies that a new
// connector added in draw.io carries its label (value attribute) through to
// the Added relationship change (#204).
func TestDetectChanges_NewRelationshipFromDrawioHasLabel(t *testing.T) {
	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"a": {Kind: "system", Title: "A"},
			"b": {Kind: "system", Title: "B"},
		},
		Relationships: []model.Relationship{},
	}

	// Create drawio doc with a connector from a to b.
	doc := drawio.NewDocument()
	doc.AddPage("view-ctx", "Context")
	page := requirePage(t, doc, "view-ctx")
	for _, id := range []string{"a", "b"} {
		if err := page.CreateElement(drawio.ElementData{
			ID: id, CellID: "ctx--" + id,
			Kind: "system", Title: strings.ToUpper(id),
		}, "shape=test;"); err != nil {
			t.Fatal(err)
		}
	}
	// Add a connector with a label.
	root := page.Root()
	conn := root.CreateElement("mxCell")
	conn.CreateAttr("id", "user-edge-1")
	conn.CreateAttr("value", "reads from")
	conn.CreateAttr("edge", "1")
	conn.CreateAttr("source", "ctx--a")
	conn.CreateAttr("target", "ctx--b")
	conn.CreateAttr("parent", "1")

	lastState := &SyncState{
		Elements: map[string]ElementState{
			"a": {Title: "A"},
			"b": {Title: "B"},
		},
		Relationships: []RelationshipState{},
	}

	cs := DetectChanges(m, doc, lastState, nil)

	// Should detect an Added relationship change with the label.
	var found bool
	for _, ch := range cs.DrawioRelationshipChanges {
		if ch.Type == Added && ch.From == "a" && ch.To == "b" {
			if ch.NewValue != "reads from" {
				t.Errorf("expected NewValue=%q, got %q", "reads from", ch.NewValue)
			}
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected Added relationship change a->b, got: %+v", cs.DrawioRelationshipChanges)
	}
}

func TestDetectChanges_SelfRefConnectorNotClassifiedAsLifted(t *testing.T) {
	// Model has "backend" with children "backend.api" and "backend.db",
	// and a child-to-child relationship backend.api → backend.db.
	// A user draws a self-referencing connector backend → backend in draw.io.
	// This should NOT be classified as a "lifted" version of backend.api → backend.db.
	state := emptyState()
	state.Elements["backend"] = ElementState{Title: "Backend", Kind: "system"}
	state.Relationships = []RelationshipState{
		{From: "backend.api", To: "backend.db", Label: "queries"},
	}

	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"backend": {Kind: "system", Title: "Backend", Children: map[string]model.Element{
				"api": {Kind: "container", Title: "API"},
				"db":  {Kind: "container", Title: "DB"},
			}},
		},
		Relationships: []model.Relationship{
			{From: "backend.api", To: "backend.db", Label: "queries"},
		},
	}

	doc := drawio.NewDocument()
	page := doc.AddPage("view-ctx", "Context")
	_ = page.CreateElement(drawio.ElementData{
		ID: "backend", CellID: "ctx--backend", Kind: "system", Title: "Backend",
	}, "")
	// Existing lifted connector for backend.api → backend.db
	page.CreateConnector(drawio.ConnectorData{
		From: "backend", To: "backend",
		Label:     "monitors",
		SourceRef: "ctx--backend",
		TargetRef: "ctx--backend",
	}, "")

	cs := DetectChanges(m, doc, state, nil)

	// The self-referencing connector should be detected as Added, not ignored.
	found := false
	for _, ch := range cs.DrawioRelationshipChanges {
		if ch.From == "backend" && ch.To == "backend" && ch.Type == Added {
			found = true
			if ch.NewValue != "monitors" {
				t.Errorf("expected label 'monitors', got %q", ch.NewValue)
			}
		}
	}
	if !found {
		t.Errorf("self-referencing connector backend→backend should be detected as Added, got: %+v",
			cs.DrawioRelationshipChanges)
	}
}

func TestDetectChanges_ConnectorToNewUnmanagedElementUsesModelID(t *testing.T) {
	// When a user draws both a new element AND a connector to it in the
	// same draw.io session, the connector endpoint should resolve to the
	// sanitized model ID (not the raw cell ID). See #211.
	state := emptyState()
	state.Elements["customer"] = ElementState{Title: "Customer", Kind: "system"}

	m := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"system": {Notation: "System"},
			},
		},
		Model: map[string]model.Element{
			"customer": {Kind: "system", Title: "Customer"},
		},
	}

	doc := drawio.NewDocument()
	page := doc.AddPage("view-ctx", "Context")
	_ = page.CreateElement(drawio.ElementData{
		ID: "customer", CellID: "ctx--customer", Kind: "system", Title: "Customer",
	}, "")
	// Add an unmanaged element (no bausteinsicht_id) — simulates user drawing a shape.
	root := page.Root()
	obj := root.CreateElement("object")
	obj.CreateAttr("label", "Gateway")
	obj.CreateAttr("id", "new-gw-elem")
	cell := obj.CreateElement("mxCell")
	cell.CreateAttr("style", "rounded=1;whiteSpace=wrap;html=1;")
	cell.CreateAttr("vertex", "1")
	cell.CreateAttr("parent", "1")
	geo := cell.CreateElement("mxGeometry")
	geo.CreateAttr("x", "400")
	geo.CreateAttr("y", "200")
	geo.CreateAttr("width", "160")
	geo.CreateAttr("height", "70")
	geo.CreateAttr("as", "geometry")

	// Add connector from customer to the new unmanaged element.
	page.CreateConnector(drawio.ConnectorData{
		From:      "customer",
		To:        "gateway", // won't matter — source/target use cell IDs
		Label:     "routes",
		SourceRef: "ctx--customer",
		TargetRef: "new-gw-elem",
	}, "")

	cs := DetectChanges(m, doc, state, nil)

	// The connector's target should resolve to "gateway" (sanitized from label),
	// not "new-gw-elem" (raw cell ID).
	found := false
	for _, ch := range cs.DrawioRelationshipChanges {
		if ch.Type == Added && ch.From == "customer" && ch.To == "gateway" {
			found = true
		}
		if ch.Type == Added && ch.To == "new-gw-elem" {
			t.Errorf("connector resolved to raw cell ID 'new-gw-elem' instead of sanitized model ID 'gateway'")
		}
	}
	if !found {
		t.Errorf("expected Added relationship customer→gateway, got: %+v", cs.DrawioRelationshipChanges)
	}
}

func TestDetectChanges_DuplicateBausteinsichtIDKeepsFirst(t *testing.T) {
	// When draw.io contains two <object> elements with the same
	// bausteinsicht_id (e.g., copy-paste), the first occurrence should
	// win. The duplicate should not overwrite the original's data (#213).
	state := emptyState()
	state.Elements["svc"] = ElementState{
		Title:       "Original Service",
		Description: "Original description",
		Kind:        "system",
	}

	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"svc": {Kind: "system", Title: "Original Service", Description: "Original description"},
		},
	}

	doc := drawio.NewDocument()
	page := doc.AddPage("view-ctx", "Context")
	// First element: matches the model (no change expected).
	_ = page.CreateElement(drawio.ElementData{
		ID:          "svc",
		CellID:      "ctx--svc",
		Kind:        "system",
		Title:       "Original Service",
		Description: "Original description",
	}, "")
	// Second element: duplicate bausteinsicht_id with different title.
	// Simulates a copy-paste in draw.io.
	_ = page.CreateElement(drawio.ElementData{
		ID:     "svc",
		CellID: "ctx--svc-copy",
		Kind:   "system",
		Title:  "Collision Copy",
	}, "")

	cs := DetectChanges(m, doc, state, nil)

	// There should be NO Modified element change for "svc" — the first
	// occurrence matches the model/last-sync state exactly.
	for _, ch := range cs.DrawioElementChanges {
		if ch.ID == "svc" && ch.Type == Modified {
			t.Errorf("duplicate bausteinsicht_id caused spurious Modified change: field=%q old=%q new=%q",
				ch.Field, ch.OldValue, ch.NewValue)
		}
	}
}

// --- Cross-view consistency (#236) ---

func TestDetectChanges_CrossViewStaleTitle(t *testing.T) {
	// Scenario: reverse sync updated the title in one view (and the model/state),
	// but a second view still shows the old title. The next sync must emit a
	// forward change to bring the stale view up to date.
	//
	// After reverse sync:
	//   model  = "New Title"
	//   state  = "New Title"
	//   View A = "New Title"  (was edited here)
	//   View B = "Old Title"  (stale)

	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"svc": {Kind: "system", Title: "New Title"},
		},
		Views: map[string]model.View{
			"context":    {Title: "Context"},
			"containers": {Title: "Containers"},
		},
		Relationships: []model.Relationship{},
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"system": {},
			},
		},
	}

	state := emptyState()
	state.Elements["svc"] = ElementState{Title: "New Title", Kind: "system"}

	// Build a two-page document: page 1 has the updated title, page 2 is stale.
	doc := drawio.NewDocument()
	page1 := doc.AddPage("view-context", "Context View")
	_ = page1.CreateElement(drawio.ElementData{
		ID: "svc", CellID: "context--svc", Kind: "system", Title: "New Title",
	}, "")
	page2 := doc.AddPage("view-containers", "Containers View")
	_ = page2.CreateElement(drawio.ElementData{
		ID: "svc", CellID: "containers--svc", Kind: "system", Title: "Old Title",
	}, "")

	cs := DetectChanges(m, doc, state, nil)

	// Expect a forward change to update the stale title.
	found := false
	for _, ch := range cs.ModelElementChanges {
		if ch.ID == "svc" && ch.Type == Modified && ch.Field == "title" &&
			ch.OldValue == "Old Title" && ch.NewValue == "New Title" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected forward change to fix stale cross-view title, got model changes: %+v",
			cs.ModelElementChanges)
	}

	// Should NOT produce a drawio-side change (no conflict — model is authoritative).
	for _, ch := range cs.DrawioElementChanges {
		if ch.ID == "svc" && ch.Type == Modified && ch.Field == "title" {
			t.Errorf("unexpected drawio-side title change: %+v", ch)
		}
	}
}

func TestDetectChanges_CrossViewConsistentNoSpuriousChange(t *testing.T) {
	// When all views have the same title as the model, no cross-view change
	// should be emitted.
	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"svc": {Kind: "system", Title: "Service"},
		},
		Views: map[string]model.View{
			"context":    {Title: "Context"},
			"containers": {Title: "Containers"},
		},
		Relationships: []model.Relationship{},
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"system": {},
			},
		},
	}

	state := emptyState()
	state.Elements["svc"] = ElementState{Title: "Service", Kind: "system"}

	doc := drawio.NewDocument()
	page1 := doc.AddPage("view-context", "Context View")
	_ = page1.CreateElement(drawio.ElementData{
		ID: "svc", CellID: "context--svc", Kind: "system", Title: "Service",
	}, "")
	page2 := doc.AddPage("view-containers", "Containers View")
	_ = page2.CreateElement(drawio.ElementData{
		ID: "svc", CellID: "containers--svc", Kind: "system", Title: "Service",
	}, "")

	cs := DetectChanges(m, doc, state, nil)

	for _, ch := range cs.ModelElementChanges {
		if ch.ID == "svc" && ch.Type == Modified {
			t.Errorf("no cross-view change expected when all views are consistent, got: %+v", ch)
		}
	}
}

// --- View include expansion (#240) ---

func TestDetectChanges_ViewIncludeExpansionNotTreatedAsDeleted(t *testing.T) {
	// Scenario: element "svc" is in model and state (previously synced) but not
	// in draw.io because it was not included in any view. The user adds it to a
	// view's include list. The element should NOT be treated as "deleted from
	// draw.io" — forward sync will create it on the view page.
	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"svc": {Kind: "system", Title: "Service"},
		},
		Views: map[string]model.View{
			"context": {Title: "Context", Include: []string{"svc"}},
		},
		Relationships: []model.Relationship{},
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"system": {},
			},
		},
	}

	state := emptyState()
	state.Elements["svc"] = ElementState{Title: "Service", Kind: "system"}
	// "svc" was NOT on any draw.io page during the last sync (view didn't include it).
	state.RenderedElements = map[string]bool{}

	// draw.io has NO "svc" element (it was filtered before).
	doc := drawio.NewDocument()
	doc.AddPage("view-context", "Context View")

	cs := DetectChanges(m, doc, state, nil)

	// Should NOT have a draw.io-side deletion.
	for _, ch := range cs.DrawioElementChanges {
		if ch.ID == "svc" && ch.Type == Deleted {
			t.Errorf("element newly included in view should not be treated as deleted from draw.io: %+v", ch)
		}
	}
}

func TestDetectChanges_ActualDrawioDeletionStillDetected(t *testing.T) {
	// Scenario: element "svc" was visible in draw.io, user deleted it manually.
	// Model still has it with unchanged data, state still has it, draw.io does NOT.
	// This should be treated as a draw.io deletion because RenderedElements
	// confirms the element was previously on a page.
	//
	// The difference from #240: here RenderedElements includes "svc" (it was
	// on a draw.io page during the last sync).
	m := simpleModel("svc", "Service", "", "")

	state := stateWithElem("svc", "Service", "", "")
	// "svc" WAS on a draw.io page during the last sync.
	state.RenderedElements = map[string]bool{"svc": true}

	doc := emptyDoc() // no "svc" element — user deleted it

	cs := DetectChanges(m, doc, state, nil)

	// Draw.io-side deletion should be detected because element was rendered.
	foundDeletion := false
	for _, ch := range cs.DrawioElementChanges {
		if ch.ID == "svc" && ch.Type == Deleted {
			foundDeletion = true
		}
	}
	if !foundDeletion {
		t.Error("expected draw.io-side deletion when element was previously rendered")
	}
}
