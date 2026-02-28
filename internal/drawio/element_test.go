package drawio

import (
	"strings"
	"testing"
)

func newInternalTestPage(t *testing.T) *Page {
	t.Helper()
	doc := NewDocument()
	page := doc.AddPage("test-page", "Test Page")
	return page
}

func TestCreateElement(t *testing.T) {
	page := newInternalTestPage(t)
	data := ElementData{
		ID:          "webshop.api",
		Kind:        "container",
		Title:       "API Gateway",
		Technology:  "Spring Boot",
		Description: "Handles all API requests",
		ParentID:    "1",
		X:           200,
		Y:           150,
		Width:       160,
		Height:      70,
	}
	err := page.CreateElement(data, "rounded=1;")
	if err != nil {
		t.Fatalf("CreateElement error: %v", err)
	}

	obj := page.FindElement("webshop.api")
	if obj == nil {
		t.Fatal("expected to find created element, got nil")
	}

	if got := obj.SelectAttrValue("bausteinsicht_id", ""); got != "webshop.api" {
		t.Errorf("bausteinsicht_id: got %q, want %q", got, "webshop.api")
	}
	if got := obj.SelectAttrValue("bausteinsicht_kind", ""); got != "container" {
		t.Errorf("bausteinsicht_kind: got %q, want %q", got, "container")
	}
	if got := obj.SelectAttrValue("tooltip", ""); got != "Handles all API requests" {
		t.Errorf("tooltip: got %q, want %q", got, "Handles all API requests")
	}

	expectedLabel := GenerateLabel("API Gateway", "Spring Boot")
	if got := obj.SelectAttrValue("label", ""); got != expectedLabel {
		t.Errorf("label: got %q, want %q", got, expectedLabel)
	}

	cell := obj.FindElement("mxCell")
	if cell == nil {
		t.Fatal("expected mxCell child")
	}
	if got := cell.SelectAttrValue("vertex", ""); got != "1" {
		t.Errorf("vertex: got %q, want %q", got, "1")
	}
	if got := cell.SelectAttrValue("parent", ""); got != "1" {
		t.Errorf("parent: got %q, want %q", got, "1")
	}
	if got := cell.SelectAttrValue("style", ""); got != "rounded=1;" {
		t.Errorf("style: got %q, want %q", got, "rounded=1;")
	}

	geom := cell.FindElement("mxGeometry")
	if geom == nil {
		t.Fatal("expected mxGeometry child")
	}
	if got := geom.SelectAttrValue("x", ""); got != "200" {
		t.Errorf("x: got %q, want %q", got, "200")
	}
	if got := geom.SelectAttrValue("y", ""); got != "150" {
		t.Errorf("y: got %q, want %q", got, "150")
	}
	if got := geom.SelectAttrValue("width", ""); got != "160" {
		t.Errorf("width: got %q, want %q", got, "160")
	}
	if got := geom.SelectAttrValue("height", ""); got != "70" {
		t.Errorf("height: got %q, want %q", got, "70")
	}
	if got := geom.SelectAttrValue("as", ""); got != "geometry" {
		t.Errorf("as: got %q, want %q", got, "geometry")
	}
}

func TestCreateElementWithLink(t *testing.T) {
	page := newInternalTestPage(t)
	data := ElementData{
		ID:       "webshop",
		Kind:     "system",
		Title:    "Webshop",
		Link:     "data:page/id,view-containers",
		ParentID: "1",
	}
	err := page.CreateElement(data, "")
	if err != nil {
		t.Fatalf("CreateElement error: %v", err)
	}

	obj := page.FindElement("webshop")
	if obj == nil {
		t.Fatal("expected to find created element, got nil")
	}
	if got := obj.SelectAttrValue("link", ""); got != "data:page/id,view-containers" {
		t.Errorf("link: got %q, want %q", got, "data:page/id,view-containers")
	}
}

func TestFindElementNotFound(t *testing.T) {
	page := newInternalTestPage(t)
	obj := page.FindElement("nonexistent")
	if obj != nil {
		t.Errorf("expected nil, got element")
	}
}

func TestFindAllElements(t *testing.T) {
	page := newInternalTestPage(t)

	for _, id := range []string{"a.one", "a.two", "a.three"} {
		data := ElementData{ID: id, Kind: "component", Title: id, ParentID: "1"}
		if err := page.CreateElement(data, ""); err != nil {
			t.Fatalf("CreateElement(%q): %v", id, err)
		}
	}

	elems := page.FindAllElements()
	if len(elems) != 3 {
		t.Errorf("FindAllElements: got %d elements, want 3", len(elems))
	}
}

