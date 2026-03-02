package sync

import (
	"strconv"
	"strings"
	"testing"

	"github.com/docToolchain/Bauteinsicht/internal/drawio"
	"github.com/docToolchain/Bauteinsicht/internal/model"
)

// minimalTemplates returns a TemplateSet loaded from a small inline template.
func minimalTemplates(t *testing.T) *drawio.TemplateSet {
	t.Helper()
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<mxfile>
  <diagram id="t1" name="templates">
    <mxGraphModel>
      <root>
        <mxCell id="0"/>
        <mxCell id="1" parent="0"/>
        <object bausteinsicht_template="container" label="" id="tpl-container">
          <mxCell style="shape=mxgraph.c4;c4Type=container;" vertex="1" parent="1">
            <mxGeometry width="120" height="60" as="geometry"/>
          </mxCell>
        </object>
        <mxCell bausteinsicht_template="relationship" style="endArrow=block;" edge="1" parent="1">
          <mxGeometry relative="1" as="geometry"/>
        </mxCell>
      </root>
    </mxGraphModel>
  </diagram>
</mxfile>`
	ts, err := drawio.LoadTemplateFromBytes([]byte(xml))
	if err != nil {
		t.Fatalf("LoadTemplateFromBytes: %v", err)
	}
	return ts
}

// modelWithElem returns a model containing a single element.
func modelWithElem(id, kind, title string) *model.BausteinsichtModel {
	return &model.BausteinsichtModel{
		Model: map[string]model.Element{
			id: {Kind: kind, Title: title},
		},
		Relationships: []model.Relationship{},
	}
}

// fwdModelWithRel returns a model with two elements and one relationship.
func fwdModelWithRel(fromID, toID string) *model.BausteinsichtModel {
	return &model.BausteinsichtModel{
		Model: map[string]model.Element{
			fromID: {Kind: "container", Title: "From"},
			toID:   {Kind: "container", Title: "To"},
		},
		Relationships: []model.Relationship{
			{From: fromID, To: toID, Label: "calls"},
		},
	}
}

// templatesWithComponent returns a TemplateSet with both "container" and "component" kinds.
func templatesWithComponent(t *testing.T) *drawio.TemplateSet {
	t.Helper()
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<mxfile>
  <diagram id="t1" name="templates">
    <mxGraphModel>
      <root>
        <mxCell id="0"/>
        <mxCell id="1" parent="0"/>
        <object bausteinsicht_template="container" label="" id="tpl-container">
          <mxCell style="shape=mxgraph.c4;c4Type=container;" vertex="1" parent="1">
            <mxGeometry width="120" height="60" as="geometry"/>
          </mxCell>
        </object>
        <object bausteinsicht_template="component" label="" id="tpl-component">
          <mxCell style="shape=mxgraph.c4;c4Type=component;" vertex="1" parent="1">
            <mxGeometry width="100" height="50" as="geometry"/>
          </mxCell>
        </object>
        <mxCell bausteinsicht_template="relationship" style="endArrow=block;" edge="1" parent="1">
          <mxGeometry relative="1" as="geometry"/>
        </mxCell>
      </root>
    </mxGraphModel>
  </diagram>
</mxfile>`
	ts, err := drawio.LoadTemplateFromBytes([]byte(xml))
	if err != nil {
		t.Fatalf("LoadTemplateFromBytes: %v", err)
	}
	return ts
}

