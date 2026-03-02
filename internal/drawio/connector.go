package drawio

import (
	"fmt"

	"github.com/beevik/etree"
)

// ConnectorData holds data for creating/updating connectors.
type ConnectorData struct {
	From      string // source bausteinsicht_id
	To        string // target bausteinsicht_id
	Label     string // display label on the connector
	SourceRef string // source cell ID (defaults to From if empty)
	TargetRef string // target cell ID (defaults to To if empty)
	Index     int    // relationship index for disambiguation (0-based)
}

// connectorID returns the canonical ID for a connector between two elements.
// The index disambiguates multiple relationships between the same pair.
func connectorID(from, to string, index int) string {
	return fmt.Sprintf("rel-%s-%s-%d", from, to, index)
}

// CreateConnector creates an edge mxCell connecting From to To.
// Connectors always use parent="1" regardless of container nesting.
func (p *Page) CreateConnector(data ConnectorData, style string) {
	root := p.Root()
	if root == nil {
		return
	}

	srcRef := data.SourceRef
	if srcRef == "" {
		srcRef = data.From
	}
	tgtRef := data.TargetRef
	if tgtRef == "" {
		tgtRef = data.To
	}

	cell := root.CreateElement("mxCell")
	cell.CreateAttr("id", connectorID(srcRef, tgtRef, data.Index))
	cell.CreateAttr("value", data.Label)
	cell.CreateAttr("style", style)
	cell.CreateAttr("edge", "1")
	cell.CreateAttr("source", srcRef)
	cell.CreateAttr("target", tgtRef)
	cell.CreateAttr("parent", "1")

	geom := cell.CreateElement("mxGeometry")
	geom.CreateAttr("relative", "1")
	geom.CreateAttr("as", "geometry")
}

// FindConnector returns the mxCell edge with id="rel-<from>-<to>-<index>", or nil.
func (p *Page) FindConnector(from, to string, index int) *etree.Element {
	root := p.Root()
	if root == nil {
		return nil
	}
	id := connectorID(from, to, index)
	for _, cell := range root.SelectElements("mxCell") {
		if cell.SelectAttrValue("id", "") == id {
			return cell
		}
	}
	return nil
}

// FindAllConnectors returns all mxCell elements with edge="1".
func (p *Page) FindAllConnectors() []*etree.Element {
	root := p.Root()
	if root == nil {
		return nil
	}
	var result []*etree.Element
	for _, cell := range root.SelectElements("mxCell") {
		if cell.SelectAttrValue("edge", "") == "1" {
			result = append(result, cell)
		}
	}
	return result
}

// UpdateConnectorLabel sets the value attribute on the connector between from and to.
func (p *Page) UpdateConnectorLabel(from, to string, index int, label string) {
	conn := p.FindConnector(from, to, index)
	if conn == nil {
		return
	}
	attr := conn.SelectAttr("value")
	if attr != nil {
		attr.Value = label
	} else {
		conn.CreateAttr("value", label)
	}
}

// DeleteConnector removes the connector between from and to at the given index.
func (p *Page) DeleteConnector(from, to string, index int) {
	root := p.Root()
	if root == nil {
		return
	}
	id := connectorID(from, to, index)
	for _, cell := range root.SelectElements("mxCell") {
		if cell.SelectAttrValue("id", "") == id {
			root.RemoveChild(cell)
			return
		}
	}
}

// DeleteConnectorsFor removes all connectors where source or target matches elementID.
func (p *Page) DeleteConnectorsFor(elementID string) {
	root := p.Root()
	if root == nil {
		return
	}
	var toRemove []*etree.Element
	for _, cell := range root.SelectElements("mxCell") {
		if cell.SelectAttrValue("edge", "") != "1" {
			continue
		}
		src := cell.SelectAttrValue("source", "")
		tgt := cell.SelectAttrValue("target", "")
		if src == elementID || tgt == elementID {
			toRemove = append(toRemove, cell)
		}
	}
	for _, cell := range toRemove {
		root.RemoveChild(cell)
	}
}
