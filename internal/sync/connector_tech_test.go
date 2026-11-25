package sync

import (
	"testing"

	"github.com/beevik/etree"
	"github.com/docToolchain/Bausteinsicht/internal/drawio"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

func techSyncModel(technology string) *model.BausteinsichtModel {
	return &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"shop":    {Kind: "container", Title: "Shop"},
			"payment": {Kind: "container", Title: "Payment"},
		},
		Relationships: []model.Relationship{
			{From: "shop", To: "payment", Label: "charges", Technology: technology},
		},
		Views: map[string]model.View{
			"ctx": {Title: "Context", Include: []string{"shop", "payment"}},
		},
	}
}

// stateOf mirrors what BuildState records after a sync.
func stateOf(m *model.BausteinsichtModel) *SyncState {
	s := emptyState()
	for id, e := range m.Model {
		s.Elements[id] = ElementState{Title: e.Title, Kind: e.Kind}
	}
	for i, r := range m.Relationships {
		s.Relationships = append(s.Relationships, RelationshipState{From: r.From, To: r.To, Index: i, Label: r.Label, Kind: r.Kind})
	}
	return s
}

// techLabels returns the technology label cells of a page by cell ID.
func techLabels(page *drawio.Page) map[string]*etree.Element {
	labels := map[string]*etree.Element{}
	for _, cell := range page.Root().SelectElements("mxCell") {
		if id := cell.SelectAttrValue("id", ""); len(id) > 5 && id[len(id)-5:] == "-tech" {
			labels[id] = cell
		}
	}
	return labels
}

// A relationship technology is shown on the connector as a second label,
// "[technology]", and kept in sync when it changes (#642). The connector's own
// label is not touched, so reverse sync still reads only the label.
func TestSync_RelationshipTechnologyIsShownOnTheConnector(t *testing.T) {
	ts := minimalTemplates(t)
	doc := drawio.NewDocument()
	doc.AddPage("view-ctx", "Context")
	page := doc.GetPage("view-ctx")

	m := techSyncModel("HTTPS/JSON")
	Run(m, doc, emptyState(), ts, nil)

	connectors := page.FindAllConnectors()
	if len(connectors) != 1 {
		t.Fatalf("want 1 connector, got %d", len(connectors))
	}
	conn := connectors[0]
	if conn.SelectAttrValue("value", "") != "charges" {
		t.Errorf("the connector label must stay the plain label, got %q", conn.SelectAttrValue("value", ""))
	}
	labels := techLabels(page)
	label, ok := labels[conn.SelectAttrValue("id", "")+"-tech"]
	if !ok || len(labels) != 1 {
		t.Fatalf("expected exactly one technology label for the connector, got %v", labels)
	}
	if label.SelectAttrValue("value", "") != "[HTTPS/JSON]" {
		t.Errorf("label value = %q, want [HTTPS/JSON]", label.SelectAttrValue("value", ""))
	}
	if label.SelectAttrValue("parent", "") != conn.SelectAttrValue("id", "") || label.SelectAttrValue("vertex", "") != "1" {
		t.Errorf("the label must be a vertex child of the connector: %v", label.Attr)
	}

	// Changing the technology updates the label in place.
	m = techSyncModel("gRPC")
	Run(m, doc, stateOf(techSyncModel("HTTPS/JSON")), ts, nil)
	if got := techLabels(page); len(got) != 1 || got[conn.SelectAttrValue("id", "")+"-tech"].SelectAttrValue("value", "") != "[gRPC]" {
		t.Errorf("technology change not shown: %v", got)
	}

	// Removing it removes the label.
	m = techSyncModel("")
	Run(m, doc, stateOf(techSyncModel("gRPC")), ts, nil)
	if got := techLabels(page); len(got) != 0 {
		t.Errorf("label must go when the technology is removed: %v", got)
	}
}

func TestSync_TechnologyLabelDoesNotCauseChanges(t *testing.T) {
	ts := minimalTemplates(t)
	doc := drawio.NewDocument()
	doc.AddPage("view-ctx", "Context")
	m := techSyncModel("HTTPS/JSON")
	Run(m, doc, emptyState(), ts, nil)

	r := Run(m, doc, stateOf(m), ts, nil)
	if r.Forward.ConnectorsCreated+r.Forward.ConnectorsUpdated+r.Forward.ConnectorsDeleted != 0 ||
		r.Reverse.RelationshipsCreated+r.Reverse.RelationshipsDeleted != 0 ||
		r.Reverse.ElementsCreated+r.Reverse.ElementsDeleted != 0 {
		t.Errorf("a second sync must be a no-op: forward=%+v reverse=%+v", r.Forward, r.Reverse)
	}
	if len(r.Changes.DrawioElementChanges) != 0 || len(r.Changes.DrawioRelationshipChanges) != 0 {
		t.Errorf("the label cell must not look like a user change: %+v", r.Changes)
	}
}

// When the relationship goes away, its technology label must not stay behind.
func TestSync_TechnologyLabelOfADeletedRelationshipIsRemoved(t *testing.T) {
	ts := minimalTemplates(t)
	doc := drawio.NewDocument()
	doc.AddPage("view-ctx", "Context")
	page := doc.GetPage("view-ctx")
	m := techSyncModel("HTTPS/JSON")
	Run(m, doc, emptyState(), ts, nil)
	if len(techLabels(page)) != 1 {
		t.Fatal("setup: label missing")
	}

	m.Relationships = nil
	Run(m, doc, stateOf(techSyncModel("HTTPS/JSON")), ts, nil)
	if len(page.FindAllConnectors()) != 0 {
		t.Fatalf("setup: connector should be gone")
	}
	if got := techLabels(page); len(got) != 0 {
		t.Errorf("orphaned technology label left behind: %v", got)
	}
}

// If a relationship is inserted before it, the connector gets a new index (a
// new cell ID); the label follows it instead of being left on the old one.
func TestSync_TechnologyLabelFollowsARekeyedConnector(t *testing.T) {
	ts := minimalTemplates(t)
	doc := drawio.NewDocument()
	doc.AddPage("view-ctx", "Context")
	page := doc.GetPage("view-ctx")
	m := techSyncModel("HTTPS/JSON")
	Run(m, doc, emptyState(), ts, nil)

	m.Model["mail"] = model.Element{Kind: "container", Title: "Mail"}
	m.Views["ctx"] = model.View{Title: "Context", Include: []string{"shop", "payment", "mail"}}
	m.Relationships = append([]model.Relationship{{From: "shop", To: "mail", Label: "notifies"}}, m.Relationships...)
	Run(m, doc, stateOf(techSyncModel("HTTPS/JSON")), ts, nil)

	labels := techLabels(page)
	if len(labels) != 1 {
		t.Fatalf("want exactly one technology label, got %v", labels)
	}
	for id, label := range labels {
		parent := label.SelectAttrValue("parent", "")
		if id != parent+"-tech" || page.Root().FindElement("mxCell[@id='"+parent+"']") == nil {
			t.Errorf("label %q hangs on %q, which is not a connector of the page", id, parent)
		}
	}
}
