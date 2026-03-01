package sync

import (
	"testing"

	"github.com/docToolchain/Bauteinsicht/internal/drawio"
	"github.com/docToolchain/Bauteinsicht/internal/model"
)

func findConnectorCount(page *drawio.Page) int {
	return len(page.FindAllConnectors())
}

// modelWithViews returns a model with two top-level elements, a child, and two views.
func modelWithViews() *model.BausteinsichtModel {
	return &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"customer": {Kind: "container", Title: "Customer"},
			"webshop": {Kind: "container", Title: "Webshop", Children: map[string]model.Element{
				"api": {Kind: "container", Title: "API"},
				"db":  {Kind: "container", Title: "Database"},
			}},
		},
		Relationships: []model.Relationship{
			{From: "customer", To: "webshop", Label: "uses"},
			{From: "webshop.api", To: "webshop.db", Label: "reads"},
		},
		Views: map[string]model.View{
			"context": {
				Title:   "System Context",
				Include: []string{"customer", "webshop"},
			},
			"containers": {
				Title:   "Container View",
				Scope:   "webshop",
				Include: []string{"customer", "webshop.*"},
			},
		},
	}
}

// docWithViewPages creates a document with pages matching the model's views.
func docWithViewPages() *drawio.Document {
	doc := drawio.NewDocument()
	doc.AddPage("view-context", "System Context")
	doc.AddPage("view-containers", "Container View")
	return doc
}

// TestApplyForward_ElementOnCorrectViewPage verifies that elements are placed
// on the page corresponding to their view, not all on the first page.
func TestApplyForward_ElementOnCorrectViewPage(t *testing.T) {
	doc := docWithViewPages()
	ts := minimalTemplates(t)
	m := modelWithViews()

	cs := &ChangeSet{
		ModelElementChanges: []ElementChange{
			{ID: "customer", Type: Added},
			{ID: "webshop", Type: Added},
			{ID: "webshop.api", Type: Added},
			{ID: "webshop.db", Type: Added},
		},
		ModelRelationshipChanges: []RelationshipChange{
			{From: "customer", To: "webshop", Type: Added, NewValue: "uses"},
			{From: "webshop.api", To: "webshop.db", Type: Added, NewValue: "reads"},
		},
	}

	ApplyForward(cs, doc, ts, m)

	// Context view should have customer and webshop.
	contextPage := doc.GetPage("view-context")
	if contextPage == nil {
		t.Fatal("context page not found")
	}
	if contextPage.FindElement("customer") == nil {
		t.Error("expected 'customer' on context page")
	}
	if contextPage.FindElement("webshop") == nil {
		t.Error("expected 'webshop' on context page")
	}

	// Context view should NOT have api or db (those are children, not in context includes).
	if contextPage.FindElement("webshop.api") != nil {
		t.Error("'webshop.api' should NOT be on context page")
	}
	if contextPage.FindElement("webshop.db") != nil {
		t.Error("'webshop.db' should NOT be on context page")
	}

	// Container view should have customer, api, db (from includes: customer + webshop.*).
	containerPage := doc.GetPage("view-containers")
	if containerPage == nil {
		t.Fatal("container page not found")
	}
	if containerPage.FindElement("customer") == nil {
		t.Error("expected 'customer' on container page")
	}
	if containerPage.FindElement("webshop.api") == nil {
		t.Error("expected 'webshop.api' on container page")
	}
	if containerPage.FindElement("webshop.db") == nil {
		t.Error("expected 'webshop.db' on container page")
	}
}

