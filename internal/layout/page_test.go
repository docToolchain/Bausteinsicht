package layout

import (
	"strconv"
	"strings"
	"testing"

	"github.com/beevik/etree"
	"github.com/docToolchain/Bausteinsicht/internal/drawio"
)

// addElem adds an element cell. parent is the cell ID of the container ("" for
// the page); w/h are the starting size.
func addElem(t *testing.T, page *drawio.Page, id, parent string, w, h float64) {
	t.Helper()
	if err := page.CreateElement(drawio.ElementData{
		ID: id, CellID: id, ParentID: parent, Kind: "container", Title: id, X: 1, Y: 1, Width: w, Height: h,
	}, "rounded=1"); err != nil {
		t.Fatal(err)
	}
}

func addEdge(page *drawio.Page, from, to string) {
	page.CreateConnector(drawio.ConnectorData{From: from, To: to, SourceRef: from, TargetRef: to}, "")
}

type geo struct{ x, y, w, h float64 }

func geometry(t *testing.T, page *drawio.Page, id string) geo {
	t.Helper()
	obj := page.FindElement(id)
	if obj == nil {
		t.Fatalf("element %s not on the page", id)
	}
	g := obj.FindElement("mxCell/mxGeometry")
	if g == nil {
		t.Fatalf("no geometry for %s", id)
	}
	num := func(name string) float64 {
		v, err := strconv.ParseFloat(g.SelectAttrValue(name, "0"), 64)
		if err != nil {
			t.Fatalf("bad %s on %s", name, id)
		}
		return v
	}
	return geo{num("x"), num("y"), num("width"), num("height")}
}

func overlap(a, b geo) bool {
	return a.x < b.x+b.w && b.x < a.x+a.w && a.y < b.y+b.h && b.y < a.y+a.h
}

