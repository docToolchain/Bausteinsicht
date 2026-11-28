package drawio

import (
	"math"
	"strconv"
	"testing"
)

func waypointOf(t *testing.T, p *Page, from, to string, index int) (x, y float64, ok bool) {
	t.Helper()
	conn := p.FindConnector(from, to, index)
	if conn == nil {
		t.Fatalf("no connector %s -> %s", from, to)
	}
	pt := conn.FindElement("mxGeometry/Array[@as='points']/mxPoint")
	if pt == nil {
		return 0, 0, false
	}
	x, _ = strconv.ParseFloat(pt.SelectAttrValue("x", ""), 64)
	y, _ = strconv.ParseFloat(pt.SelectAttrValue("y", ""), 64)
	return x, y, true
}

func oppositePage(t *testing.T) *Page {
	t.Helper()
	p := NewDocument().AddPage("view-v", "V")
	for _, e := range []ElementData{
		{ID: "u", CellID: "u", Kind: "actor", Title: "U", X: 0, Y: 100, Width: 100, Height: 60},
		{ID: "s", CellID: "s", Kind: "system", Title: "S", X: 300, Y: 100, Width: 100, Height: 60},
	} {
		if err := p.CreateElement(e, "rounded=1"); err != nil {
			t.Fatal(err)
		}
	}
	p.CreateConnector(ConnectorData{From: "u", To: "s", Label: "submits requests", Index: 0}, "edgeStyle=orthogonalEdgeStyle;")
	p.CreateConnector(ConnectorData{From: "s", To: "u", Label: "sends notifications", Index: 1}, "edgeStyle=orthogonalEdgeStyle;")
	return p
}

// #717: A->B and B->A had the same route and the same label position, so one
// label was hidden. Each gets a waypoint on its own side of the line between
// the element centres.
func TestSeparateOppositeConnectors_RoutesThemOnOppositeSides(t *testing.T) {
	p := oppositePage(t)

	p.SeparateOppositeConnectors()

	x1, y1, ok1 := waypointOf(t, p, "u", "s", 0)
	x2, y2, ok2 := waypointOf(t, p, "s", "u", 1)
	if !ok1 || !ok2 {
		t.Fatalf("both connectors need a waypoint, got %v %v", ok1, ok2)
	}
	// The centres are (50,130) and (350,130): the line is horizontal at y=130.
	if math.Abs(x1-200) > 1 || math.Abs(x2-200) > 1 {
		t.Errorf("the waypoints sit in the middle between the elements: x=%v and %v", x1, x2)
	}
	if (y1-130)*(y2-130) >= 0 {
		t.Errorf("the waypoints must lie on opposite sides of the line: y=%v and %v", y1, y2)
	}
	if math.Abs(y1-y2) < 40 {
		t.Errorf("the routes are too close for two readable labels: y=%v and %v", y1, y2)
	}
	// Each label sits towards its own source, so the two labels are at the two
	// ends of the line even when they are wider than the distance of the routes.
	for _, c := range [][3]string{{"u", "s", "0"}, {"s", "u", "1"}} {
		idx, _ := strconv.Atoi(c[2])
		if x := p.FindConnector(c[0], c[1], idx).FindElement("mxGeometry").SelectAttrValue("x", ""); x != "-0.4" {
			t.Errorf("connector %s->%s: label position x=%q, want -0.4", c[0], c[1], x)
		}
	}
}

func TestSeparateOppositeConnectors_IsIdempotentAndKeepsUserWaypoints(t *testing.T) {
	p := oppositePage(t)
	// The user routed the second connector by hand.
	geom := p.FindConnector("s", "u", 1).SelectElement("mxGeometry")
	arr := geom.CreateElement("Array")
	arr.CreateAttr("as", "points")
	pt := arr.CreateElement("mxPoint")
	pt.CreateAttr("x", "111")
	pt.CreateAttr("y", "222")

	p.SeparateOppositeConnectors()
	x1, y1, _ := waypointOf(t, p, "u", "s", 0)
	p.SeparateOppositeConnectors()

	if x, y, _ := waypointOf(t, p, "s", "u", 1); x != 111 || y != 222 {
		t.Errorf("the waypoint of the user was changed to %v,%v", x, y)
	}
	if x, y, _ := waypointOf(t, p, "u", "s", 0); x != x1 || y != y1 {
		t.Errorf("a second run moved the waypoint: %v,%v -> %v,%v", x1, y1, x, y)
	}
	if n := len(p.FindConnector("u", "s", 0).FindElements("mxGeometry/Array")); n != 1 {
		t.Errorf("%d point lists after two runs, want 1", n)
	}
}

func TestSeparateOppositeConnectors_LeavesOneWayConnectorsAlone(t *testing.T) {
	p := oppositePage(t)
	p.DeleteConnector("s", "u", 1)
	p.SeparateOppositeConnectors()
	if _, _, ok := waypointOf(t, p, "u", "s", 0); ok {
		t.Error("a connector without a counterpart must keep the automatic route")
	}
}

func TestSeparateOppositeConnectors_NestedElementsUseAbsolutePositions(t *testing.T) {
	p := NewDocument().AddPage("view-v", "V")
	for _, e := range []ElementData{
		{ID: "box", CellID: "box", Kind: "system", Title: "Box", X: 1000, Y: 500, Width: 600, Height: 300},
		{ID: "a", CellID: "a", ParentID: "box", Kind: "c", Title: "A", X: 20, Y: 60, Width: 100, Height: 60},
		{ID: "b", CellID: "b", ParentID: "box", Kind: "c", Title: "B", X: 320, Y: 60, Width: 100, Height: 60},
	} {
		if err := p.CreateElement(e, "rounded=1"); err != nil {
			t.Fatal(err)
		}
	}
	p.CreateConnector(ConnectorData{From: "a", To: "b", Index: 0}, "")
	p.CreateConnector(ConnectorData{From: "b", To: "a", Index: 1}, "")

	p.SeparateOppositeConnectors()

	x, y, ok := waypointOf(t, p, "a", "b", 0)
	// centres (1070,590) and (1370,590) in page coordinates.
	if !ok || math.Abs(x-1220) > 1 || math.Abs(y-590) < 20 {
		t.Errorf("waypoint %v,%v ok=%v: expected x=1220 and an offset from y=590", x, y, ok)
	}
}
