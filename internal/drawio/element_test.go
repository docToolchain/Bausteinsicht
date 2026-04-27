package drawio

import (
	"strings"
	"testing"

	"github.com/beevik/etree"
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

	expectedLabel := GenerateLabel("API Gateway", "Spring Boot", "Handles all API requests")
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
	if got := cell.SelectAttrValue("style", ""); got != "rounded=1;html=1;" {
		t.Errorf("style: got %q, want %q", got, "rounded=1;html=1;")
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

	expectedLabel := GenerateLabel("New Title", "MySQL", "New description")
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

// --- Sub-cell tests ---

func testSubCellTemplates() *SubCellTemplates {
	return &SubCellTemplates{
		Title: &SubCellStyle{
			Style: "text;html=1;fontSize=14;fontStyle=1;fontColor=#ffffff;fillColor=none;strokeColor=none;align=center;verticalAlign=middle;movable=0;resizable=0;deletable=0;editable=0;",
			X:     0, Y: 20, Width: 240, Height: 30,
		},
		Tech: &SubCellStyle{
			Style: "text;html=1;fontSize=11;fontStyle=2;fontColor=#CCCCCC;fillColor=none;strokeColor=none;align=center;verticalAlign=middle;movable=0;resizable=0;deletable=0;editable=0;",
			X:     0, Y: 55, Width: 240, Height: 20,
		},
		Desc: &SubCellStyle{
			Style: "text;html=1;fontSize=10;fontColor=#BBBBBB;fillColor=none;strokeColor=none;align=center;verticalAlign=middle;movable=0;resizable=0;deletable=0;editable=0;",
			X:     0, Y: 80, Width: 240, Height: 40,
		},
	}
}

func TestCreateElementWithSubCells(t *testing.T) {
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
		Width:       240,
		Height:      150,
		SubCells:    testSubCellTemplates(),
	}
	err := page.CreateElement(data, "rounded=1;container=1;")
	if err != nil {
		t.Fatalf("CreateElement error: %v", err)
	}

	obj := page.FindElement("webshop.api")
	if obj == nil {
		t.Fatal("expected to find created element, got nil")
	}

	// Label should be empty when using sub-cells.
	if got := obj.SelectAttrValue("label", ""); got != "" {
		t.Errorf("label: got %q, want empty", got)
	}

	// Check child text cells exist.
	root := page.Root()
	titleCell := findCellByID(root, "webshop.api-title")
	if titleCell == nil {
		t.Fatal("expected title sub-cell, got nil")
	}
	if got := titleCell.SelectAttrValue("value", ""); got != "API Gateway" {
		t.Errorf("title value: got %q, want %q", got, "API Gateway")
	}
	if got := titleCell.SelectAttrValue("parent", ""); got != "webshop.api" {
		t.Errorf("title parent: got %q, want %q", got, "webshop.api")
	}

	techCell := findCellByID(root, "webshop.api-tech")
	if techCell == nil {
		t.Fatal("expected tech sub-cell, got nil")
	}
	if got := techCell.SelectAttrValue("value", ""); got != "[Spring Boot]" {
		t.Errorf("tech value: got %q, want %q", got, "[Spring Boot]")
	}

	descCell := findCellByID(root, "webshop.api-desc")
	if descCell == nil {
		t.Fatal("expected desc sub-cell, got nil")
	}
	if got := descCell.SelectAttrValue("value", ""); got != "Handles all API requests" {
		t.Errorf("desc value: got %q, want %q", got, "Handles all API requests")
	}
}

func TestCreateElementWithSubCells_NoTechNoDesc(t *testing.T) {
	page := newInternalTestPage(t)
	data := ElementData{
		ID:       "customer",
		Kind:     "actor",
		Title:    "Customer",
		ParentID: "1",
		Width:    110,
		Height:   130,
		SubCells: testSubCellTemplates(),
	}
	err := page.CreateElement(data, "shape=mxgraph.c4.person2;container=1;")
	if err != nil {
		t.Fatalf("CreateElement error: %v", err)
	}

	root := page.Root()
	titleCell := findCellByID(root, "customer-title")
	if titleCell == nil {
		t.Fatal("expected title sub-cell")
	}

	// Tech and desc should NOT be created when empty.
	if findCellByID(root, "customer-tech") != nil {
		t.Error("tech sub-cell should not be created for empty technology")
	}
	if findCellByID(root, "customer-desc") != nil {
		t.Error("desc sub-cell should not be created for empty description")
	}
}

func TestUpdateElementWithSubCells(t *testing.T) {
	page := newInternalTestPage(t)
	data := ElementData{
		ID:          "svc.db",
		Kind:        "container",
		Title:       "Old Title",
		Technology:  "Postgres",
		Description: "Old description",
		ParentID:    "1",
		Width:       240,
		Height:      150,
		SubCells:    testSubCellTemplates(),
	}
	if err := page.CreateElement(data, "rounded=1;container=1;"); err != nil {
		t.Fatalf("CreateElement: %v", err)
	}

	updated := ElementData{
		Title:       "New Title",
		Technology:  "MySQL",
		Description: "New description",
	}
	page.UpdateElement("svc.db", updated)

	root := page.Root()
	titleCell := findCellByID(root, "svc.db-title")
	if titleCell == nil {
		t.Fatal("title sub-cell not found after update")
	}
	if got := titleCell.SelectAttrValue("value", ""); got != "New Title" {
		t.Errorf("title value: got %q, want %q", got, "New Title")
	}

	techCell := findCellByID(root, "svc.db-tech")
	if techCell == nil {
		t.Fatal("tech sub-cell not found after update")
	}
	if got := techCell.SelectAttrValue("value", ""); got != "[MySQL]" {
		t.Errorf("tech value: got %q, want %q", got, "[MySQL]")
	}

	descCell := findCellByID(root, "svc.db-desc")
	if descCell == nil {
		t.Fatal("desc sub-cell not found after update")
	}
	if got := descCell.SelectAttrValue("value", ""); got != "New description" {
		t.Errorf("desc value: got %q, want %q", got, "New description")
	}
}

func TestUpdateElementWithSubCells_RemoveTech(t *testing.T) {
	page := newInternalTestPage(t)
	data := ElementData{
		ID:         "svc",
		Kind:       "container",
		Title:      "Service",
		Technology: "Go",
		ParentID:   "1",
		Width:      240,
		Height:     150,
		SubCells:   testSubCellTemplates(),
	}
	if err := page.CreateElement(data, "rounded=1;container=1;"); err != nil {
		t.Fatalf("CreateElement: %v", err)
	}

	// Update with empty technology — should remove tech sub-cell.
	page.UpdateElement("svc", ElementData{Title: "Service", Technology: ""})

	root := page.Root()
	if findCellByID(root, "svc-tech") != nil {
		t.Error("tech sub-cell should have been removed when technology is empty")
	}
}

func TestDeleteElementRemovesSubCells(t *testing.T) {
	page := newInternalTestPage(t)
	data := ElementData{
		ID:          "del.me",
		Kind:        "container",
		Title:       "Delete Me",
		Technology:  "Go",
		Description: "To be deleted",
		ParentID:    "1",
		Width:       240,
		Height:      150,
		SubCells:    testSubCellTemplates(),
	}
	if err := page.CreateElement(data, "rounded=1;container=1;"); err != nil {
		t.Fatalf("CreateElement: %v", err)
	}

	page.DeleteElement("del.me")

	root := page.Root()
	if findCellByID(root, "del.me-title") != nil {
		t.Error("title sub-cell should have been removed on delete")
	}
	if findCellByID(root, "del.me-tech") != nil {
		t.Error("tech sub-cell should have been removed on delete")
	}
	if findCellByID(root, "del.me-desc") != nil {
		t.Error("desc sub-cell should have been removed on delete")
	}
}

func TestReadElementFields_SubCells(t *testing.T) {
	page := newInternalTestPage(t)
	data := ElementData{
		ID:          "api",
		Kind:        "container",
		Title:       "API Gateway",
		Technology:  "Spring Boot",
		Description: "REST API",
		ParentID:    "1",
		Width:       240,
		Height:      150,
		SubCells:    testSubCellTemplates(),
	}
	if err := page.CreateElement(data, "rounded=1;container=1;"); err != nil {
		t.Fatalf("CreateElement: %v", err)
	}

	obj := page.FindElement("api")
	if obj == nil {
		t.Fatal("element not found")
	}

	title, tech, desc := page.ReadElementFields(obj)
	if title != "API Gateway" {
		t.Errorf("title: got %q, want %q", title, "API Gateway")
	}
	if tech != "Spring Boot" {
		t.Errorf("technology: got %q, want %q", tech, "Spring Boot")
	}
	if desc != "REST API" {
		t.Errorf("description: got %q, want %q", desc, "REST API")
	}
}

func TestReadElementFields_HTMLFallback(t *testing.T) {
	page := newInternalTestPage(t)
	// Create element without sub-cells (legacy HTML label).
	data := ElementData{
		ID:         "legacy",
		Kind:       "system",
		Title:      "Legacy System",
		Technology: "COBOL",
		ParentID:   "1",
	}
	if err := page.CreateElement(data, "rounded=1;"); err != nil {
		t.Fatalf("CreateElement: %v", err)
	}

	obj := page.FindElement("legacy")
	if obj == nil {
		t.Fatal("element not found")
	}

	title, tech, desc := page.ReadElementFields(obj)
	if title != "Legacy System" {
		t.Errorf("title: got %q, want %q", title, "Legacy System")
	}
	if tech != "COBOL" {
		t.Errorf("technology: got %q, want %q", tech, "COBOL")
	}
	if desc != "" {
		t.Errorf("description: got %q, want empty", desc)
	}
}

func TestFindAllElements_ExcludesSubCells(t *testing.T) {
	page := newInternalTestPage(t)
	data := ElementData{
		ID:          "elem",
		Kind:        "container",
		Title:       "Element",
		Technology:  "Go",
		Description: "Desc",
		ParentID:    "1",
		Width:       240,
		Height:      150,
		SubCells:    testSubCellTemplates(),
	}
	if err := page.CreateElement(data, "rounded=1;container=1;"); err != nil {
		t.Fatalf("CreateElement: %v", err)
	}

	elems := page.FindAllElements()
	if len(elems) != 1 {
		t.Errorf("FindAllElements: got %d elements, want 1 (sub-cells should not be returned)", len(elems))
	}
}

// TestCreateElementSubCells_OverflowHidden verifies that sub-cells always have
// overflow=hidden so that long text is clipped at the fixed boundary. (#307)
func TestCreateElementSubCells_OverflowHidden(t *testing.T) {
	page := newInternalTestPage(t)
	data := ElementData{
		ID:          "svc",
		Kind:        "container",
		Title:       "Service",
		Technology:  "Go",
		Description: "This is a very long description that would overflow the fixed sub-cell height if not clipped by overflow=hidden.",
		ParentID:    "1",
		Width:       240,
		Height:      150,
		SubCells:    testSubCellTemplates(),
	}
	if err := page.CreateElement(data, "rounded=1;container=1;"); err != nil {
		t.Fatalf("CreateElement: %v", err)
	}

	root := page.Root()
	for _, suffix := range []string{"-title", "-tech", "-desc"} {
		cell := findCellByID(root, "svc"+suffix)
		if cell == nil {
			t.Fatalf("sub-cell %s not found", suffix)
		}
		style := cell.SelectAttrValue("style", "")
		if !strings.Contains(style, "overflow=hidden") {
			t.Errorf("sub-cell %s: expected overflow=hidden in style, got %q", suffix, style)
		}
	}
}

// TestCreateElement_UnknownKindGetsHtml1 verifies that elements with an unknown kind
// (which receive an empty style fallback) still have html=1 injected so that draw.io
// renders the HTML label as rich text instead of raw markup. (#307)
func TestCreateElement_UnknownKindGetsHtml1(t *testing.T) {
	page := newInternalTestPage(t)
	data := ElementData{
		ID:          "bbcli",
		Kind:        "client_tool", // kind not in template → empty style fallback
		Title:       "bbcli",
		Technology:  "Go / Cobra",
		Description: "CLI tool",
		ParentID:    "1",
	}
	if err := page.CreateElement(data, ""); err != nil {
		t.Fatalf("CreateElement: %v", err)
	}

	obj := page.FindElement("bbcli")
	if obj == nil {
		t.Fatal("element not found")
	}
	cell := obj.FindElement("mxCell")
	if cell == nil {
		t.Fatal("mxCell not found")
	}
	style := cell.SelectAttrValue("style", "")
	if !strings.Contains(style, "html=1") {
		t.Errorf("expected html=1 in style, got %q", style)
	}
}

// TestCreateElement_ExistingHtml1NotDuplicated ensures html=1 is not appended twice
// when a template already contains it.
func TestCreateElement_ExistingHtml1NotDuplicated(t *testing.T) {
	page := newInternalTestPage(t)
	data := ElementData{
		ID:    "svc",
		Kind:  "container",
		Title: "Service",
	}
	if err := page.CreateElement(data, "rounded=1;html=1;"); err != nil {
		t.Fatalf("CreateElement: %v", err)
	}

	obj := page.FindElement("svc")
	if obj == nil {
		t.Fatal("element not found")
	}
	cell := obj.FindElement("mxCell")
	if cell == nil {
		t.Fatal("mxCell not found")
	}
	style := cell.SelectAttrValue("style", "")
	count := strings.Count(style, "html=1")
	if count != 1 {
		t.Errorf("expected html=1 to appear exactly once, got %d times in style %q", count, style)
	}
}

func findCellByID(root *etree.Element, id string) *etree.Element {
	if root == nil {
		return nil
	}
	for _, cell := range root.SelectElements("mxCell") {
		if cell.SelectAttrValue("id", "") == id {
			return cell
		}
	}
	return nil
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
