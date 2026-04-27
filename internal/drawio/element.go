package drawio

import (
	"fmt"
	"strings"

	"github.com/beevik/etree"
)

// ElementData holds the data needed to create or update an element.
type ElementData struct {
	ID          string            // bausteinsicht_id (e.g., "webshop.api")
	CellID      string            // draw.io cell ID (file-wide unique); defaults to ID if empty
	Kind        string            // bausteinsicht_kind (e.g., "container")
	Title       string            // display title
	Technology  string            // technology string
	Description string            // tooltip text
	Link        string            // drill-down link (e.g., "data:page/id,view-containers")
	ParentID    string            // parent cell ID ("1" for top-level, container ID for children)
	X, Y        float64           // position
	Width       float64           // element width
	Height      float64           // element height
	SubCells    *SubCellTemplates // sub-cell templates; nil for legacy HTML labels
}

// CreateElement creates an <object> wrapping an <mxCell vertex="1"> with <mxGeometry>.
// ParentID defaults to "1" if empty.
// When SubCells is non-nil, the element uses grouped sub-cells for title/tech/desc
// instead of an HTML label.
func (p *Page) CreateElement(data ElementData, style string) error {
	root := p.Root()
	if root == nil {
		return fmt.Errorf("CreateElement: page has no root element")
	}

	parentID := data.ParentID
	if parentID == "" {
		parentID = "1"
	}

	cellID := data.CellID
	if cellID == "" {
		cellID = data.ID
	}

	obj := root.CreateElement("object")
	if data.SubCells != nil {
		obj.CreateAttr("label", "")
	} else {
		obj.CreateAttr("label", GenerateLabel(data.Title, data.Technology, data.Description))
	}
	obj.CreateAttr("id", cellID)
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

	// Ensure container=1 is set when using sub-cells (required for child grouping).
	if data.SubCells != nil {
		// Replace container=0 with container=1, or append if missing.
		if strings.Contains(style, "container=0") {
			style = strings.Replace(style, "container=0", "container=1", 1)
		} else if !strings.Contains(style, "container=1") {
			style = strings.TrimRight(style, ";") + ";container=1;"
		}
	}

	// HTML labels require html=1 in the cell style; without it draw.io renders
	// the raw markup as plain text. This guard covers elements whose kind has no
	// template entry and therefore receives an empty style fallback.
	if data.SubCells == nil && !strings.Contains(style, "html=1") {
		if style != "" && !strings.HasSuffix(style, ";") {
			style += ";"
		}
		style += "html=1;"
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

	// Create grouped sub-cells for title, technology, and description.
	if data.SubCells != nil {
		createSubCells(root, cellID, data, data.SubCells)
	}

	return nil
}

// SubCellTemplates holds the template styles for creating text sub-cells.
type SubCellTemplates struct {
	Title *SubCellStyle
	Tech  *SubCellStyle
	Desc  *SubCellStyle
}

// createSubCells creates child mxCell text elements inside the parent element.
func createSubCells(root *etree.Element, parentCellID string, data ElementData, sc *SubCellTemplates) {
	// Title sub-cell (always created).
	if sc.Title != nil {
		createTextSubCell(root, parentCellID+"-title", parentCellID, data.Title,
			sc.Title, data.Width, data.Height)
	}

	// Technology sub-cell (only when technology is non-empty).
	if sc.Tech != nil && data.Technology != "" {
		createTextSubCell(root, parentCellID+"-tech", parentCellID, "["+data.Technology+"]",
			sc.Tech, data.Width, data.Height)
	}

	// Description sub-cell (only when description is non-empty).
	if sc.Desc != nil && data.Description != "" {
		createTextSubCell(root, parentCellID+"-desc", parentCellID, data.Description,
			sc.Desc, data.Width, data.Height)
	}
}

// createTextSubCell creates a single text mxCell child element.
// Sub-cells are locked (non-movable, non-resizable, non-deletable, non-connectable)
// so that clicking the shape always selects the parent element.
func createTextSubCell(root *etree.Element, id, parentID, value string, sub *SubCellStyle, parentW, parentH float64) {
	cell := root.CreateElement("mxCell")
	cell.CreateAttr("id", id)
	cell.CreateAttr("value", value)
	// Make sub-cells transparent to mouse events so clicks pass through
	// to the parent element. This lets users grab the whole shape at once.
	style := setStyleFlags(sub.Style, "pointerEvents=0")
	cell.CreateAttr("style", style)
	cell.CreateAttr("vertex", "1")
	cell.CreateAttr("connectable", "0")
	cell.CreateAttr("parent", parentID)

	geom := cell.CreateElement("mxGeometry")
	// Scale sub-cell width to parent width, keep x/y/height from template.
	w := parentW
	if w == 0 {
		w = sub.Width
	}
	geom.CreateAttr("x", formatFloat(sub.X))
	geom.CreateAttr("y", formatFloat(sub.Y))
	geom.CreateAttr("width", formatFloat(w))
	geom.CreateAttr("height", formatFloat(sub.Height))
	geom.CreateAttr("as", "geometry")
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
// If the element uses sub-cells, child text cells are updated instead of the HTML label.
func (p *Page) UpdateElement(id string, data ElementData) {
	obj := p.FindElement(id)
	if obj == nil {
		return
	}

	cellID := obj.SelectAttrValue("id", "")

	// Check if element uses sub-cells (label is empty and has child text cells).
	root := p.Root()
	childCells := findChildTextCells(root, cellID)
	if len(childCells) > 0 {
		// Update existing sub-cells and manage tech/desc cells.
		updateSubCells(root, cellID, childCells, data)
		setAttr(obj, "label", "")
	} else {
		setAttr(obj, "label", GenerateLabel(data.Title, data.Technology, data.Description))
	}

	setAttr(obj, "tooltip", data.Description)
	setAttr(obj, "link", data.Link)
	if data.Technology != "" {
		setAttr(obj, "technology", data.Technology)
	} else {
		setAttr(obj, "technology", "")
	}
}

// findChildTextCells finds all text sub-cells that are children of the given parent cell ID.
// Returns a map of role ("title", "tech", "desc") to the mxCell element.
func findChildTextCells(root *etree.Element, parentCellID string) map[string]*etree.Element {
	if root == nil || parentCellID == "" {
		return nil
	}
	result := make(map[string]*etree.Element)
	for _, cell := range root.SelectElements("mxCell") {
		if cell.SelectAttrValue("parent", "") != parentCellID {
			continue
		}
		cellID := cell.SelectAttrValue("id", "")
		style := cell.SelectAttrValue("style", "")
		if !isTextSubCell(style) {
			continue
		}
		switch {
		case hasSuffix(cellID, "-title"):
			result["title"] = cell
		case hasSuffix(cellID, "-tech"):
			result["tech"] = cell
		case hasSuffix(cellID, "-desc"):
			result["desc"] = cell
		}
	}
	return result
}

// isTextSubCell returns true if the style indicates a text sub-cell.
func isTextSubCell(style string) bool {
	return containsStyleKey(style, "text")
}

// containsStyleKey checks if a draw.io style string starts with or contains the given key.
func containsStyleKey(style, key string) bool {
	// Style format: "key1;key2=val;key3=val;..."
	// "text" appears as a flag (no =) at the beginning.
	if style == key || style == key+";" {
		return true
	}
	if len(style) > len(key) && style[:len(key)+1] == key+";" {
		return true
	}
	return false
}

// hasSuffix is a simple string suffix check.
func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

// updateSubCells updates existing sub-cell values and adds/removes tech/desc cells.
func updateSubCells(root *etree.Element, parentCellID string, cells map[string]*etree.Element, data ElementData) {
	// Update title.
	if tc, ok := cells["title"]; ok {
		setAttr(tc, "value", data.Title)
	}

	// Update or add/remove technology cell.
	if tc, ok := cells["tech"]; ok {
		if data.Technology != "" {
			setAttr(tc, "value", "["+data.Technology+"]")
		} else {
			root.RemoveChild(tc)
		}
	}

	// Update or add/remove description cell.
	if dc, ok := cells["desc"]; ok {
		if data.Description != "" {
			setAttr(dc, "value", data.Description)
		} else {
			root.RemoveChild(dc)
		}
	}
}

// UpdateElementKind updates the bausteinsicht_kind attribute and the mxCell style
// of an existing element. If style is empty, the mxCell style is not changed.
func (p *Page) UpdateElementKind(id, kind, style string) {
	obj := p.FindElement(id)
	if obj == nil {
		return
	}
	setAttr(obj, "bausteinsicht_kind", kind)
	cell := obj.FindElement("mxCell")
	if cell != nil && style != "" {
		setAttr(cell, "style", style)
	}
}

// DeleteElement removes the <object> element with the given bausteinsicht_id.
// It also removes any child text sub-cells and mxCell connectors that reference
// this element as source or target.
func (p *Page) DeleteElement(id string) {
	root := p.Root()
	if root == nil {
		return
	}

	obj := p.FindElement(id)
	if obj != nil {
		cellID := obj.SelectAttrValue("id", "")
		root.RemoveChild(obj)

		// Remove child text sub-cells (their parent references the cell ID).
		if cellID != "" {
			removeChildCells(root, cellID)
		}
	}

	for _, cell := range root.SelectElements("mxCell") {
		src := cell.SelectAttrValue("source", "")
		dst := cell.SelectAttrValue("target", "")
		if src == id || dst == id {
			root.RemoveChild(cell)
		}
	}
}

// removeChildCells removes all mxCell elements whose parent attribute matches parentID.
func removeChildCells(root *etree.Element, parentID string) {
	var toRemove []*etree.Element
	for _, cell := range root.SelectElements("mxCell") {
		if cell.SelectAttrValue("parent", "") == parentID {
			toRemove = append(toRemove, cell)
		}
	}
	for _, cell := range toRemove {
		root.RemoveChild(cell)
	}
}

// ReadElementFields extracts title, technology, and description from an element.
// It first looks for child text sub-cells; if none are found, falls back to
// parsing the HTML label (backward compatibility).
func (p *Page) ReadElementFields(obj *etree.Element) (title, technology, description string) {
	cellID := obj.SelectAttrValue("id", "")
	root := p.Root()
	childCells := findChildTextCells(root, cellID)

	if len(childCells) > 0 {
		if tc, ok := childCells["title"]; ok {
			title = tc.SelectAttrValue("value", "")
		}
		if tc, ok := childCells["tech"]; ok {
			technology = trimBrackets(tc.SelectAttrValue("value", ""))
		}
		if dc, ok := childCells["desc"]; ok {
			description = dc.SelectAttrValue("value", "")
		}
		return title, technology, description
	}

	// Fallback: parse HTML label (backward compat).
	label := obj.SelectAttrValue("label", "")
	return ParseLabel(label)
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

// setStyleFlags sets key=value flags in a draw.io style string,
// replacing any existing value for each key.
func setStyleFlags(style string, flags ...string) string {
	for _, flag := range flags {
		parts := strings.SplitN(flag, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		// Remove existing key=value pair.
		segments := strings.Split(style, ";")
		var filtered []string
		for _, seg := range segments {
			if seg == "" {
				continue
			}
			if strings.HasPrefix(seg, key+"=") {
				continue
			}
			filtered = append(filtered, seg)
		}
		filtered = append(filtered, flag)
		style = strings.Join(filtered, ";") + ";"
	}
	return style
}
