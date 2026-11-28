package drawio

import (
	"fmt"
	"math"
	"strconv"

	"github.com/beevik/etree"
)

// oppositeOffset is how far, in pixels, the waypoint of each connector of an
// opposite pair is moved off the line between the element centres: the two
// routes end up twice that apart, enough for two readable labels.
const oppositeOffset = 30

// oppositeLabelPosition is the position of the label along the connector, from
// -1 (at the source) to 1 (at the target). Both connectors of a pair put their
// label towards their own source, which are the two ends of the line: the
// labels do not overlap even when they are wider than the distance between the
// two routes (a vertical arrangement).
const oppositeLabelPosition = "-0.4"

// SeparateOppositeConnectors gives the connectors of opposite directions
// between the same two elements different routes (#717). Both would otherwise
// follow the same automatic route and their labels would be drawn on top of
// each other. Each one gets a waypoint in the middle between the two elements,
// moved by oppositeOffset to its own side of the line between the centres, and
// moves its label towards the source (oppositeLabelPosition).
// Connectors that already have waypoints (set by the user or by an earlier run)
// are left alone, so the call can be repeated, and a connector without a
// counterpart keeps the automatic route.
func (p *Page) SeparateOppositeConnectors() {
	root := p.Root()
	if root == nil {
		return
	}
	type pair struct{ lo, hi string }
	pairs := make(map[pair][]*etree.Element)
	for _, c := range p.FindAllConnectors() {
		src, tgt := c.SelectAttrValue("source", ""), c.SelectAttrValue("target", "")
		if src == "" || tgt == "" || src == tgt {
			continue
		}
		key := pair{src, tgt}
		if tgt < src {
			key = pair{tgt, src}
		}
		pairs[key] = append(pairs[key], c)
	}
	for key, conns := range pairs {
		forward, backward := false, false
		for _, c := range conns {
			if c.SelectAttrValue("source", "") == key.lo {
				forward = true
			} else {
				backward = true
			}
		}
		if !forward || !backward {
			continue
		}
		lo, okLo := cellCentre(root, key.lo)
		hi, okHi := cellCentre(root, key.hi)
		if !okLo || !okHi {
			continue
		}
		dx, dy := hi[0]-lo[0], hi[1]-lo[1]
		length := math.Hypot(dx, dy)
		if length == 0 {
			continue
		}
		nx, ny := -dy/length, dx/length
		for _, c := range conns {
			if hasWaypoints(c) {
				continue
			}
			side := 1.0
			if c.SelectAttrValue("source", "") != key.lo {
				side = -1
			}
			addWaypoint(c, (lo[0]+hi[0])/2+side*oppositeOffset*nx, (lo[1]+hi[1])/2+side*oppositeOffset*ny)
		}
	}
}

func hasWaypoints(conn *etree.Element) bool {
	return conn.FindElement("mxGeometry/Array[@as='points']") != nil
}

func addWaypoint(conn *etree.Element, x, y float64) {
	geom := conn.SelectElement("mxGeometry")
	if geom == nil {
		geom = conn.CreateElement("mxGeometry")
		geom.CreateAttr("relative", "1")
		geom.CreateAttr("as", "geometry")
	}
	if geom.SelectAttr("x") == nil {
		geom.CreateAttr("x", oppositeLabelPosition)
	}
	arr := geom.CreateElement("Array")
	arr.CreateAttr("as", "points")
	pt := arr.CreateElement("mxPoint")
	pt.CreateAttr("x", fmt.Sprintf("%.0f", x))
	pt.CreateAttr("y", fmt.Sprintf("%.0f", y))
}

// cellCentre returns the centre of the cell with the given ID in page
// coordinates: the geometry of a nested cell is relative to its container.
func cellCentre(root *etree.Element, id string) ([2]float64, bool) {
	x, y, w, h, ok := cellBounds(root, id, 0)
	return [2]float64{x + w/2, y + h/2}, ok
}

func cellBounds(root *etree.Element, id string, depth int) (x, y, w, h float64, ok bool) {
	if depth > 50 {
		return 0, 0, 0, 0, false
	}
	var cell *etree.Element
	for _, el := range root.ChildElements() {
		if el.SelectAttrValue("id", "") != id {
			continue
		}
		cell = el
		if el.Tag == "object" {
			cell = el.SelectElement("mxCell")
		}
		break
	}
	if cell == nil {
		return 0, 0, 0, 0, false
	}
	geom := cell.SelectElement("mxGeometry")
	if geom == nil {
		return 0, 0, 0, 0, false
	}
	num := func(name string) float64 {
		v, _ := strconv.ParseFloat(geom.SelectAttrValue(name, "0"), 64)
		return v
	}
	x, y, w, h = num("x"), num("y"), num("width"), num("height")
	if parent := cell.SelectAttrValue("parent", ""); parent != "" && parent != "0" && parent != "1" {
		px, py, _, _, pok := cellBounds(root, parent, depth+1)
		if !pok {
			return 0, 0, 0, 0, false
		}
		x, y = x+px, y+py
	}
	return x, y, w, h, true
}
