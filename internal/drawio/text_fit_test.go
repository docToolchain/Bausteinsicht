package drawio

import (
	"strconv"
	"strings"
	"testing"
)

const longTitle = "Customer Relationship Management Back-Office Operator"

func cellGeom(t *testing.T, p *Page, id string) (y, h float64) {
	t.Helper()
	for _, c := range p.Root().SelectElements("mxCell") {
		if c.SelectAttrValue("id", "") == id {
			g := c.SelectElement("mxGeometry")
			y, _ = strconv.ParseFloat(g.SelectAttrValue("y", "0"), 64)
			h, _ = strconv.ParseFloat(g.SelectAttrValue("height", "0"), 64)
			return y, h
		}
	}
	t.Fatalf("no cell %q", id)
	return 0, 0
}

func objectHeight(t *testing.T, p *Page, id string) float64 {
	t.Helper()
	g := p.FindElement(id).FindElement("mxCell/mxGeometry")
	h, _ := strconv.ParseFloat(g.SelectAttrValue("height", "0"), 64)
	return h
}

func fitPage(t *testing.T, title string) *Page {
	t.Helper()
	p := NewDocument().AddPage("view-v", "V")
	if err := p.CreateElement(ElementData{
		ID: "u", CellID: "u", Kind: "actor", Title: title, Technology: "Go", Description: "d",
		SubCells: testSubCellTemplates(), Width: 160, Height: 140,
	}, "rounded=1"); err != nil {
		t.Fatal(err)
	}
	return p
}

func cellStyle(p *Page, id string) string {
	for _, c := range p.Root().SelectElements("mxCell") {
		if c.SelectAttrValue("id", "") == id {
			return c.SelectAttrValue("style", "")
		}
	}
	return ""
}

// #718: the title cell had a fixed size and no wrapping, so a long title was
// clipped ("Customer Relatio"). The text wraps, and the title cell and the
// shape grow with the number of lines.
func TestCreateElement_LongTitleWrapsAndGrowsTheShape(t *testing.T) {
	p := fitPage(t, longTitle)

	if !strings.Contains(cellStyle(p, "u-title"), "whiteSpace=wrap") {
		t.Errorf("the title cell must wrap: %s", cellStyle(p, "u-title"))
	}
	_, titleH := cellGeom(t, p, "u-title")
	if titleH < 3*14 {
		t.Errorf("title height %v cannot hold the wrapped text (at least 3 lines of 14 px)", titleH)
	}
	delta := titleH - 30 // the template's title height
	if got := objectHeight(t, p, "u"); got != 140+delta {
		t.Errorf("shape height %v, want %v (140 + growth of the title)", got, 140+delta)
	}
	if y, _ := cellGeom(t, p, "u-tech"); y != 55+delta {
		t.Errorf("technology cell at y=%v, want %v (shifted by the growth)", y, 55+delta)
	}
	if y, _ := cellGeom(t, p, "u-desc"); y != 80+delta {
		t.Errorf("description cell at y=%v, want %v", y, 80+delta)
	}
}

func TestCreateElement_ShortTitleKeepsTheTemplateSize(t *testing.T) {
	p := fitPage(t, "API")
	if _, h := cellGeom(t, p, "u-title"); h != 30 {
		t.Errorf("title height %v, want the template's 30", h)
	}
	if got := objectHeight(t, p, "u"); got != 140 {
		t.Errorf("shape height %v, want 140", got)
	}
	if y, _ := cellGeom(t, p, "u-tech"); y != 55 {
		t.Errorf("technology cell moved to %v", y)
	}
}

// The title of the issue's second case mixes quotes, markup, CJK and an emoji.
func TestCreateElement_WideCharactersNeedMoreRoomThanLatinOnes(t *testing.T) {
	latin := fitPage(t, "abcdefghijklmno") // 15 characters, one line
	cjk := fitPage(t, "日本語日本語日本語日本語日本")    // 14 characters, twice as wide each: two lines
	_, latinH := cellGeom(t, latin, "u-title")
	_, cjkH := cellGeom(t, cjk, "u-title")
	if cjkH <= latinH {
		t.Errorf("14 CJK characters (title height %v) need more room than 15 Latin ones (%v)", cjkH, latinH)
	}
}

func TestUpdateElement_LongTitleGrowsTheShapeOnceAndIsIdempotent(t *testing.T) {
	p := fitPage(t, "API")
	p.UpdateElement("u", ElementData{Title: longTitle, Technology: "Go", Description: "d", SubCells: testSubCellTemplates()})
	_, h1 := cellGeom(t, p, "u-title")
	shape1 := objectHeight(t, p, "u")
	if h1 < 3*14 || shape1 <= 140 {
		t.Fatalf("after the update: title %v, shape %v", h1, shape1)
	}

	p.UpdateElement("u", ElementData{Title: longTitle, Technology: "Go", Description: "d", SubCells: testSubCellTemplates()})
	if _, h2 := cellGeom(t, p, "u-title"); h2 != h1 {
		t.Errorf("the title grew again: %v -> %v", h1, h2)
	}
	if shape2 := objectHeight(t, p, "u"); shape2 != shape1 {
		t.Errorf("the shape grew again: %v -> %v", shape1, shape2)
	}
}

// Older files have sub-cells without wrapping; FitElementText repairs them.
func TestFitElementText_RepairsAnOlderCell(t *testing.T) {
	p := fitPage(t, "API")
	for _, c := range p.Root().SelectElements("mxCell") {
		if c.SelectAttrValue("id", "") == "u-title" {
			c.CreateAttr("value", longTitle)
			c.CreateAttr("style", strings.ReplaceAll(c.SelectAttrValue("style", ""), "whiteSpace=wrap;", ""))
		}
	}
	p.FitElementText(p.FindElement("u"))
	if !strings.Contains(cellStyle(p, "u-title"), "whiteSpace=wrap") {
		t.Error("wrapping not restored")
	}
	if _, h := cellGeom(t, p, "u-title"); h < 3*14 {
		t.Errorf("title height %v", h)
	}
}