// TestApplyForward_RelationshipOnlyOnPageWithBothEndpoints verifies that
// connectors are only created on pages where both from and to elements exist.
func TestApplyForward_RelationshipOnlyOnPageWithBothEndpoints(t *testing.T) {
	doc := docWithViewPages()
	ts := minimalTemplates(t)
	m := modelWithViews()

	cs := &ChangeSet{
		ModelElementChanges: []ElementChange{
			{ID: "customer", Type: Added},
			{ID: "webshop", Type: Added},
			{ID: "webshop.api", Type: Added},
			{ID: "webshop.db", Type: Added},
		},
		ModelRelationshipChanges: []RelationshipChange{
			{From: "customer", To: "webshop", Type: Added, NewValue: "uses"},
			{From: "webshop.api", To: "webshop.db", Type: Added, NewValue: "reads"},
		},
	}

	ApplyForward(cs, doc, ts, m)

	// Context page: customer→webshop connector should exist (using scoped cell IDs).
	contextPage := doc.GetPage("view-context")
	if contextPage.FindConnector("context--customer", "context--webshop") == nil {
		t.Error("expected connector customer→webshop on context page")
	}

	// Context page: api→db connector should NOT exist (endpoints not on this page).
	if findConnectorCount(contextPage) > 1 {
		t.Error("context page should only have one connector")
	}

	// Container page: api→db connector should exist (using scoped cell IDs).
	containerPage := doc.GetPage("view-containers")
	if containerPage.FindConnector("containers--webshop.api", "containers--webshop.db") == nil {
		t.Error("expected connector api→db on container page")
	}
}

// TestApplyForward_RelationshipLifting verifies that relationships are lifted
// to parent elements when the original endpoint is not on the page.
// Example: customer → webshop.frontend should render as customer → webshop
// on the context page where only customer and webshop are present.
func TestApplyForward_RelationshipLifting(t *testing.T) {
	doc := drawio.NewDocument()
	doc.AddPage("view-context", "System Context")
	doc.AddPage("view-containers", "Container View")
	ts := minimalTemplates(t)

	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"customer": {Kind: "actor", Title: "Customer"},
			"webshop": {Kind: "system", Title: "Online Shop", Children: map[string]model.Element{
				"frontend": {Kind: "container", Title: "Frontend"},
				"api":      {Kind: "container", Title: "API"},
			}},
		},
		Relationships: []model.Relationship{
			{From: "customer", To: "webshop.frontend", Label: "uses"},
			{From: "webshop.frontend", To: "webshop.api", Label: "calls"},
		},
		Views: map[string]model.View{
			"context": {
				Title:   "System Context",
				Include: []string{"customer", "webshop"},
			},
			"containers": {
				Title:   "Container View",
				Scope:   "webshop",
				Include: []string{"customer", "webshop.*"},
			},
		},
	}

	cs := &ChangeSet{
		ModelElementChanges: []ElementChange{
			{ID: "customer", Type: Added},
			{ID: "webshop", Type: Added},
			{ID: "webshop.frontend", Type: Added},
			{ID: "webshop.api", Type: Added},
		},
		ModelRelationshipChanges: []RelationshipChange{
			{From: "customer", To: "webshop.frontend", Type: Added, NewValue: "uses"},
			{From: "webshop.frontend", To: "webshop.api", Type: Added, NewValue: "calls"},
		},
	}

	ApplyForward(cs, doc, ts, m)

	// Context page: customer → webshop.frontend should be lifted to customer → webshop
	contextPage := doc.GetPage("view-context")
	if contextPage == nil {
		t.Fatal("context page not found")
	}

	conns := contextPage.FindAllConnectors()
	if len(conns) == 0 {
		t.Fatal("expected at least one connector on context page (lifted relationship)")
	}

	// Should have a connector from customer to webshop (lifted from webshop.frontend)
	found := false
	for _, c := range conns {
		src := c.SelectAttrValue("source", "")
		tgt := c.SelectAttrValue("target", "")
		if src == "context--customer" && tgt == "context--webshop" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected lifted connector customer→webshop on context page")
	}

	// Container page: customer → webshop.frontend should NOT be lifted (frontend is on the page)
	containerPage := doc.GetPage("view-containers")
	if containerPage == nil {
		t.Fatal("container page not found")
	}
	if containerPage.FindConnector("containers--customer", "containers--webshop.frontend") == nil {
		t.Error("expected original connector customer→webshop.frontend on container page")
	}
}

