package drawio

import (
	"strconv"
	"strings"

	"github.com/beevik/etree"
)

// RenameElement gives the cells of the element oldID, and of every element
// below it, the ID newID: the bausteinsicht_id attribute, the page-scoped cell
// IDs ("<view>--<id>"), the sub-cells and icons that hang on those cells, and
// the connectors that start or end at them, so that the next sync finds the
// cells under the new name and keeps their geometry and style (#686). It
// returns how many element cells it renamed.
func (d *Document) RenameElement(oldID, newID string) int {
	renamed := 0
	for _, page := range d.Pages() {
		renamed += page.renameElement(oldID, newID)
	}
	return renamed
}

func (p *Page) renameElement(oldID, newID string) int {
	root := p.Root()
	if root == nil {
		return 0
	}
	scope := strings.TrimPrefix(p.ID(), "view-") + "--"
	ids := map[string]string{} // old cell ID → new cell ID
	renamed := 0

	// The element cells: carry the new bausteinsicht_id and the new scoped ID.
	for _, obj := range root.SelectElements("object") {
		b := obj.SelectAttrValue("bausteinsicht_id", "")
		if b != oldID && !strings.HasPrefix(b, oldID+".") {
			continue
		}
		nb := newID + b[len(oldID):]
		obj.CreateAttr("bausteinsicht_id", nb)
		renamed++
		if cell := obj.SelectAttrValue("id", ""); cell == scope+b {
			ids[cell] = scope + nb
		}
	}

	// Sub-cells and icons hang on a renamed cell (parent) and carry its ID in
	// their own; follow the parent links until nothing new is found.
	for changed := true; changed; {
		changed = false
		for _, el := range cellsOf(root) {
			id := el.SelectAttrValue("id", "")
			parent := el.SelectAttrValue("parent", "")
			if id == "" || ids[id] != "" || ids[parent] == "" || !strings.Contains(id, parent) {
				continue
			}
			ids[id] = strings.Replace(id, parent, ids[parent], 1)
			changed = true
		}
	}

	for _, el := range cellsOf(root) {
		id := el.SelectAttrValue("id", "")
		src, tgt := el.SelectAttrValue("source", ""), el.SelectAttrValue("target", "")
		if el.SelectAttrValue("edge", "") == "1" && id != "" {
			if index, ok := connectorIndex(id, src, tgt); ok {
				ids[id] = ConnectorID(mapped(ids, src), mapped(ids, tgt), index)
			}
		}
	}

	for _, el := range cellsOf(root) {
		if id := el.SelectAttrValue("id", ""); ids[id] != "" {
			el.CreateAttr("id", ids[id])
		}
		for _, attr := range []string{"parent", "source", "target"} {
			if v := el.SelectAttrValue(attr, ""); ids[v] != "" {
				el.CreateAttr(attr, ids[v])
			}
		}
	}
	// A cell wrapped in an <object> has no ID of its own; its links move with
	// the cells above.
	for _, obj := range root.SelectElements("object") {
		for _, cell := range obj.SelectElements("mxCell") {
			for _, attr := range []string{"parent", "source", "target"} {
				if v := cell.SelectAttrValue(attr, ""); ids[v] != "" {
					cell.CreateAttr(attr, ids[v])
				}
			}
		}
	}
	return renamed
}

// cellsOf returns the <object> and <mxCell> children of a page's root.
func cellsOf(root *etree.Element) []*etree.Element {
	var cells []*etree.Element
	for _, el := range root.ChildElements() {
		if el.Tag == "object" || el.Tag == "mxCell" {
			cells = append(cells, el)
		}
	}
	return cells
}

func mapped(ids map[string]string, id string) string {
	if n := ids[id]; n != "" {
		return n
	}
	return id
}

// connectorIndex reports whether id is the canonical ID of a connector from
// src to tgt ("rel-<src>-<tgt>-<index>") and returns the index.
func connectorIndex(id, src, tgt string) (int, bool) {
	prefix := "rel-" + src + "-" + tgt + "-"
	if !strings.HasPrefix(id, prefix) {
		return 0, false
	}
	index, err := strconv.Atoi(id[len(prefix):])
	return index, err == nil
}
