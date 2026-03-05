package sync

import (
	"testing"

	"github.com/docToolchain/Bauteinsicht/internal/drawio"
	"github.com/docToolchain/Bauteinsicht/internal/model"
)

// TestApplyForward_ScopedIDs verifies that when an element appears on multiple
// pages, each page uses a page-scoped cell ID (e.g., "context--customer") while
// bausteinsicht_id remains the original element ID.
func TestApplyForward_ScopedIDs(t *testing.T) {
	doc := drawio.NewDocument()
	doc.AddPage("view-context", "System Context")
	doc.AddPage("view-containers", "Container View")
	ts := minimalTemplates(t)

	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"customer": {Kind: "container", Title: "Customer"},
			"shop": {Kind: "container", Title: "Shop", Children: map[string]model.Element{
				"api": {Kind: "container", Title: "API"},
			}},
		},
		Relationships: []model.Relationship{},
		Views: map[string]model.View{
			"context": {
				Title:   "System Context",
				Include: []string{"customer", "shop"},
			},
			"containers": {
				Title:   "Container View",
				Include: []string{"customer", "shop.api"},
			},
		},
	}

	cs := &ChangeSet{
		ModelElementChanges: []ElementChange{
			{ID: "customer", Type: Added},
			{ID: "shop", Type: Added},
			{ID: "shop.api", Type: Added},
		},
	}

	ApplyForward(cs, doc, ts, m)

	contextPage := requirePage(t, doc, "view-context")
	containerPage := requirePage(t, doc, "view-containers")

	// Both pages should have customer (found by bausteinsicht_id).
	if contextPage.FindElement("customer") == nil {
		t.Fatal("customer not found on context page")
	}
	if containerPage.FindElement("customer") == nil {
		t.Fatal("customer not found on container page")
	}

	// The cell IDs must be different (page-scoped) to avoid draw.io conflicts.
	contextCustomer := contextPage.FindElement("customer")
	containerCustomer := containerPage.FindElement("customer")

	contextCellID := contextCustomer.SelectAttrValue("id", "")
	containerCellID := containerCustomer.SelectAttrValue("id", "")

	if contextCellID == containerCellID {
		t.Errorf("cell IDs must differ across pages, both are %q", contextCellID)
	}

	// bausteinsicht_id should remain "customer" on both pages.
	if contextCustomer.SelectAttrValue("bausteinsicht_id", "") != "customer" {
		t.Error("bausteinsicht_id should be 'customer' on context page")
	}
	if containerCustomer.SelectAttrValue("bausteinsicht_id", "") != "customer" {
		t.Error("bausteinsicht_id should be 'customer' on container page")
	}
}

// TestApplyForward_ScopedConnectorRefs verifies that connectors reference
// page-scoped cell IDs for source and target.
func TestApplyForward_ScopedConnectorRefs(t *testing.T) {
	doc := drawio.NewDocument()
	doc.AddPage("view-containers", "Container View")
	ts := minimalTemplates(t)

	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"a": {Kind: "container", Title: "A"},
			"b": {Kind: "container", Title: "B"},
		},
		Relationships: []model.Relationship{
			{From: "a", To: "b", Label: "calls"},
		},
		Views: map[string]model.View{
			"containers": {
				Title:   "Container View",
				Include: []string{"a", "b"},
			},
		},
	}

	cs := &ChangeSet{
		ModelElementChanges: []ElementChange{
			{ID: "a", Type: Added},
			{ID: "b", Type: Added},
		},
		ModelRelationshipChanges: []RelationshipChange{
			{From: "a", To: "b", Type: Added, NewValue: "calls"},
		},
	}

	ApplyForward(cs, doc, ts, m)

	page := requirePage(t, doc, "view-containers")

	// Connector should reference page-scoped cell IDs.
	elemA := page.FindElement("a")
	elemB := page.FindElement("b")
	if elemA == nil || elemB == nil {
		t.Fatal("elements not found")
	}

	cellA := elemA.SelectAttrValue("id", "")
	cellB := elemB.SelectAttrValue("id", "")

	// Find the connector and check source/target match cell IDs.
	conns := page.FindAllConnectors()
	if len(conns) == 0 {
		t.Fatal("no connectors found")
	}

	conn := conns[0]
	src := conn.SelectAttrValue("source", "")
	tgt := conn.SelectAttrValue("target", "")

	if src != cellA {
		t.Errorf("connector source=%q, want cell ID %q", src, cellA)
	}
	if tgt != cellB {
		t.Errorf("connector target=%q, want cell ID %q", tgt, cellB)
	}
}
