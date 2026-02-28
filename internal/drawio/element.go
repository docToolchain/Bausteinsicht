package drawio

import (
	"fmt"

	"github.com/beevik/etree"
)

// ElementData holds the data needed to create or update an element.
type ElementData struct {
	ID          string  // bausteinsicht_id (e.g., "webshop.api")
	Kind        string  // bausteinsicht_kind (e.g., "container")
	Title       string  // display title
	Technology  string  // technology string
	Description string  // tooltip text
	Link        string  // drill-down link (e.g., "data:page/id,view-containers")
	ParentID    string  // parent cell ID ("1" for top-level, container ID for children)
	X, Y        float64 // position
	Width       float64 // element width
	Height      float64 // element height
}

// CreateElement creates an <object> wrapping an <mxCell vertex="1"> with <mxGeometry>.
// ParentID defaults to "1" if empty.
func (p *Page) CreateElement(data ElementData, style string) error {
	root := p.Root()
	if root == nil {
		return fmt.Errorf("CreateElement: page has no root element")
	}

	parentID := data.ParentID
	if parentID == "" {
		parentID = "1"
	}

	obj := root.CreateElement("object")
	obj.CreateAttr("label", GenerateLabel(data.Title, data.Technology))
	obj.CreateAttr("id", data.ID)
	obj.CreateAttr("bausteinsicht_id", data.ID)
	obj.CreateAttr("bausteinsicht_kind", data.Kind)
	if data.Technology != "" {
		obj.CreateAttr("technology", data.Technology)
	}
	if data.Description != "" {
		obj.CreateAttr("tooltip", data.Description)
	}
	if data.Link != "" {
		obj.CreateAttr("link", data.Link)
	}

	cell := obj.CreateElement("mxCell")
	cell.CreateAttr("style", style)
	cell.CreateAttr("vertex", "1")
	cell.CreateAttr("parent", parentID)

	geom := cell.CreateElement("mxGeometry")
	geom.CreateAttr("x", formatFloat(data.X))
	geom.CreateAttr("y", formatFloat(data.Y))
	geom.CreateAttr("width", formatFloat(data.Width))
	geom.CreateAttr("height", formatFloat(data.Height))
	geom.CreateAttr("as", "geometry")

	return nil
}

// FindElement returns the <object> element with the given bausteinsicht_id, or nil if not found.
func (p *Page) FindElement(bausteinsichtID string) *etree.Element {
	root := p.Root()
	if root == nil {
		return nil
	}
	for _, obj := range root.SelectElements("object") {
		if obj.SelectAttrValue("bausteinsicht_id", "") == bausteinsichtID {
			return obj
		}
	}
	return nil
}

// FindAllElements returns all <object> elements that have a bausteinsicht_id attribute.
func (p *Page) FindAllElements() []*etree.Element {
	root := p.Root()
	if root == nil {
		return nil
	}
	var result []*etree.Element
	for _, obj := range root.SelectElements("object") {
		if obj.SelectAttrValue("bausteinsicht_id", "") != "" {
			result = append(result, obj)
		}
	}
	return result
}

// UpdateElement updates label, tooltip, technology, and link on an existing element.
func (p *Page) UpdateElement(id string, data ElementData) {
	obj := p.FindElement(id)
	if obj == nil {
		return
	}

	setAttr(obj, "label", GenerateLabel(data.Title, data.Technology))
	setAttr(obj, "tooltip", data.Description)
	setAttr(obj, "link", data.Link)
	if data.Technology != "" {
		setAttr(obj, "technology", data.Technology)
	}
}

// DeleteElement removes the <object> element with the given bausteinsicht_id.
// It also removes any mxCell connectors that reference this element as source or target.
func (p *Page) DeleteElement(id string) {
	root := p.Root()
	if root == nil {
		return
	}

	obj := p.FindElement(id)
	if obj != nil {
		root.RemoveChild(obj)
	}

	for _, cell := range root.SelectElements("mxCell") {
		src := cell.SelectAttrValue("source", "")
		dst := cell.SelectAttrValue("target", "")
		if src == id || dst == id {
			root.RemoveChild(cell)
		}
	}
}

// setAttr sets an attribute on an element, creating it if it doesn't exist.
// If value is empty, any existing attribute is removed.
func setAttr(el *etree.Element, key, value string) {
	if value == "" {
		el.RemoveAttr(key)
		return
	}
	attr := el.SelectAttr(key)
	if attr != nil {
		attr.Value = value
	} else {
		el.CreateAttr(key, value)
	}
}

// formatFloat formats a float64 as a string without trailing zeros where possible.
func formatFloat(f float64) string {
	if f == float64(int(f)) {
		return fmt.Sprintf("%d", int(f))
	}
	return fmt.Sprintf("%g", f)
}
