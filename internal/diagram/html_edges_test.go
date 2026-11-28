package diagram

import (
	"encoding/json"
	"math"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

func htmlData(t *testing.T, m *model.BausteinsichtModel, view string) (HTMLDiagramData, string) {
	t.Helper()
	out, err := RenderHTML(m, view)
	if err != nil {
		t.Fatal(err)
	}
	start := strings.Index(out, "const DIAGRAM_DATA = ")
	if start < 0 {
		t.Fatal("no DIAGRAM_DATA in the HTML")
	}
	start += len("const DIAGRAM_DATA = ")
	end := strings.Index(out[start:], ";\n")
	var data HTMLDiagramData
	if err := json.Unmarshal([]byte(out[start:start+end]), &data); err != nil {
		t.Fatalf("embedded JSON: %v", err)
	}
	return data, out
}

func onBorder(n HTMLNode, x, y float64) bool {
	const eps = 1e-6
	inX := x >= n.X-eps && x <= n.X+htmlNodeWidth+eps
	inY := y >= n.Y-eps && y <= n.Y+htmlNodeHeight+eps
	onV := math.Abs(x-n.X) < eps || math.Abs(x-(n.X+htmlNodeWidth)) < eps
	onH := math.Abs(y-n.Y) < eps || math.Abs(y-(n.Y+htmlNodeHeight)) < eps
	return inX && inY && (onV || onH)
}

// A line that ends in the middle of its node is hidden by the node, and so is its
// arrowhead: every edge runs from the border of the source to the border of the
// target, so that the arrowhead can be seen (#732).
func TestHTML_EdgesRunBorderToBorder(t *testing.T) {
	data, _ := htmlData(t, testModel(), "context")
	nodes := map[string]HTMLNode{}
	for _, n := range data.Nodes {
		nodes[n.ID] = n
	}
	if len(data.Edges) == 0 {
		t.Fatal("setup: no edges")
	}
	for _, e := range data.Edges {
		from, to := nodes[e.From], nodes[e.To]
		if !onBorder(from, e.X1, e.Y1) {
			t.Errorf("%s -> %s: the line starts at (%v, %v), not on the border of %s (%v, %v)", e.From, e.To, e.X1, e.Y1, e.From, from.X, from.Y)
		}
		if !onBorder(to, e.X2, e.Y2) {
			t.Errorf("%s -> %s: the line ends at (%v, %v), not on the border of %s (%v, %v)", e.From, e.To, e.X2, e.Y2, e.To, to.X, to.Y)
		}
	}
}

func TestBorderPoint(t *testing.T) {
	at := func(x, y float64) HTMLNode { return HTMLNode{X: x, Y: y} }
	for name, tc := range map[string]struct {
		a, b         HTMLNode
		wantX, wantY float64
	}{
		"to the right": {at(0, 0), at(300, 0), 160, 40},
		"to the left":  {at(300, 0), at(0, 0), 300, 40},
		"below":        {at(0, 0), at(0, 300), 80, 80},
		"above":        {at(0, 300), at(0, 0), 80, 300},
		// the centres are 400 apart in x and 100 in y: the line leaves through the
		// right side, 80 right of the centre and 20 below it
		"diagonal, flat": {at(0, 0), at(400, 100), 160, 60},
		// 100 apart in x and 400 in y: it leaves through the bottom, 40 below the
		// centre and 10 to the right of it
		"diagonal, steep": {at(0, 0), at(100, 400), 90, 80},
	} {
		x, y := borderPoint(tc.a, tc.b)
		if math.Abs(x-tc.wantX) > 1e-6 || math.Abs(y-tc.wantY) > 1e-6 {
			t.Errorf("%s: want (%v, %v), got (%v, %v)", name, tc.wantX, tc.wantY, x, y)
		}
	}
}

// The heading of the page is the title of the view, not a fixed text (#732).
func TestHTML_HeadingIsTheViewTitle(t *testing.T) {
	m := testModel()
	m.Views["context"] = model.View{Title: "Context <of> the shop", Include: m.Views["context"].Include}
	_, out := htmlData(t, m, "context")
	if !strings.Contains(out, "<h1>Context &lt;of&gt; the shop</h1>") {
		t.Errorf("want the escaped view title as the heading, got:\n%s", out[:min(len(out), 3000)])
	}
	if strings.Contains(out, "<h1>Architecture Diagram</h1>") {
		t.Error("the fixed heading is still there")
	}
}

// Labels sit on other lines: a white outline keeps them readable.
func TestHTML_LabelsHaveAnOutline(t *testing.T) {
	_, out := htmlData(t, testModel(), "context")
	if !strings.Contains(out, "paint-order: stroke") {
		t.Error("edge labels need an outline (paint-order: stroke)")
	}
}
