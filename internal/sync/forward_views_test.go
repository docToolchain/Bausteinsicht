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