// Root cause 1 of #654: only the first page that contained an element was
// updated. Every page must be laid out.
func TestLayoutDocument_EveryPageIsLaidOut(t *testing.T) {
	doc := drawio.NewDocument()
	for _, id := range []string{"p1", "p2", "p3"} {
		page := doc.AddPage(id, id)
		addElem(t, page, id+"-a", "", 160, 60)
		addElem(t, page, id+"-b", "", 160, 60)
		addEdge(page, id+"-a", id+"-b")
	}
	report, err := LayoutDocument(doc, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if report.Pages != 3 || report.Elements != 6 {
		t.Errorf("report = %+v, want 3 pages and 6 elements", report)
	}
	for _, id := range []string{"p1", "p2", "p3"} {
		a, b := geometry(t, doc.GetPage(id), id+"-a"), geometry(t, doc.GetPage(id), id+"-b")
		if a.y >= b.y {
			t.Errorf("page %s: the source must be above the target, got a.y=%v b.y=%v", id, a.y, b.y)
		}
	}
}

// Root cause 2: the layout used every relationship of the model. Only the
// connectors on the page count.
func TestLayoutDocument_OnlyConnectorsOnThePageCount(t *testing.T) {
	doc := drawio.NewDocument()
	page := doc.AddPage("v", "v")
	for _, id := range []string{"a", "b", "c"} {
		addElem(t, page, id, "", 160, 60)
	}
	addEdge(page, "a", "b") // c has no connector on this page

	if _, err := LayoutDocument(doc, Options{}); err != nil {
		t.Fatal(err)
	}
	a, b, c := geometry(t, page, "a"), geometry(t, page, "b"), geometry(t, page, "c")
	if a.y >= b.y {
		t.Errorf("a must be above b: %v %v", a, b)
	}
	if c.y != a.y {
		t.Errorf("an unconnected element belongs on the first layer, got c.y=%v a.y=%v", c.y, a.y)
	}
}

// Root cause 3: every element got 160x60. Sizes stay.
func TestLayoutDocument_KeepsElementSizes(t *testing.T) {
	doc := drawio.NewDocument()
	page := doc.AddPage("v", "v")
	addElem(t, page, "person", "", 80, 120)
	addElem(t, page, "system", "", 300, 90)
	addEdge(page, "person", "system")
	if _, err := LayoutDocument(doc, Options{}); err != nil {
		t.Fatal(err)
	}
	if g := geometry(t, page, "person"); g.w != 80 || g.h != 120 {
		t.Errorf("person resized to %vx%v", g.w, g.h)
	}
	if g := geometry(t, page, "system"); g.w != 300 || g.h != 90 {
		t.Errorf("system resized to %vx%v", g.w, g.h)
	}
}

// Root cause 4: absolute coordinates put children outside their boundary.
func TestLayoutDocument_ChildrenAreRelativeAndInsideTheirBoundary(t *testing.T) {
	doc := drawio.NewDocument()
	page := doc.AddPage("v", "v")
	addElem(t, page, "user", "", 160, 60)
	addElem(t, page, "shop", "", 400, 300)
	for _, id := range []string{"shop-api", "shop-db", "shop-web"} {
		addElem(t, page, id, "shop", 160, 60)
	}
	addEdge(page, "user", "shop")
	addEdge(page, "shop-web", "shop-api")
	addEdge(page, "shop-api", "shop-db")

	if _, err := LayoutDocument(doc, Options{}); err != nil {
		t.Fatal(err)
	}
	shop := geometry(t, page, "shop")
	user := geometry(t, page, "user")
	if user.y >= shop.y {
		t.Errorf("the user (source of a connector into the boundary) must be above it: user=%v shop=%v", user, shop)
	}
	var children []geo
	for _, id := range []string{"shop-api", "shop-db", "shop-web"} {
		g := geometry(t, page, id)
		if g.x < 0 || g.y < 0 || g.x+g.w > shop.w || g.y+g.h > shop.h {
			t.Errorf("%s at %v is outside the boundary of size %vx%v (coordinates must be relative to it)", id, g, shop.w, shop.h)
		}
		children = append(children, g)
	}
	for i := range children {
		for j := i + 1; j < len(children); j++ {
			if overlap(children[i], children[j]) {
				t.Errorf("children overlap: %v and %v", children[i], children[j])
			}
		}
	}
	if web, api := geometry(t, page, "shop-web"), geometry(t, page, "shop-api"); web.y >= api.y {
		t.Errorf("the chain web -> api -> db must run downwards: web=%v api=%v", web, api)
	}
}

// Root cause 5: the order came from map iteration. The same input must give the
// same output, whatever order the cells were added in.
func TestLayoutDocument_IsDeterministic(t *testing.T) {
	build := func(ids []string) *drawio.Document {
		doc := drawio.NewDocument()
		page := doc.AddPage("v", "v")
		for _, id := range ids {
			addElem(t, page, id, "", 160, 60)
		}
		addEdge(page, "a", "c")
		addEdge(page, "b", "c")
		addEdge(page, "c", "d")
		return doc
	}
	render := func(doc *drawio.Document) string {
		var s string
		for _, id := range []string{"a", "b", "c", "d"} {
			g := geometry(t, doc.GetPage("v"), id)
			s += id + ":" + strconv.FormatFloat(g.x, 'f', 0, 64) + "," + strconv.FormatFloat(g.y, 'f', 0, 64) + ";"
		}
		return s
	}
	first := build([]string{"a", "b", "c", "d"})
	second := build([]string{"d", "c", "b", "a"})
	for _, doc := range []*drawio.Document{first, second} {
		if _, err := LayoutDocument(doc, Options{}); err != nil {
			t.Fatal(err)
		}
	}
	if render(first) != render(second) {
		t.Errorf("layout depends on the order of the cells:\n%s\n%s", render(first), render(second))
	}
}

func TestLayoutDocument_RankDirLR(t *testing.T) {
	doc := drawio.NewDocument()
	page := doc.AddPage("v", "v")
	addElem(t, page, "a", "", 160, 60)
	addElem(t, page, "b", "", 160, 60)
	addEdge(page, "a", "b")
	if _, err := LayoutDocument(doc, Options{RankDir: "LR"}); err != nil {
		t.Fatal(err)
	}
	a, b := geometry(t, page, "a"), geometry(t, page, "b")
	if a.x >= b.x || a.y != b.y {
		t.Errorf("with LR the target must be to the right on the same row: a=%v b=%v", a, b)
	}
}

func TestLayoutDocument_CycleDoesNotHang(t *testing.T) {
	doc := drawio.NewDocument()
	page := doc.AddPage("v", "v")
	for _, id := range []string{"a", "b", "c"} {
		addElem(t, page, id, "", 160, 60)
	}
	addEdge(page, "a", "b")
	addEdge(page, "b", "c")
	addEdge(page, "c", "a")
	if _, err := LayoutDocument(doc, Options{}); err != nil {
		t.Fatal(err)
	}
	if a, b, c := geometry(t, page, "a"), geometry(t, page, "b"), geometry(t, page, "c"); a.y >= b.y || b.y >= c.y {
		t.Errorf("a cycle is drawn as a chain: a=%v b=%v c=%v", a, b, c)
	}
}

func TestLayoutDocument_PinnedElementsStay(t *testing.T) {
	doc := drawio.NewDocument()
	page := doc.AddPage("v", "v")
	addElem(t, page, "a", "", 160, 60)
	addElem(t, page, "b", "", 160, 60)
	addEdge(page, "a", "b")
	page.FindElement("a").CreateAttr("bausteinsicht-pinned", "true")
	before := geometry(t, page, "a")

	report, err := LayoutDocument(doc, Options{PreservePinned: true})
	if err != nil {
		t.Fatal(err)
	}
	if after := geometry(t, page, "a"); after != before {
		t.Errorf("pinned element moved from %v to %v", before, after)
	}
	if report.Pinned != 1 || report.Elements != 1 {
		t.Errorf("report = %+v, want 1 pinned and 1 positioned", report)
	}

	// Without PreservePinned it is laid out like any other element.
	if _, err := LayoutDocument(doc, Options{}); err != nil {
		t.Fatal(err)
	}
	if after := geometry(t, page, "a"); after == before {
		t.Error("pinned element must move when PreservePinned is off")
	}
}

func TestLayoutDocument_ClearsWaypointsAndSelectsView(t *testing.T) {
	doc := drawio.NewDocument()
	one, two := doc.AddPage("one", "one"), doc.AddPage("two", "two")
	for _, page := range []*drawio.Page{one, two} {
		addElem(t, page, "a", "", 160, 60)
		addElem(t, page, "b", "", 160, 60)
		addEdge(page, "a", "b")
		geom := page.FindAllConnectors()[0].SelectElement("mxGeometry")
		arr := geom.CreateElement("Array")
		arr.CreateAttr("as", "points")
		arr.CreateElement("mxPoint").CreateAttr("x", "5")
	}
	untouched := geometry(t, two, "a")

	report, err := LayoutDocument(doc, Options{View: "one"})
	if err != nil {
		t.Fatal(err)
	}
	if report.Pages != 1 {
		t.Errorf("report = %+v, want exactly the selected page", report)
	}
	if hasPoints(one.FindAllConnectors()[0]) {
		t.Error("waypoints of the laid out page must be dropped")
	}
	if !hasPoints(two.FindAllConnectors()[0]) || geometry(t, two, "a") != untouched {
		t.Error("the page that was not selected must stay as it was")
	}
}

func hasPoints(connector *etree.Element) bool {
	g := connector.SelectElement("mxGeometry")
	return g != nil && g.SelectElement("Array") != nil
}

// #691: an unknown algorithm is rejected, like an unknown rank direction, and
// the error names the valid value.
func TestLayoutDocument_UnknownAlgorithm(t *testing.T) {
	doc := drawio.NewDocument()
	doc.AddPage("v", "v")
	_, err := LayoutDocument(doc, Options{Algorithm: "foo"})
	if err == nil || !strings.Contains(err.Error(), `"foo"`) || !strings.Contains(err.Error(), "hierarchical") {
		t.Errorf("want an error naming foo and hierarchical, got %v", err)
	}
	for _, ok := range []string{"", "hierarchical"} {
		if _, err := LayoutDocument(doc, Options{Algorithm: ok}); err != nil {
			t.Errorf("algorithm %q must be accepted: %v", ok, err)
		}
	}
}

func TestLayoutDocument_Errors(t *testing.T) {
	doc := drawio.NewDocument()
	doc.AddPage("v", "v")
	if _, err := LayoutDocument(doc, Options{View: "missing"}); err == nil {
		t.Error("expected an error for an unknown view")
	}
	if _, err := LayoutDocument(doc, Options{RankDir: "diagonal"}); err == nil {
		t.Error("expected an error for an unknown rank direction")
	}
}

// #717: the layout drops the waypoints (they belong to the old positions); the
// opposite connectors of a pair get theirs again for the new positions.
func TestLayoutDocument_OppositeConnectorsGetSeparateRoutes(t *testing.T) {
	doc := drawio.NewDocument()
	page := doc.AddPage("view-v", "V")
	addElem(t, page, "u", "", 100, 60)
	addElem(t, page, "s", "", 100, 60)
	page.CreateConnector(drawio.ConnectorData{From: "u", To: "s", SourceRef: "u", TargetRef: "s", Index: 0}, "")
	page.CreateConnector(drawio.ConnectorData{From: "s", To: "u", SourceRef: "s", TargetRef: "u", Index: 1}, "")

	if _, err := LayoutDocument(doc, Options{}); err != nil {
		t.Fatal(err)
	}

	pts := map[string]string{}
	for _, c := range page.FindAllConnectors() {
		if pt := c.FindElement("mxGeometry/Array[@as='points']/mxPoint"); pt != nil {
			pts[c.SelectAttrValue("id", "")] = pt.SelectAttrValue("x", "") + "," + pt.SelectAttrValue("y", "")
		}
	}
	if len(pts) != 2 {
		t.Fatalf("both connectors need a waypoint after the layout, got %v", pts)
	}
	var first string
	for _, p := range pts {
		if first == "" {
			first = p
		} else if p == first {
			t.Errorf("both connectors have the waypoint %s", p)
		}
	}
}

// #718: a title cell from an older file has no wrapping; the layout repairs it
// and grows the shape before it places the elements.
func TestLayoutDocument_GrowsShapesForLongTitles(t *testing.T) {
	doc := drawio.NewDocument()
	page := doc.AddPage("view-v", "V")
	sub := &drawio.SubCellTemplates{
		Title: &drawio.SubCellStyle{Style: "text;html=1;fontSize=13;fontStyle=1;align=center;", X: 0, Y: 0, Width: 160, Height: 24},
	}
	if err := page.CreateElement(drawio.ElementData{ID: "u", CellID: "u", Kind: "actor", Title: "U",
		X: 1, Y: 1, Width: 160, Height: 80, SubCells: sub}, "rounded=1"); err != nil {
		t.Fatal(err)
	}
	for _, c := range page.Root().SelectElements("mxCell") {
		if c.SelectAttrValue("id", "") == "u-title" {
			c.CreateAttr("value", "Customer Relationship Management Back-Office Operator")
			c.CreateAttr("style", "text;html=1;fontSize=13;fontStyle=1;align=center;")
		}
	}

	if _, err := LayoutDocument(doc, Options{}); err != nil {
		t.Fatal(err)
	}

	if h := geometry(t, page, "u").h; h <= 80 {
		t.Errorf("shape height %v: it must grow for the three lines of the title", h)
	}
	for _, c := range page.Root().SelectElements("mxCell") {
		if c.SelectAttrValue("id", "") == "u-title" && !strings.Contains(c.SelectAttrValue("style", ""), "whiteSpace=wrap") {
			t.Errorf("the title cell does not wrap: %s", c.SelectAttrValue("style", ""))
		}
	}
}
