package sync

import (
	"html"
	"sort"
	"strings"

	"github.com/beevik/etree"

	"github.com/docToolchain/Bausteinsicht/internal/drawio"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// techLabelSuffix ends the cell ID of a connector's technology label:
// "<connector cell ID>-tech". Connector IDs end in the relationship index, so
// the suffix cannot collide with one.
const techLabelSuffix = "-tech"

// techLabelStyle is a second, smaller italic label on the connector, below its
// own label.
const techLabelStyle = "edgeLabel;html=1;align=center;verticalAlign=middle;resizable=0;points=[];" +
	"fontSize=10;fontStyle=2;labelBackgroundColor=none;"

// synchronizeConnectorTechnology shows the technology of each relationship as a
// second label "[technology]" on its connector (#642). The label is a child
// cell of the connector, so the connector's own label — the only thing reverse
// sync reads — stays the plain relationship label.
//
// Like the doc-link icons, the labels are reconciled from the model on every
// sync instead of being tracked in the sync state: labels whose connector or
// technology is gone are removed, the others are created or updated in place.
// That also keeps them attached when a connector gets a new cell ID because
// its relationship index shifted.
func synchronizeConnectorTechnology(page *drawio.Page, viewID string, conns []viewConnector) {
	root := page.Root()
	if root == nil {
		return
	}

	wanted := make(map[string]string) // label cell ID → text
	parent := make(map[string]string) // label cell ID → connector cell ID
	for _, vc := range conns {
		if vc.technology == "" {
			continue
		}
		srcRef, tgtRef := scopedCellID(viewID, vc.from), scopedCellID(viewID, vc.to)
		if page.FindConnector(srcRef, tgtRef, vc.index) == nil {
			continue
		}
		connID := drawio.ConnectorID(srcRef, tgtRef, vc.index)
		wanted[connID+techLabelSuffix] = "[" + vc.technology + "]"
		parent[connID+techLabelSuffix] = connID
	}

	// Connector IDs on a view page start with "rel-<view>--"; without views the
	// IDs are unscoped and start with "rel-". Only edge-label cells count, so
	// the sub-cell of an element that happens to be called "rel-…" is not ours.
	prefix := "rel-"
	if viewID != "" {
		prefix += viewID + "--"
	}
	existing := make(map[string]*etree.Element)
	for _, cell := range root.SelectElements("mxCell") {
		id := cell.SelectAttrValue("id", "")
		if !strings.HasPrefix(id, prefix) || !strings.HasSuffix(id, techLabelSuffix) ||
			!strings.HasPrefix(cell.SelectAttrValue("style", ""), "edgeLabel;") {
			continue
		}
		if _, keep := wanted[id]; keep {
			existing[id] = cell
		} else {
			root.RemoveChild(cell)
		}
	}

	ids := make([]string, 0, len(wanted))
	for id := range wanted {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		value := html.EscapeString(wanted[id])
		if cell, ok := existing[id]; ok {
			cell.CreateAttr("value", value)
			cell.CreateAttr("parent", parent[id])
			continue
		}
		createTechLabel(root, id, parent[id], value)
	}
}

func createTechLabel(root *etree.Element, id, connectorID, value string) {
	cell := root.CreateElement("mxCell")
	cell.CreateAttr("id", id)
	cell.CreateAttr("value", value)
	cell.CreateAttr("style", techLabelStyle)
	cell.CreateAttr("vertex", "1")
	cell.CreateAttr("connectable", "0")
	cell.CreateAttr("parent", connectorID)

	geom := cell.CreateElement("mxGeometry")
	geom.CreateAttr("relative", "1")
	geom.CreateAttr("as", "geometry")
	offset := geom.CreateElement("mxPoint")
	offset.CreateAttr("y", "14")
	offset.CreateAttr("as", "offset")
}

// legacyConnectors lists the connectors of the no-views mode: one per model
// relationship, with unscoped cell IDs.
func legacyConnectors(relationships []model.Relationship) []viewConnector {
	conns := make([]viewConnector, len(relationships))
	for i, r := range relationships {
		conns[i] = viewConnector{from: r.From, to: r.To, index: i, label: r.Label, technology: r.Technology}
	}
	return conns
}