// TestApplyForward_RelationshipLiftingDedup verifies that when multiple child
// relationships lift to the same parent pair, only one connector is created.
func TestApplyForward_RelationshipLiftingDedup(t *testing.T) {
	doc := drawio.NewDocument()
	doc.AddPage("view-context", "System Context")
	ts := minimalTemplates(t)

	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"a": {Kind: "system", Title: "A", Children: map[string]model.Element{
				"x": {Kind: "container", Title: "X"},
				"y": {Kind: "container", Title: "Y"},
			}},
			"b": {Kind: "system", Title: "B", Children: map[string]model.Element{
				"z": {Kind: "container", Title: "Z"},
			}},
		},
		Relationships: []model.Relationship{
			{From: "a.x", To: "b.z", Label: "calls"},
			{From: "a.y", To: "b.z", Label: "reads"},
		},
		Views: map[string]model.View{
			"context": {
				Title:   "System Context",
				Include: []string{"a", "b"},
			},
		},
	}

	cs := &ChangeSet{
		ModelElementChanges: []ElementChange{
			{ID: "a", Type: Added},
			{ID: "b", Type: Added},
			{ID: "a.x", Type: Added},
			{ID: "a.y", Type: Added},
			{ID: "b.z", Type: Added},
		},
		ModelRelationshipChanges: []RelationshipChange{
			{From: "a.x", To: "b.z", Type: Added, NewValue: "calls"},
			{From: "a.y", To: "b.z", Type: Added, NewValue: "reads"},
		},
	}

	ApplyForward(cs, doc, ts, m)

	contextPage := doc.GetPage("view-context")
	conns := contextPage.FindAllConnectors()

	// Both relationships lift to a→b, but only one connector should be created.
	if len(conns) != 1 {
		t.Errorf("expected 1 connector (deduped), got %d", len(conns))
	}
}

// TestApplyForward_ScopeBoundingBox verifies that the scope element of a view
// is rendered as a boundary/swimlane on the page, with children nested inside.
func TestApplyForward_ScopeBoundingBox(t *testing.T) {
	doc := drawio.NewDocument()
	doc.AddPage("view-containers", "Container View")
	ts := minimalTemplates(t)

	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"customer": {Kind: "actor", Title: "Customer"},
			"shop": {Kind: "system", Title: "Online Shop", Children: map[string]model.Element{
				"api": {Kind: "container", Title: "API", Technology: "Go"},
				"db":  {Kind: "container", Title: "Database", Technology: "PostgreSQL"},
			}},
		},
		Relationships: []model.Relationship{},
		Views: map[string]model.View{
			"containers": {
				Title:   "Container View",
				Scope:   "shop",
				Include: []string{"customer", "shop.*"},
			},
		},
	}

	cs := &ChangeSet{
		ModelElementChanges: []ElementChange{
			{ID: "customer", Type: Added},
			{ID: "shop.api", Type: Added},
			{ID: "shop.db", Type: Added},
		},
	}

	ApplyForward(cs, doc, ts, m)

	page := doc.GetPage("view-containers")
	if page == nil {
		t.Fatal("container page not found")
	}

	// The scope element "shop" should appear as a boundary.
	scopeElem := page.FindElement("shop")
	if scopeElem == nil {
		t.Fatal("expected scope element 'shop' as boundary on container page")
	}

	// It should have a bausteinsicht_kind attribute indicating it's a boundary.
	kind := scopeElem.SelectAttrValue("bausteinsicht_kind", "")
	if kind != "system_boundary" {
		t.Errorf("scope element kind: got %q, want %q", kind, "system_boundary")
	}

	// Children (api, db) should be parented to the scope boundary cell.
	scopeCellID := scopeElem.SelectAttrValue("id", "")
	apiElem := page.FindElement("shop.api")
	if apiElem == nil {
		t.Fatal("expected 'shop.api' on container page")
	}
	apiCell := apiElem.FindElement("mxCell")
	if apiCell == nil {
		t.Fatal("shop.api has no mxCell")
	}
	apiParent := apiCell.SelectAttrValue("parent", "")
	if apiParent != scopeCellID {
		t.Errorf("shop.api parent: got %q, want scope cell ID %q", apiParent, scopeCellID)
	}

	// customer should NOT be parented to the scope (it's external).
	custElem := page.FindElement("customer")
	if custElem == nil {
		t.Fatal("expected 'customer' on container page")
	}
	custCell := custElem.FindElement("mxCell")
	if custCell == nil {
		t.Fatal("customer has no mxCell")
	}
	custParent := custCell.SelectAttrValue("parent", "")
	if custParent == scopeCellID {
		t.Error("customer should NOT be parented to scope boundary")
	}
}

