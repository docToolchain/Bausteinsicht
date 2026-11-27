package drawio

import (
	"strings"
	"testing"

	"github.com/beevik/etree"
)

const renameDoc = `<mxfile><diagram id="view-ctx" name="Ctx"><mxGraphModel><root>
<mxCell id="0"/><mxCell id="1" parent="0"/>
<object id="ctx--shop" bausteinsicht_id="shop"><mxCell style="s" vertex="1" parent="1"><mxGeometry x="777" y="5" as="geometry"/></mxCell></object>
<mxCell id="ctx--shop-title" value="Shop" vertex="1" parent="ctx--shop"/>
<object id="ctx--shop.api" bausteinsicht_id="shop.api"><mxCell vertex="1" parent="ctx--shop"><mxGeometry x="5" as="geometry"/></mxCell></object>
<mxCell id="ctx--shop.api-desc" value="API" vertex="1" parent="ctx--shop.api"/>
<mxCell id="doc-ctx--shop.api-doc-0" vertex="1" parent="ctx--shop.api"/>
<object id="ctx--shopper" bausteinsicht_id="shopper"><mxCell vertex="1" parent="1"/></object>
<mxCell id="ctx--shopper-title" parent="ctx--shopper"/>
<mxCell id="rel-ctx--shop-ctx--shopper-0" edge="1" source="ctx--shop" target="ctx--shopper" parent="1"/>
<mxCell id="rel-ctx--shop.api-ctx--shopper-3" edge="1" source="ctx--shop.api" target="ctx--shopper" parent="1"/>
<mxCell id="user-arrow" edge="1" source="ctx--shop" target="ctx--shopper" parent="1"/>
</root></mxGraphModel></diagram></mxfile>`

func ids(t *testing.T, doc *Document) string {
	t.Helper()
	root := doc.Pages()[0].Root()
	var parts []string
	for _, el := range root.ChildElements() {
		parts = append(parts, el.Tag+":"+el.SelectAttrValue("id", "")+"<"+el.SelectAttrValue("parent", "")+
			"|"+el.SelectAttrValue("bausteinsicht_id", "")+"|"+el.SelectAttrValue("source", "")+">"+el.SelectAttrValue("target", ""))
	}
	return strings.Join(parts, "\n")
}

// Renaming shop to store renames the cells of shop and its child, their
// sub-cells and icons and the connectors at them, keeps the geometry and leaves
// shopper, whose ID merely starts with "shop", alone (#686).
func TestDocument_RenameElement(t *testing.T) {
	d := etree.NewDocument()
	if err := d.ReadFromString(renameDoc); err != nil {
		t.Fatal(err)
	}
	doc := &Document{tree: d}

	if n := doc.RenameElement("shop", "store"); n != 2 {
		t.Errorf("renamed %d element cells, want 2", n)
	}
	got := ids(t, doc)
	for _, want := range []string{
		"object:ctx--store<|store|>",
		"mxCell:ctx--store-title<ctx--store||>",
		"object:ctx--store.api<|store.api|>",
		"mxCell:ctx--store.api-desc<ctx--store.api||>",
		"mxCell:doc-ctx--store.api-doc-0<ctx--store.api||>",
		"object:ctx--shopper<|shopper|>",
		"mxCell:ctx--shopper-title<ctx--shopper||>",
		"mxCell:rel-ctx--store-ctx--shopper-0<1||ctx--store>ctx--shopper",
		"mxCell:rel-ctx--store.api-ctx--shopper-3<1||ctx--store.api>ctx--shopper",
		"mxCell:user-arrow<1||ctx--store>ctx--shopper",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "ctx--shop\"") || strings.Contains(got, "bausteinsicht_id=\"shop\"") {
		t.Errorf("old IDs left:\n%s", got)
	}
	if x := doc.Pages()[0].FindElement("store").SelectElement("mxCell").SelectElement("mxGeometry").SelectAttrValue("x", ""); x != "777" {
		t.Errorf("the geometry changed: x=%s", x)
	}
	// The parent link of the wrapped cells moved with the cells above.
	if p := doc.Pages()[0].FindElement("store.api").SelectElement("mxCell").SelectAttrValue("parent", ""); p != "ctx--store" {
		t.Errorf("parent of the wrapped child cell = %q", p)
	}
	if n := doc.RenameElement("missing", "x"); n != 0 {
		t.Errorf("renamed %d cells of an element that is not there", n)
	}
}
