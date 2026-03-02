package drawio_test

import (
	"testing"

	"github.com/docToolchain/Bauteinsicht/internal/drawio"
)

const testStyle = "edgeStyle=orthogonalEdgeStyle;rounded=1;html=1;"

func newTestPage(t *testing.T) *drawio.Page {
	t.Helper()
	doc := drawio.NewDocument()
	return doc.AddPage("p1", "Page 1")
}

func TestCreateConnector(t *testing.T) {
	p := newTestPage(t)
	data := drawio.ConnectorData{From: "customer", To: "webshop.api", Label: "uses"}
	p.CreateConnector(data, testStyle)

	root := p.Root()
	cells := root.SelectElements("mxCell")

	// base cells (id=0, id=1) + connector = 3
	if len(cells) != 3 {
		t.Fatalf("expected 3 mxCells, got %d", len(cells))
	}

	conn := cells[2]
	if got := conn.SelectAttrValue("id", ""); got != "rel-customer-webshop.api-0" {
		t.Errorf("id = %q, want %q", got, "rel-customer-webshop.api-0")
	}
	if got := conn.SelectAttrValue("value", ""); got != "uses" {
		t.Errorf("value = %q, want %q", got, "uses")
	}
	if got := conn.SelectAttrValue("edge", ""); got != "1" {
		t.Errorf("edge = %q, want %q", got, "1")
	}
	if got := conn.SelectAttrValue("source", ""); got != "customer" {
		t.Errorf("source = %q, want %q", got, "customer")
	}
	if got := conn.SelectAttrValue("target", ""); got != "webshop.api" {
		t.Errorf("target = %q, want %q", got, "webshop.api")
	}
	if got := conn.SelectAttrValue("parent", ""); got != "1" {
		t.Errorf("parent = %q, want %q", got, "1")
	}
	if got := conn.SelectAttrValue("style", ""); got != testStyle {
		t.Errorf("style = %q, want %q", got, testStyle)
	}

	geom := conn.FindElement("mxGeometry")
	if geom == nil {
		t.Fatal("expected child mxGeometry element")
	}
	if got := geom.SelectAttrValue("relative", ""); got != "1" {
		t.Errorf("mxGeometry relative = %q, want %q", got, "1")
	}
	if got := geom.SelectAttrValue("as", ""); got != "geometry" {
		t.Errorf("mxGeometry as = %q, want %q", got, "geometry")
	}
}

func TestFindConnector(t *testing.T) {
	p := newTestPage(t)
	data := drawio.ConnectorData{From: "a", To: "b", Label: "link"}
	p.CreateConnector(data, testStyle)

	got := p.FindConnector("a", "b", 0)
	if got == nil {
		t.Fatal("FindConnector returned nil, expected element")
	}
	if got.SelectAttrValue("id", "") != "rel-a-b-0" {
		t.Errorf("unexpected id: %q", got.SelectAttrValue("id", ""))
	}

	if p.FindConnector("x", "y", 0) != nil {
		t.Error("FindConnector should return nil for non-existent connector")
	}
}

func TestFindAllConnectors(t *testing.T) {
	p := newTestPage(t)
	p.CreateConnector(drawio.ConnectorData{From: "a", To: "b"}, testStyle)
	p.CreateConnector(drawio.ConnectorData{From: "b", To: "c"}, testStyle)

	all := p.FindAllConnectors()
	if len(all) != 2 {
		t.Errorf("FindAllConnectors: got %d, want 2", len(all))
	}
}

func TestUpdateConnectorLabel(t *testing.T) {
	p := newTestPage(t)
	p.CreateConnector(drawio.ConnectorData{From: "x", To: "y", Label: "old"}, testStyle)

	p.UpdateConnectorLabel("x", "y", 0, "new")

	conn := p.FindConnector("x", "y", 0)
	if conn == nil {
		t.Fatal("connector not found after update")
	}
	if got := conn.SelectAttrValue("value", ""); got != "new" {
		t.Errorf("label = %q, want %q", got, "new")
	}
}

func TestDeleteConnector(t *testing.T) {
	p := newTestPage(t)
	p.CreateConnector(drawio.ConnectorData{From: "a", To: "b"}, testStyle)
	p.CreateConnector(drawio.ConnectorData{From: "c", To: "d"}, testStyle)

	p.DeleteConnector("a", "b", 0)

	if p.FindConnector("a", "b", 0) != nil {
		t.Error("deleted connector still found")
	}
	if p.FindConnector("c", "d", 0) == nil {
		t.Error("unrelated connector was removed")
	}
}

func TestDeleteConnectorsFor(t *testing.T) {
	p := newTestPage(t)
	// node1 is source
	p.CreateConnector(drawio.ConnectorData{From: "node1", To: "node2"}, testStyle)
	// node1 is target
	p.CreateConnector(drawio.ConnectorData{From: "node3", To: "node1"}, testStyle)
	// unrelated
	p.CreateConnector(drawio.ConnectorData{From: "nodeA", To: "nodeB"}, testStyle)

	p.DeleteConnectorsFor("node1")

	if p.FindConnector("node1", "node2", 0) != nil {
		t.Error("source connector should have been deleted")
	}
	if p.FindConnector("node3", "node1", 0) != nil {
		t.Error("target connector should have been deleted")
	}
	if p.FindConnector("nodeA", "nodeB", 0) == nil {
		t.Error("unrelated connector should remain")
	}
}

// TestMultipleConnectorsSamePair verifies that multiple connectors between the
// same pair of elements are stored separately using distinct indices. (#142)
func TestMultipleConnectorsSamePair(t *testing.T) {
	p := newTestPage(t)

	// Create two connectors between A and B with different indices.
	p.CreateConnector(drawio.ConnectorData{
		From: "a", To: "b", Label: "uses", Index: 0,
	}, testStyle)
	p.CreateConnector(drawio.ConnectorData{
		From: "a", To: "b", Label: "calls", Index: 1,
	}, testStyle)

	all := p.FindAllConnectors()
	if len(all) != 2 {
		t.Fatalf("expected 2 connectors, got %d", len(all))
	}

	// Verify each connector has a distinct ID.
	conn0 := p.FindConnector("a", "b", 0)
	if conn0 == nil {
		t.Fatal("connector index=0 not found")
	}
	if got := conn0.SelectAttrValue("id", ""); got != "rel-a-b-0" {
		t.Errorf("connector 0 id = %q, want %q", got, "rel-a-b-0")
	}
	if got := conn0.SelectAttrValue("value", ""); got != "uses" {
		t.Errorf("connector 0 value = %q, want %q", got, "uses")
	}

	conn1 := p.FindConnector("a", "b", 1)
	if conn1 == nil {
		t.Fatal("connector index=1 not found")
	}
	if got := conn1.SelectAttrValue("id", ""); got != "rel-a-b-1" {
		t.Errorf("connector 1 id = %q, want %q", got, "rel-a-b-1")
	}
	if got := conn1.SelectAttrValue("value", ""); got != "calls" {
		t.Errorf("connector 1 value = %q, want %q", got, "calls")
	}

	// Delete only index=0; index=1 should remain.
	p.DeleteConnector("a", "b", 0)
	if p.FindConnector("a", "b", 0) != nil {
		t.Error("connector index=0 should be deleted")
	}
	if p.FindConnector("a", "b", 1) == nil {
		t.Error("connector index=1 should remain after deleting index=0")
	}
}