// TestApplyForward_ElementKindChanged verifies that a kind change updates
// the bausteinsicht_kind attribute and the mxCell style.
func TestApplyForward_ElementKindChanged(t *testing.T) {
	// Create a doc with an element of kind "container"
	doc := docWithElem("api", "API", "Go", "")
	ts := templatesWithComponent(t)
	// Model now has kind "component"
	m := modelWithElem("api", "component", "API")

	cs := &ChangeSet{
		ModelElementChanges: []ElementChange{
			{ID: "api", Type: Modified, Field: "kind", OldValue: "container", NewValue: "component"},
		},
	}

	result := ApplyForward(cs, doc, ts, m)

	if result.ElementsUpdated != 1 {
		t.Fatalf("expected 1 element updated, got %d", result.ElementsUpdated)
	}

	page := doc.Pages()[0]
	obj := page.FindElement("api")
	if obj == nil {
		t.Fatal("element 'api' not found")
	}

	// Verify bausteinsicht_kind attribute was updated
	kindAttr := obj.SelectAttrValue("bausteinsicht_kind", "")
	if kindAttr != "component" {
		t.Errorf("expected bausteinsicht_kind='component', got %q", kindAttr)
	}

	// Verify style was updated to component style
	cell := obj.FindElement("mxCell")
	if cell == nil {
		t.Fatal("mxCell not found under element")
	}
	style := cell.SelectAttrValue("style", "")
	if !strings.Contains(style, "c4Type=component") {
		t.Errorf("expected style to contain component template style, got: %s", style)
	}
}

// TestApplyForward_NewElement verifies that an Added element change creates the element.
func TestApplyForward_NewElement(t *testing.T) {
	doc := emptyDoc()
	ts := minimalTemplates(t)
	m := modelWithElem("api", "container", "API")

	cs := &ChangeSet{
		ModelElementChanges: []ElementChange{
			{ID: "api", Type: Added},
		},
	}

	result := ApplyForward(cs, doc, ts, m)

	if result.ElementsCreated != 1 {
		t.Fatalf("expected 1 element created, got %d", result.ElementsCreated)
	}

	page := doc.Pages()[0]
	obj := page.FindElement("api")
	if obj == nil {
		t.Fatal("element 'api' not found in document")
	}
	if len(result.Warnings) > 0 {
		t.Errorf("unexpected warnings: %v", result.Warnings)
	}
}

// TestApplyForward_NewElementVisualMarker verifies the new-element style marker is applied.
func TestApplyForward_NewElementVisualMarker(t *testing.T) {
	doc := emptyDoc()
	ts := minimalTemplates(t)
	m := modelWithElem("api", "container", "API")

	cs := &ChangeSet{
		ModelElementChanges: []ElementChange{{ID: "api", Type: Added}},
	}
	ApplyForward(cs, doc, ts, m)

	page := doc.Pages()[0]
	obj := page.FindElement("api")
	if obj == nil {
		t.Fatal("element 'api' not found")
	}
	cell := obj.FindElement("mxCell")
	if cell == nil {
		t.Fatal("mxCell not found under element")
	}
	style := cell.SelectAttrValue("style", "")
	if style == "" {
		t.Fatal("style is empty")
	}
	if !strings.Contains(style, "strokeColor=#FF0000") {
		t.Errorf("expected visual marker in style, got: %s", style)
	}
}

// TestApplyForward_UpdatedTitle verifies that a Modified/title change updates the label.
func TestApplyForward_UpdatedTitle(t *testing.T) {
	doc := docWithElem("api", "Old Title", "", "")
	ts := minimalTemplates(t)
	m := modelWithElem("api", "container", "New Title")

	cs := &ChangeSet{
		ModelElementChanges: []ElementChange{
			{ID: "api", Type: Modified, Field: "title", OldValue: "Old Title", NewValue: "New Title"},
		},
	}

	result := ApplyForward(cs, doc, ts, m)

	if result.ElementsUpdated != 1 {
		t.Fatalf("expected 1 element updated, got %d", result.ElementsUpdated)
	}

	page := doc.Pages()[0]
	obj := page.FindElement("api")
	if obj == nil {
		t.Fatal("element 'api' not found")
	}
	label := obj.SelectAttrValue("label", "")
	if !strings.Contains(label, "New Title") {
		t.Errorf("expected label to contain 'New Title', got: %s", label)
	}
}

// TestApplyForward_DeletedElement verifies that a Deleted change removes the element.
func TestApplyForward_DeletedElement(t *testing.T) {
	doc := docWithElem("api", "API", "", "")
	ts := minimalTemplates(t)
	m := emptyModel()

	cs := &ChangeSet{
		ModelElementChanges: []ElementChange{
			{ID: "api", Type: Deleted},
		},
	}

	result := ApplyForward(cs, doc, ts, m)

	if result.ElementsDeleted != 1 {
		t.Fatalf("expected 1 element deleted, got %d", result.ElementsDeleted)
	}

	page := doc.Pages()[0]
	if page.FindElement("api") != nil {
		t.Fatal("element 'api' should have been removed")
	}
}

