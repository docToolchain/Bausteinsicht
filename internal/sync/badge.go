package sync

import (
	"fmt"

	"github.com/beevik/etree"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// AddStatusBadge adds a status badge as a child cell to an element shape.
// The badge is positioned in the top-right corner of the element.
func AddStatusBadge(elementCell *etree.Element, status string) {
	if status == "" {
		return // No badge for unset status
	}

	// Get element width (or use default if not set)
	width := getAttrFloat(elementCell, "width", 100)

	// Badge dimensions and positioning
	badgeWidth := 60.0
	badgeHeight := 20.0
	badgeX := width - badgeWidth - 2
	badgeY := 2.0

	// Create badge cell
	badge := etree.NewElement("mxCell")
	badge.CreateAttr("id", fmt.Sprintf("%s_badge", getAttr(elementCell, "id")))
	badge.CreateAttr("value", status)
	badge.CreateAttr("style", fmt.Sprintf(
		"rounded=1;fillColor=%s;strokeColor=%s;fontSize=11;fontColor=#000000;"+
			"whiteSpace=wrap;overflow=hidden;connectable=0",
		model.StatusColor(status), model.StatusColor(status)))
	badge.CreateAttr("vertex", "1")
	badge.CreateAttr("parent", getAttr(elementCell, "id"))

	// Geometry for the badge
	geom := etree.NewElement("mxGeometry")
	geom.CreateAttr("x", fmt.Sprintf("%.0f", badgeX))
	geom.CreateAttr("y", fmt.Sprintf("%.0f", badgeY))
	geom.CreateAttr("width", fmt.Sprintf("%.0f", badgeWidth))
	geom.CreateAttr("height", fmt.Sprintf("%.0f", badgeHeight))
	geom.CreateAttr("as", "geometry")

	badge.AddChild(geom)
	elementCell.AddChild(badge)
}

// getAttr retrieves a string attribute value
func getAttr(el *etree.Element, name string) string {
	attr := el.SelectAttr(name)
	if attr == nil {
		return ""
	}
	return attr.Value
}

// getAttrFloat retrieves a float attribute value with default
func getAttrFloat(el *etree.Element, name string, defaultVal float64) float64 {
	attr := el.SelectAttr(name)
	if attr == nil {
		return defaultVal
	}
	var val float64
	_, _ = fmt.Sscanf(attr.Value, "%f", &val)
	return val
}
