package drawio

import (
	"github.com/beevik/etree"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

func textCellPage(t *testing.T) *Page {
	t.Helper()
	doc := NewDocument()
	return doc.AddPage("view-v", "V")
}

// fataler is what both *testing.T and *rapid.T offer.
type fataler interface {
	Helper()
	Fatalf(format string, args ...any)
}

func cellValue(t fataler, p *Page, id string) string {
	t.Helper()
	for _, c := range p.Root().SelectElements("mxCell") {
		if c.SelectAttrValue("id", "") == id {
			return c.SelectAttrValue("value", "")
		}
	}
	t.Fatalf("no cell %q", id)
	return ""
}

// The text cells have html=1, so the value is HTML: "<" and "&" of the model
// text must be escaped, otherwise draw.io reads "<Team>" as a tag (#699).
func TestSubCells_WriteHTMLEscapedValues(t *testing.T) {
	p := textCellPage(t)
	if err := p.CreateElement(ElementData{
		ID: "u", CellID: "u", Kind: "actor", Title: "R&D <Team>", Technology: "a<b", Description: "a < b & c",
		SubCells: testSubCellTemplates(), Width: 160, Height: 80,
	}, "rounded=1;"); err != nil {
		t.Fatal(err)
	}
	for id, want := range map[string]string{
		"u-title": "R&amp;D &lt;Team&gt;",
		"u-tech":  "[a&lt;b]",
		"u-desc":  "a &lt; b &amp; c",
	} {
		if got := cellValue(t, p, id); got != want {
			t.Errorf("%s: value %q, want %q", id, got, want)
		}
	}
	title, tech, desc := p.ReadElementFields(p.FindElement("u"))
	if title != "R&D <Team>" || tech != "a<b" || desc != "a < b & c" {
		t.Errorf("read back %q %q %q", title, tech, desc)
	}

	p.UpdateElement("u", ElementData{Title: "x & <y>", Technology: "t&", Description: "d>", SubCells: testSubCellTemplates()})
	if got := cellValue(t, p, "u-title"); got != "x &amp; &lt;y&gt;" {
		t.Errorf("updated title value %q", got)
	}
	title, tech, desc = p.ReadElementFields(p.FindElement("u"))
	if title != "x & <y>" || tech != "t&" || desc != "d>" {
		t.Errorf("read back after the update %q %q %q", title, tech, desc)
	}
}

func TestConnectorLabel_IsHTMLEscaped(t *testing.T) {
	p := textCellPage(t)
	p.CreateConnector(ConnectorData{From: "a", To: "b", Label: "x < y & z"}, "html=1;")
	conn := p.FindConnector("a", "b", 0)
	if got := conn.SelectAttrValue("value", ""); got != "x &lt; y &amp; z" {
		t.Errorf("connector value %q", got)
	}
	p.UpdateConnectorLabel("a", "b", 0, "<b>")
	if got := p.FindConnector("a", "b", 0).SelectAttrValue("value", ""); got != "&lt;b&gt;" {
		t.Errorf("updated connector value %q", got)
	}
	if got := UnescapeCellText(p.FindConnector("a", "b", 0).SelectAttrValue("value", "")); got != "<b>" {
		t.Errorf("unescaped %q", got)
	}
}

// Whatever the text, it comes back unchanged from the text cells.
func TestSubCells_TextRoundtripProperty(t *testing.T) {
	text := rapid.StringMatching(`[A-Za-z0-9 <>&"'_;#.-]{1,40}`)
	rapid.Check(t, func(t *rapid.T) {
		title, tech, desc := text.Draw(t, "title"), text.Draw(t, "tech"), text.Draw(t, "desc")
		doc := NewDocument()
		p := doc.AddPage("view-v", "V")
		if err := p.CreateElement(ElementData{ID: "e", CellID: "e", Kind: "k", Title: title, Technology: tech, Description: desc,
			SubCells: testSubCellTemplates(), Width: 160, Height: 80}, "s;"); err != nil {
			t.Fatal(err)
		}
		gt, gtech, gd := p.ReadElementFields(p.FindElement("e"))
		if gt != title || gtech != tech || gd != desc {
			t.Fatalf("read back %q %q %q, want %q %q %q", gt, gtech, gd, title, tech, desc)
		}
		for _, id := range []string{"e-title", "e-tech", "e-desc"} {
			if v := cellValue(t, p, id); strings.ContainsAny(v, "<>") {
				t.Fatalf("%s holds raw markup characters: %q", id, v)
			}
		}
	})
}

// Cells without html=1 show their value literally, so it must not be decoded
// when read (#699 review).
func TestReadElementFields_NoDecodingInSubCellsWithoutHTML(t *testing.T) {
	p := textCellPage(t)
	if err := p.CreateElement(ElementData{ID: "u", CellID: "u", Kind: "actor", Title: "x", Technology: "y", Description: "z",
		SubCells: testSubCellTemplates(), Width: 160, Height: 80}, "rounded=1;"); err != nil {
		t.Fatal(err)
	}
	for _, c := range p.Root().SelectElements("mxCell") {
		id := c.SelectAttrValue("id", "")
		if strings.HasPrefix(id, "u-") {
			c.CreateAttr("style", strings.ReplaceAll(c.SelectAttrValue("style", ""), "html=1", "html=0"))
			c.CreateAttr("value", "&lt;"+id+"&gt;")
		}
	}
	title, tech, desc := p.ReadElementFields(p.FindElement("u"))
	if title != "&lt;u-title&gt;" || tech != "&lt;u-tech&gt;" || desc != "&lt;u-desc&gt;" {
		t.Errorf("read %q %q %q, want the literal values", title, tech, desc)
	}
}

func TestCellText_DecodesOnlyHTMLCells(t *testing.T) {
	doc := etree.NewDocument()
	html := doc.CreateElement("mxCell")
	html.CreateAttr("style", "text;html=1;")
	html.CreateAttr("value", "a &lt; b")
	plain := doc.CreateElement("mxCell")
	plain.CreateAttr("style", "text;html=0;")
	plain.CreateAttr("value", "a &lt; b")
	if got := CellText(html); got != "a < b" {
		t.Errorf("html cell: %q", got)
	}
	if got := CellText(plain); got != "a &lt; b" {
		t.Errorf("plain cell: %q", got)
	}
}

// A connector whose style is replaced (kind change) must keep html=1, because
// its label is stored escaped (#699 review).
func TestSetConnectorStyle_KeepsTheHTMLFlag(t *testing.T) {
	p := textCellPage(t)
	p.CreateConnector(ConnectorData{From: "a", To: "b", Label: "x & y"}, "html=1;")
	p.SetConnectorStyle("a", "b", 0, "endArrow=block;dashed=1;")
	if style := p.FindConnector("a", "b", 0).SelectAttrValue("style", ""); !strings.Contains(style, "html=1") {
		t.Errorf("style %q lacks html=1", style)
	}
}