// TestApplyForward_NewRelationship verifies that an Added relationship creates a connector.
func TestApplyForward_NewRelationship(t *testing.T) {
	doc := emptyDoc()
	ts := minimalTemplates(t)
	m := fwdModelWithRel("frontend", "backend")

	cs := &ChangeSet{
		ModelRelationshipChanges: []RelationshipChange{
			{From: "frontend", To: "backend", Type: Added, NewValue: "calls"},
		},
	}

	result := ApplyForward(cs, doc, ts, m)

	if result.ConnectorsCreated != 1 {
		t.Fatalf("expected 1 connector created, got %d", result.ConnectorsCreated)
	}

	page := doc.Pages()[0]
	if page.FindConnector("frontend", "backend", 0) == nil {
		t.Fatal("connector 'frontend→backend' not found")
	}
}

// TestApplyForward_MultipleNewElementsNoOverlap verifies placement doesn't overlap elements.
func TestApplyForward_MultipleNewElementsNoOverlap(t *testing.T) {
	doc := emptyDoc()
	ts := minimalTemplates(t)
	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"a": {Kind: "container", Title: "A"},
			"b": {Kind: "container", Title: "B"},
		},
		Relationships: []model.Relationship{},
	}

	cs := &ChangeSet{
		ModelElementChanges: []ElementChange{
			{ID: "a", Type: Added},
			{ID: "b", Type: Added},
		},
	}

	result := ApplyForward(cs, doc, ts, m)

	if result.ElementsCreated != 2 {
		t.Fatalf("expected 2 elements created, got %d", result.ElementsCreated)
	}

	page := doc.Pages()[0]
	xA := elementX(page, "a")
	xB := elementX(page, "b")
	if xA == xB {
		t.Errorf("elements 'a' and 'b' have the same X position (%g), expected no overlap", xA)
	}
}

// TestApplyForward_NoPageWarning verifies a warning is emitted when the document has no pages.
func TestApplyForward_NoPageWarning(t *testing.T) {
	doc := drawio.NewDocument()
	ts := minimalTemplates(t)
	m := emptyModel()
	cs := &ChangeSet{}

	result := ApplyForward(cs, doc, ts, m)

	if len(result.Warnings) == 0 {
		t.Fatal("expected a warning for document with no pages")
	}
}

// helpers

// elementX returns the X position of an element by bausteinsicht_id, or -1 if not found.
func elementX(page *drawio.Page, id string) float64 {
	obj := page.FindElement(id)
	if obj == nil {
		return -1
	}
	cell := obj.FindElement("mxCell")
	if cell == nil {
		return -1
	}
	geo := cell.FindElement("mxGeometry")
	if geo == nil {
		return -1
	}
	val := geo.SelectAttrValue("x", "-1")
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return -1
	}
	return f
}