// TestApplyForward_DeletedElementRemovedFromViewPages verifies that when an
// element is deleted from the model, it is removed from all view pages where
// it previously appeared. Regression test for #85.
func TestApplyForward_DeletedElementRemovedFromViewPages(t *testing.T) {
	ts := minimalTemplates(t)
	m := modelWithViews()

	// Round 1: add all elements to doc.
	doc := docWithViewPages()
	csAdd := &ChangeSet{
		ModelElementChanges: []ElementChange{
			{ID: "customer", Type: Added},
			{ID: "webshop", Type: Added},
			{ID: "webshop.api", Type: Added},
			{ID: "webshop.db", Type: Added},
		},
		ModelRelationshipChanges: []RelationshipChange{
			{From: "customer", To: "webshop", Type: Added, NewValue: "uses"},
			{From: "webshop.api", To: "webshop.db", Type: Added, NewValue: "reads"},
		},
	}
	ApplyForward(csAdd, doc, ts, m)

	// Verify db exists on container page before deletion.
	containerPage := doc.GetPage("view-containers")
	if containerPage.FindElement("webshop.db") == nil {
		t.Fatal("precondition: webshop.db should exist on container page before deletion")
	}

	// Round 2: remove webshop.db from the model.
	delete(m.Model["webshop"].Children, "db")
	// Update relationships: remove the one referencing db.
	m.Relationships = []model.Relationship{
		{From: "customer", To: "webshop", Label: "uses"},
	}
	// Also update the containers view to exclude db.
	m.Views["containers"] = model.View{
		Title:   "Container View",
		Scope:   "webshop",
		Include: []string{"customer", "webshop.api"},
	}

	csDel := &ChangeSet{
		ModelElementChanges: []ElementChange{
			{ID: "webshop.db", Type: Deleted},
		},
		ModelRelationshipChanges: []RelationshipChange{
			{From: "webshop.api", To: "webshop.db", Type: Deleted},
		},
	}
	result := ApplyForward(csDel, doc, ts, m)

	// The element should be removed from the container page.
	if containerPage.FindElement("webshop.db") != nil {
		t.Error("webshop.db should be removed from container page after deletion")
	}

	if result.ElementsDeleted == 0 {
		t.Error("expected at least 1 element deleted in forward result")
	}
}

// TestApplyForward_DeletedRelationshipRemovedFromViewPages verifies that when
// a relationship is deleted from the model, its connector is removed from
// view pages. Regression test for #85.
func TestApplyForward_DeletedRelationshipRemovedFromViewPages(t *testing.T) {
	ts := minimalTemplates(t)
	m := modelWithViews()

	// Round 1: add all elements and relationships.
	doc := docWithViewPages()
	csAdd := &ChangeSet{
		ModelElementChanges: []ElementChange{
			{ID: "customer", Type: Added},
			{ID: "webshop", Type: Added},
			{ID: "webshop.api", Type: Added},
			{ID: "webshop.db", Type: Added},
		},
		ModelRelationshipChanges: []RelationshipChange{
			{From: "customer", To: "webshop", Type: Added, NewValue: "uses"},
			{From: "webshop.api", To: "webshop.db", Type: Added, NewValue: "reads"},
		},
	}
	ApplyForward(csAdd, doc, ts, m)

	// Verify connector exists before deletion.
	containerPage := doc.GetPage("view-containers")
	if containerPage.FindConnector("containers--webshop.api", "containers--webshop.db") == nil {
		t.Fatal("precondition: api→db connector should exist on container page")
	}

	// Round 2: delete the api→db relationship.
	m.Relationships = []model.Relationship{
		{From: "customer", To: "webshop", Label: "uses"},
	}

	csDel := &ChangeSet{
		ModelRelationshipChanges: []RelationshipChange{
			{From: "webshop.api", To: "webshop.db", Type: Deleted},
		},
	}
	result := ApplyForward(csDel, doc, ts, m)

	// The connector should be removed from the container page.
	if containerPage.FindConnector("containers--webshop.api", "containers--webshop.db") != nil {
		t.Error("api→db connector should be removed from container page after deletion")
	}

	if result.ConnectorsDeleted == 0 {
		t.Error("expected at least 1 connector deleted in forward result")
	}
}