func TestUpdateElement(t *testing.T) {
	page := newInternalTestPage(t)
	data := ElementData{
		ID:          "svc.db",
		Kind:        "database",
		Title:       "Old Title",
		Technology:  "Postgres",
		Description: "Old description",
		ParentID:    "1",
	}
	if err := page.CreateElement(data, ""); err != nil {
		t.Fatalf("CreateElement: %v", err)
	}

	updated := ElementData{
		ID:          "svc.db",
		Kind:        "database",
		Title:       "New Title",
		Technology:  "MySQL",
		Description: "New description",
		Link:        "data:page/id,db-detail",
		ParentID:    "1",
	}
	page.UpdateElement("svc.db", updated)

	obj := page.FindElement("svc.db")
	if obj == nil {
		t.Fatal("element not found after update")
	}

	expectedLabel := GenerateLabel("New Title", "MySQL")
	if got := obj.SelectAttrValue("label", ""); got != expectedLabel {
		t.Errorf("label after update: got %q, want %q", got, expectedLabel)
	}
	if got := obj.SelectAttrValue("tooltip", ""); got != "New description" {
		t.Errorf("tooltip after update: got %q, want %q", got, "New description")
	}
	if got := obj.SelectAttrValue("link", ""); got != "data:page/id,db-detail" {
		t.Errorf("link after update: got %q, want %q", got, "data:page/id,db-detail")
	}
}

func TestDeleteElement(t *testing.T) {
	page := newInternalTestPage(t)
	data := ElementData{ID: "del.me", Kind: "component", Title: "Delete Me", ParentID: "1"}
	if err := page.CreateElement(data, ""); err != nil {
		t.Fatalf("CreateElement: %v", err)
	}

	if page.FindElement("del.me") == nil {
		t.Fatal("element should exist before delete")
	}

	page.DeleteElement("del.me")

	if page.FindElement("del.me") != nil {
		t.Error("element should not exist after delete")
	}
}

func TestDeleteElementRemovesConnectors(t *testing.T) {
	page := newInternalTestPage(t)

	for _, id := range []string{"src.elem", "dst.elem"} {
		data := ElementData{ID: id, Kind: "component", Title: id, ParentID: "1"}
		if err := page.CreateElement(data, ""); err != nil {
			t.Fatalf("CreateElement(%q): %v", id, err)
		}
	}

	root := page.Root()
	if root == nil {
		t.Fatal("root is nil")
	}
	connector := root.CreateElement("mxCell")
	connector.CreateAttr("id", "conn1")
	connector.CreateAttr("edge", "1")
	connector.CreateAttr("source", "src.elem")
	connector.CreateAttr("target", "dst.elem")
	connector.CreateAttr("parent", "1")

	page.DeleteElement("src.elem")

	root = page.Root()
	for _, cell := range root.SelectElements("mxCell") {
		if strings.Contains(cell.SelectAttrValue("id", ""), "conn1") {
			t.Error("connector should have been removed when source element was deleted")
		}
	}
}

func TestCreateElementDefaultParent(t *testing.T) {
	page := newInternalTestPage(t)
	data := ElementData{
		ID:    "top.level",
		Kind:  "system",
		Title: "Top Level",
		// ParentID intentionally left empty -- should default to "1"
	}
	if err := page.CreateElement(data, ""); err != nil {
		t.Fatalf("CreateElement: %v", err)
	}

	obj := page.FindElement("top.level")
	if obj == nil {
		t.Fatal("element not found")
	}
	cell := obj.FindElement("mxCell")
	if cell == nil {
		t.Fatal("mxCell not found")
	}
	if got := cell.SelectAttrValue("parent", ""); got != "1" {
		t.Errorf("default parent: got %q, want %q", got, "1")
	}
}

func TestContainerChildParent(t *testing.T) {
	page := newInternalTestPage(t)

	container := ElementData{ID: "ws.backend", Kind: "container", Title: "Backend", ParentID: "1"}
	if err := page.CreateElement(container, "swimlane;container=1;"); err != nil {
		t.Fatalf("CreateElement container: %v", err)
	}

	child := ElementData{ID: "ws.backend.auth", Kind: "component", Title: "Auth Service", ParentID: "ws.backend"}
	if err := page.CreateElement(child, ""); err != nil {
		t.Fatalf("CreateElement child: %v", err)
	}

	childObj := page.FindElement("ws.backend.auth")
	if childObj == nil {
		t.Fatal("child element not found")
	}
	childCell := childObj.FindElement("mxCell")
	if childCell == nil {
		t.Fatal("child mxCell not found")
	}
	if got := childCell.SelectAttrValue("parent", ""); got != "ws.backend" {
		t.Errorf("child parent: got %q, want %q", got, "ws.backend")
	}
}