// TestReconcileViewPage_PreservesUserAddedElements verifies that elements
// manually added by the user in draw.io (with bausteinsicht_id but NOT in
// the model) are preserved during reconciliation, while model elements not
// in the view's resolved set are deleted. Regression test for #115.
func TestReconcileViewPage_PreservesUserAddedElements(t *testing.T) {
	// Create a page with three elements:
	// - "a": in the model AND in the view filter (should be kept)
	// - "b": in the model but NOT in the view filter (should be deleted)
	// - "useradded": NOT in the model at all (user-added, should be preserved)
	doc := drawio.NewDocument()
	page := doc.AddPage("view-test", "Test View")
	_ = page.CreateElement(drawio.ElementData{
		ID: "a", CellID: "test--a", Kind: "container", Title: "A",
	}, "")
	_ = page.CreateElement(drawio.ElementData{
		ID: "b", CellID: "test--b", Kind: "container", Title: "B",
	}, "")
	_ = page.CreateElement(drawio.ElementData{
		ID: "useradded", CellID: "test--useradded", Kind: "container", Title: "User Added",
	}, "")

	elemFilter := map[string]bool{"a": true}
	flat := map[string]*model.Element{
		"a": {Kind: "container", Title: "A"},
		"b": {Kind: "container", Title: "B"},
	}

	result := &ForwardResult{}
	reconcileViewPage(page, elemFilter, flat, "", "test", result)

	// "a" is in the filter — should be kept.
	if page.FindElement("a") == nil {
		t.Error("element 'a' should be preserved (in view filter)")
	}

	// "b" is in the model but not in the filter — should be deleted.
	if page.FindElement("b") != nil {
		t.Error("element 'b' should be deleted (in model, not in view filter)")
	}

	// "useradded" is NOT in the model — should be preserved (user-added).
	if page.FindElement("useradded") == nil {
		t.Error("element 'useradded' should be preserved (not in model, user-added in draw.io)")
	}

	if result.ElementsDeleted != 1 {
		t.Errorf("expected 1 element deleted, got %d", result.ElementsDeleted)
	}
}

// TestApplyForward_EmptyModelRemovesElements verifies that when the model is
// emptied (no elements), forward sync removes all elements from the draw.io
// page, including in no-views (legacy) mode. Regression test for #110.
func TestApplyForward_EmptyModelRemovesElements(t *testing.T) {
	// Start with a document that has two elements on the page.
	doc := drawio.NewDocument()
	page := doc.AddPage("p1", "Page 1")
	_ = page.CreateElement(drawio.ElementData{
		ID: "a", CellID: "a", Kind: "container", Title: "A",
	}, "")
	_ = page.CreateElement(drawio.ElementData{
		ID: "b", CellID: "b", Kind: "container", Title: "B",
	}, "")

	// Empty model — no elements, no views.
	m := emptyModel()
	ts := minimalTemplates(t)

	// ChangeSet has explicit deletions for both elements.
	cs := &ChangeSet{
		ModelElementChanges: []ElementChange{
			{ID: "a", Type: Deleted},
			{ID: "b", Type: Deleted},
		},
	}

	result := ApplyForward(cs, doc, ts, m)

	if result.ElementsDeleted != 2 {
		t.Errorf("expected 2 elements deleted, got %d", result.ElementsDeleted)
	}
	if page.FindElement("a") != nil {
		t.Error("element 'a' should have been removed")
	}
	if page.FindElement("b") != nil {
		t.Error("element 'b' should have been removed")
	}
}

// TestApplyForward_NoViewsReconciliation verifies that in no-views (legacy)
// mode, orphaned elements on the page are cleaned up even without explicit
// Deleted changes in the ChangeSet. This handles the case where sync state
// is missing or out of sync. Regression test for #110.
func TestApplyForward_NoViewsReconciliation(t *testing.T) {
	// Document has elements "a" and "orphan" on the page.
	doc := drawio.NewDocument()
	page := doc.AddPage("p1", "Page 1")
	_ = page.CreateElement(drawio.ElementData{
		ID: "a", CellID: "a", Kind: "container", Title: "A",
	}, "")
	_ = page.CreateElement(drawio.ElementData{
		ID: "orphan", CellID: "orphan", Kind: "container", Title: "Orphan",
	}, "")

	// Model only has element "a" — "orphan" is not in the model.
	// But there's NO Deleted entry in the ChangeSet (e.g., sync state lost).
	m := modelWithElem("a", "container", "A")
	ts := minimalTemplates(t)
	cs := &ChangeSet{} // No changes — but orphan should still be cleaned up.

	result := ApplyForward(cs, doc, ts, m)

	// "a" should be preserved.
	if page.FindElement("a") == nil {
		t.Error("element 'a' should be preserved (in model)")
	}

	// "orphan" should be removed — it's not in the model.
	if page.FindElement("orphan") != nil {
		t.Error("orphan element should be removed (not in model)")
	}

	if result.ElementsDeleted != 1 {
		t.Errorf("expected 1 element deleted (orphan), got %d", result.ElementsDeleted)
	}
}

