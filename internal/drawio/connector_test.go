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
	if got := conn.SelectAttrValue("id", ""); got != "rel-customer-webshop.api" {
		t.Errorf("id = %q, want %q", got, "rel-customer-webshop.api")
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

	got := p.FindConnector("a", "b")
	if got == nil {
		t.Fatal("FindConnector returned nil, expected element")
	}
	if got.SelectAttrValue("id", "") != "rel-a-b" {
		t.Errorf("unexpected id: %q", got.SelectAttrValue("id", ""))
	}

	if p.FindConnector("x", "y") != nil {
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

	p.UpdateConnectorLabel("x", "y", "new")

	conn := p.FindConnector("x", "y")
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

	p.DeleteConnector("a", "b")

	if p.FindConnector("a", "b") != nil {
		t.Error("deleted connector still found")
	}
	if p.FindConnector("c", "d") == nil {
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

	if p.FindConnector("node1", "node2") != nil {
		t.Error("source connector should have been deleted")
	}
	if p.FindConnector("node3", "node1") != nil {
		t.Error("target connector should have been deleted")
	}
	if p.FindConnector("nodeA", "nodeB") == nil {
		t.Error("unrelated connector should remain")
	}
}