// TestApplyForward_ScopeBoundaryUpdatedOnModify verifies that when a scope
// element's properties change, the scope boundary on its view page is also
// updated. Regression test for #84.
func TestApplyForward_ScopeBoundaryUpdatedOnModify(t *testing.T) {
	ts := minimalTemplates(t)
	m := modelWithViews()

	// Round 1: add all elements (creates scope boundary for "webshop" on containers page).
	doc := docWithViewPages()
	csAdd := &ChangeSet{
		ModelElementChanges: []ElementChange{
			{ID: "customer", Type: Added},
			{ID: "webshop", Type: Added},
			{ID: "webshop.api", Type: Added},
			{ID: "webshop.db", Type: Added},
		},
	}
	ApplyForward(csAdd, doc, ts, m)

	// Verify boundary exists with original title.
	containerPage := doc.GetPage("view-containers")
	boundary := containerPage.FindElement("webshop")
	if boundary == nil {
		t.Fatal("precondition: scope boundary for webshop should exist on containers page")
	}
	origLabel := boundary.SelectAttrValue("label", "")
	if origLabel == "" {
		t.Fatal("precondition: scope boundary should have a label")
	}

	// Round 2: modify the scope element's technology.
	m.Model["webshop"] = model.Element{
		Kind:       "container",
		Title:      "Webshop",
		Technology: "Kubernetes",
		Children: map[string]model.Element{
			"api": {Kind: "container", Title: "API"},
			"db":  {Kind: "container", Title: "Database"},
		},
	}

	csMod := &ChangeSet{
		ModelElementChanges: []ElementChange{
			{ID: "webshop", Type: Modified},
		},
	}
	result := ApplyForward(csMod, doc, ts, m)

	// The boundary label on the containers page should reflect the new technology.
	boundary = containerPage.FindElement("webshop")
	if boundary == nil {
		t.Fatal("scope boundary for webshop should still exist after modify")
	}
	newLabel := boundary.SelectAttrValue("label", "")
	expectedLabel := drawio.GenerateLabel("Webshop", "Kubernetes", "")
	if newLabel != expectedLabel {
		t.Errorf("scope boundary label not updated:\ngot:  %s\nwant: %s", newLabel, expectedLabel)
	}

	if result.ElementsUpdated == 0 {
		t.Error("expected at least 1 element updated in forward result")
	}
}

// TestExcludeRemovesElementFromPage verifies that when a view's exclude list
// changes to exclude an element that was previously on the page, the element
// and its connectors are removed during forward sync — even when the ChangeSet
// is empty (no model changes, only view configuration changes). Fix for #102.
func TestExcludeRemovesElementFromPage(t *testing.T) {
	ts := minimalTemplates(t)
	m := modelWithViews()

	// Round 1: add all elements and relationships.
	doc := docWithViewPages()
	csAdd := &ChangeSet{
		ModelElementChanges: []ElementChange{
			{ID: "customer", Type: Added},
			{ID: "webshop", Type: Added},
			{ID: "webshop.api", Type: Added},
			{ID: "webshop.db", Type: Added},
		},
		ModelRelationshipChanges: []RelationshipChange{
			{From: "customer", To: "webshop", Type: Added, NewValue: "uses"},
			{From: "webshop.api", To: "webshop.db", Type: Added, NewValue: "reads"},
		},
	}
	ApplyForward(csAdd, doc, ts, m)

	// Verify preconditions: webshop.db is on the container page with a connector.
	containerPage := doc.GetPage("view-containers")
	if containerPage == nil {
		t.Fatal("precondition: container page not found")
	}
	if containerPage.FindElement("webshop.db") == nil {
		t.Fatal("precondition: webshop.db should exist on container page")
	}
	if containerPage.FindConnector("containers--webshop.api", "containers--webshop.db") == nil {
		t.Fatal("precondition: api→db connector should exist on container page")
	}

	// Round 2: add webshop.db to the view's exclude list.
	// The element still exists in the model — only the view config changes.
	m.Views["containers"] = model.View{
		Title:   "Container View",
		Scope:   "webshop",
		Include: []string{"customer", "webshop.*"},
		Exclude: []string{"webshop.db"},
	}

	// Empty ChangeSet: no model changes, only view exclude changed.
	csEmpty := &ChangeSet{}
	result := ApplyForward(csEmpty, doc, ts, m)

	// The excluded element should be removed from the page.
	if containerPage.FindElement("webshop.db") != nil {
		t.Error("webshop.db should be removed from container page after being excluded from view")
	}

	// Connectors referencing the excluded element should also be removed.
	if containerPage.FindConnector("containers--webshop.api", "containers--webshop.db") != nil {
		t.Error("api→db connector should be removed after webshop.db is excluded from view")
	}

	// Result counters should reflect the removals.
	if result.ElementsDeleted == 0 {
		t.Error("expected at least 1 element deleted in forward result")
	}
	if result.ConnectorsDeleted == 0 {
		t.Error("expected at least 1 connector deleted in forward result")
	}

	// Elements that are still in the view should remain untouched.
	if containerPage.FindElement("customer") == nil {
		t.Error("customer should still be on container page")
	}
	if containerPage.FindElement("webshop.api") == nil {
		t.Error("webshop.api should still be on container page")
	}
}