// TestApplyForward_NoDuplicateConnectors verifies that when a connector
// already exists on the page, an Added relationship change does not create
// a duplicate. This happens when sync state is deleted and all relationships
// are treated as new. Regression test for #119.
func TestApplyForward_NoDuplicateConnectors(t *testing.T) {
	// Create a document with two elements and an existing connector.
	doc := drawio.NewDocument()
	page := doc.AddPage("p1", "Page 1")
	_ = page.CreateElement(drawio.ElementData{
		ID: "a", CellID: "a", Kind: "container", Title: "A",
	}, "")
	_ = page.CreateElement(drawio.ElementData{
		ID: "b", CellID: "b", Kind: "container", Title: "B",
	}, "")
	page.CreateConnector(drawio.ConnectorData{
		From: "a", To: "b", Label: "calls",
		SourceRef: "a", TargetRef: "b",
	}, "endArrow=block;")

	// Verify precondition: one connector exists.
	if len(page.FindAllConnectors()) != 1 {
		t.Fatalf("precondition: expected 1 connector, got %d", len(page.FindAllConnectors()))
	}

	m := fwdModelWithRel("a", "b")
	ts := minimalTemplates(t)

	// Simulate sync state deletion: all relationships appear as Added.
	cs := &ChangeSet{
		ModelRelationshipChanges: []RelationshipChange{
			{From: "a", To: "b", Type: Added, NewValue: "calls"},
		},
	}

	result := ApplyForward(cs, doc, ts, m)

	// Should NOT create a duplicate connector.
	connectors := page.FindAllConnectors()
	if len(connectors) != 1 {
		t.Errorf("expected 1 connector (no duplicate), got %d", len(connectors))
	}

	// Should report 0 connectors created (already exists).
	if result.ConnectorsCreated != 0 {
		t.Errorf("expected 0 connectors created, got %d", result.ConnectorsCreated)
	}
}

// TestApplyForward_SelfReferencingRelationship verifies that a relationship
// where From == To (self-referencing) creates a connector. The condition
// that skips from == to after liftEndpoint should not skip genuine
// self-references. Regression test for #111.
func TestApplyForward_SelfReferencingRelationship(t *testing.T) {
	doc := drawio.NewDocument()
	doc.AddPage("view-default", "Default")
	ts := minimalTemplates(t)
	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"api": {Kind: "container", Title: "API"},
		},
		Relationships: []model.Relationship{
			{From: "api", To: "api", Label: "calls self"},
		},
		Views: map[string]model.View{
			"default": {Title: "Default", Include: []string{"*"}},
		},
	}

	cs := &ChangeSet{
		ModelRelationshipChanges: []RelationshipChange{
			{From: "api", To: "api", Type: Added, NewValue: "calls self"},
		},
		ModelElementChanges: []ElementChange{
			{ID: "api", Type: Added},
		},
	}

	result := ApplyForward(cs, doc, ts, m)

	if result.ConnectorsCreated < 1 {
		t.Errorf("expected at least 1 connector for self-referencing relationship, got %d", result.ConnectorsCreated)
	}
}

// TestApplyForward_NoDuplicateElements verifies that when an element
// already exists on the page, an Added element change does not create
// a duplicate. This happens when sync state is deleted or reset to {}
// and all elements are treated as new. Regression test for #141.
func TestApplyForward_NoDuplicateElements(t *testing.T) {
	// Create a doc that already has element "api" on the page.
	doc := docWithElem("api", "API", "Go", "")
	ts := minimalTemplates(t)
	m := modelWithElem("api", "container", "API")

	// Simulate a sync state reset: model says "api" is Added (because
	// it's not in sync state) but the element already exists in the
	// drawio doc.
	cs := &ChangeSet{
		ModelElementChanges: []ElementChange{
			{ID: "api", Type: Added},
		},
	}

	result := ApplyForward(cs, doc, ts, m)

	// Should NOT create a duplicate — skip existing element.
	page := doc.Pages()[0]
	count := 0
	for _, el := range page.Root().SelectElements("object") {
		if el.SelectAttrValue("bausteinsicht_id", "") == "api" {
			count++
		}
	}
	if count > 1 {
		t.Errorf("expected at most 1 element with bausteinsicht_id='api', got %d", count)
	}

	// Should report 0 elements created (already exists).
	if result.ElementsCreated != 0 {
		t.Errorf("expected 0 elements created, got %d", result.ElementsCreated)
	}
}

// TestApplyForward_MultipleRelationshipsSamePair verifies that two
// relationships between the same pair of elements create two separate
// connectors with distinct IDs. Regression test for #142.
func TestApplyForward_MultipleRelationshipsSamePair(t *testing.T) {
	doc := emptyDoc()
	ts := minimalTemplates(t)
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

	cs := &ChangeSet{
		ModelElementChanges: []ElementChange{
			{ID: "a", Type: Added},
			{ID: "b", Type: Added},
		},
		ModelRelationshipChanges: []RelationshipChange{
			{From: "a", To: "b", Index: 0, Type: Added, NewValue: "uses"},
			{From: "a", To: "b", Index: 1, Type: Added, NewValue: "calls"},
		},
	}

	result := ApplyForward(cs, doc, ts, m)

	if result.ConnectorsCreated != 2 {
		t.Fatalf("expected 2 connectors created, got %d", result.ConnectorsCreated)
	}

	page := doc.Pages()[0]
	conns := page.FindAllConnectors()
	if len(conns) != 2 {
		t.Fatalf("expected 2 connectors on page, got %d", len(conns))
	}

	// Verify each connector has the correct label.
	conn0 := page.FindConnector("a", "b", 0)
	if conn0 == nil {
		t.Fatal("connector index=0 not found")
	}
	if got := conn0.SelectAttrValue("value", ""); got != "uses" {
		t.Errorf("connector 0 label = %q, want %q", got, "uses")
	}

	conn1 := page.FindConnector("a", "b", 1)
	if conn1 == nil {
		t.Fatal("connector index=1 not found")
	}
	if got := conn1.SelectAttrValue("value", ""); got != "calls" {
		t.Errorf("connector 1 label = %q, want %q", got, "calls")
	}
}

// TestApplyForward_MultipleRelsSamePairNoDuplicateConnectors verifies that
// when connectors already exist for multiple relationships between the same
// pair, forward sync does not create duplicates. Regression test for #142/#119.
func TestApplyForward_MultipleRelsSamePairNoDuplicateConnectors(t *testing.T) {
	doc := drawio.NewDocument()
	page := doc.AddPage("p1", "Page 1")
	_ = page.CreateElement(drawio.ElementData{
		ID: "a", CellID: "a", Kind: "container", Title: "A",
	}, "")
	_ = page.CreateElement(drawio.ElementData{
		ID: "b", CellID: "b", Kind: "container", Title: "B",
	}, "")
	// Pre-existing connectors.
	page.CreateConnector(drawio.ConnectorData{
		From: "a", To: "b", Label: "uses",
		SourceRef: "a", TargetRef: "b", Index: 0,
	}, "endArrow=block;")
	page.CreateConnector(drawio.ConnectorData{
		From: "a", To: "b", Label: "calls",
		SourceRef: "a", TargetRef: "b", Index: 1,
	}, "endArrow=block;")

	if len(page.FindAllConnectors()) != 2 {
		t.Fatalf("precondition: expected 2 connectors, got %d", len(page.FindAllConnectors()))
	}

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
	ts := minimalTemplates(t)

	// Simulate sync state deletion: both relationships appear as Added.
	cs := &ChangeSet{
		ModelRelationshipChanges: []RelationshipChange{
			{From: "a", To: "b", Index: 0, Type: Added, NewValue: "uses"},
			{From: "a", To: "b", Index: 1, Type: Added, NewValue: "calls"},
		},
	}

	result := ApplyForward(cs, doc, ts, m)

	if len(page.FindAllConnectors()) != 2 {
		t.Errorf("expected 2 connectors (no duplicates), got %d", len(page.FindAllConnectors()))
	}
	if result.ConnectorsCreated != 0 {
		t.Errorf("expected 0 connectors created, got %d", result.ConnectorsCreated)
	}
}