// TestApplyForward_DeleteElementRemovesConnectors verifies that when an
// element is deleted from the model, any connectors referencing that element
// are also removed from the page — even when no explicit relationship deletion
// is in the ChangeSet. This prevents orphaned connectors that reverse sync
// would otherwise pick up as phantom relationships. Regression test for #101.
func TestApplyForward_DeleteElementRemovesConnectors(t *testing.T) {
	ts := minimalTemplates(t)
	m := modelWithViews()

	// Round 1: add all elements and relationships.
	doc := docWithViewPages()
	csAdd := &ChangeSet{
		ModelElementChanges: []ElementChange{
			{ID: "customer", Type: Added},
			{ID: "webshop", Type: Added},
			{ID: "webshop.api", Type: Added},
			{ID: "webshop.db", Type: Added},
		},
		ModelRelationshipChanges: []RelationshipChange{
			{From: "customer", To: "webshop", Type: Added, NewValue: "uses"},
			{From: "webshop.api", To: "webshop.db", Type: Added, NewValue: "reads"},
		},
	}
	ApplyForward(csAdd, doc, ts, m)

	// Verify preconditions on the container page.
	containerPage := doc.GetPage("view-containers")
	if containerPage.FindElement("webshop.db") == nil {
		t.Fatal("precondition: webshop.db should exist on container page")
	}
	if containerPage.FindConnector("containers--webshop.api", "containers--webshop.db") == nil {
		t.Fatal("precondition: api→db connector should exist on container page")
	}

	// Round 2: delete webshop.db from the model.
	// Note: we include the element deletion but intentionally omit the
	// explicit relationship deletion to test the orphan-cleanup path.
	delete(m.Model["webshop"].Children, "db")
	m.Relationships = []model.Relationship{
		{From: "customer", To: "webshop", Label: "uses"},
	}
	m.Views["containers"] = model.View{
		Title:   "Container View",
		Scope:   "webshop",
		Include: []string{"customer", "webshop.api"},
	}

	csDel := &ChangeSet{
		ModelElementChanges: []ElementChange{
			{ID: "webshop.db", Type: Deleted},
		},
		// No ModelRelationshipChanges — the element deletion should
		// clean up connectors referencing the deleted element.
	}
	result := ApplyForward(csDel, doc, ts, m)

	// The element should be removed.
	if containerPage.FindElement("webshop.db") != nil {
		t.Error("webshop.db should be removed from container page after deletion")
	}

	// The connector referencing the deleted element should also be removed.
	if containerPage.FindConnector("containers--webshop.api", "containers--webshop.db") != nil {
		t.Error("api→db connector should be removed when webshop.db is deleted")
	}

	// The customer→webshop connector on the context page should be unaffected.
	contextPage := doc.GetPage("view-context")
	if contextPage.FindConnector("context--customer", "context--webshop") == nil {
		t.Error("customer→webshop connector on context page should be unaffected")
	}

	if result.ElementsDeleted == 0 {
		t.Error("expected at least 1 element deleted in forward result")
	}
	if result.ConnectorsDeleted == 0 {
		t.Error("expected at least 1 connector deleted in forward result")
	}
}

// TestApplyForward_NoViewsFallback verifies backward compatibility:
// when no views are defined, elements go to the first page.
func TestApplyForward_NoViewsFallback(t *testing.T) {
	doc := emptyDoc()
	ts := minimalTemplates(t)
	m := modelWithElem("api", "container", "API")
	// m.Views is nil — should fall back to first page.

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
	if page.FindElement("api") == nil {
		t.Error("expected 'api' on first page when no views defined")
	}
}
