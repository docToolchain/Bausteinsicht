package sync

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/beevik/etree"
	"github.com/docToolchain/Bausteinsicht/internal/drawio"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// docLinkPrefix marks synthetic clickable icon cells created for #543
// (element and view documentation-drilldown links) so they are excluded
// from reverse-sync element/relationship extraction and other structural
// scanners, the same way "nav-back-" back-navigation buttons are (#205).
const docLinkPrefix = "doclink-"

const (
	docIconSize = 18.0
	docIconGap  = 2.0
)

// docLinkGlyph returns the icon glyph and fill/stroke colors for a DocLink
// type. Unknown or empty types fall back to a generic link glyph so an icon
// is always rendered rather than silently dropped — DocLink.Type is a free
// string, not a closed schema enum (see model.DocLink).
func docLinkGlyph(linkType string) (glyph, fill, stroke string) {
	switch linkType {
	case "arc42":
		return "\U0001F4D6", "#dae8fc", "#6c8ebf" // 📖
	case "adr":
		return "⚖", "#dae8fc", "#6c8ebf" // ⚖
	case "persona":
		return "\U0001F464", "#dae8fc", "#6c8ebf" // 👤
	case "prd":
		return "\U0001F4CB", "#dae8fc", "#6c8ebf" // 📋
	case "spec":
		return "\U0001F4C4", "#dae8fc", "#6c8ebf" // 📄
	default:
		return "\U0001F517", "#dae8fc", "#6c8ebf" // 🔗
	}
}

// docIcon describes one clickable icon to render as its own top-level
// <object>, positioned via mxGraph's parent-ID attribute rather than XML
// nesting (see createBackNavButton/createInfoBox for the same convention;
// bare <mxCell> cells, as used by the older AddStatusBadge, cannot carry a
// link attribute — only <object> wrappers can, per element.go/forward.go's
// existing use of the `link` attribute throughout this codebase).
type docIcon struct {
	id      string // globally unique cell id; always starts with docLinkPrefix
	glyph   string
	fill    string
	stroke  string
	href    string // may be empty: the icon still renders, just isn't clickable
	tooltip string
}

// createDocIcon creates one clickable icon <object> as a direct child of
// root, parented (via mxGraph's parent-ID attribute) to parentCellID at the
// given position — parentCellID may be "1" for page-absolute placement
// (view-level icons) or an element's own on-page cell ID for placement
// relative to that element's bounding box (element-level icons).
func createDocIcon(root *etree.Element, parentCellID string, icon docIcon, x, y float64) {
	obj := root.CreateElement("object")
	obj.CreateAttr("id", icon.id)
	if icon.href != "" {
		obj.CreateAttr("link", icon.href)
	}
	if icon.tooltip != "" {
		obj.CreateAttr("tooltip", icon.tooltip)
	}

	cell := obj.CreateElement("mxCell")
	cell.CreateAttr("value", icon.glyph)
	cell.CreateAttr("style", fmt.Sprintf(
		"rounded=1;fillColor=%s;strokeColor=%s;fontSize=11;fontColor=#000000;"+
			"whiteSpace=wrap;overflow=hidden;connectable=0;align=center;verticalAlign=middle;html=1;",
		icon.fill, icon.stroke))
	cell.CreateAttr("vertex", "1")
	cell.CreateAttr("parent", parentCellID)

	geom := cell.CreateElement("mxGeometry")
	geom.CreateAttr("x", strconv.FormatFloat(x, 'f', 0, 64))
	geom.CreateAttr("y", strconv.FormatFloat(y, 'f', 0, 64))
	geom.CreateAttr("width", strconv.FormatFloat(docIconSize, 'f', 0, 64))
	geom.CreateAttr("height", strconv.FormatFloat(docIconSize, 'f', 0, 64))
	geom.CreateAttr("as", "geometry")
}

// removeDocIcons removes every existing doc-link icon object whose id
// starts with the given prefix, so re-sync doesn't accumulate stale or
// duplicate icons (mirrors the decision-badge remove-then-recreate
// lifecycle this replaces).
func removeDocIcons(root *etree.Element, prefix string) {
	for _, obj := range root.SelectElements("object") {
		if strings.HasPrefix(obj.SelectAttrValue("id", ""), prefix) {
			root.RemoveChild(obj)
		}
	}
}

// buildElementDocIcons returns the ordered list of doc-link icons for one
// element: a Link icon (if set — independent of any auto-generated
// drill-down link that may separately own the element shape's own `link`
// attribute, see ADR-009's "Extension: Doc/ADR Drilldown (#543)"), one per
// Decisions entry (clickable when the ADR record has a file path), then one
// per DocLinks entry — in that left-to-right rendering order.
func buildElementDocIcons(cellID string, elem *model.Element, decisionMap map[string]*model.DecisionRecord) []docIcon {
	var icons []docIcon

	if elem.Link != "" {
		glyph, fill, stroke := docLinkGlyph("")
		icons = append(icons, docIcon{
			id:      docLinkPrefix + cellID + "-link",
			glyph:   glyph,
			fill:    fill,
			stroke:  stroke,
			href:    elem.Link,
			tooltip: "Open linked document",
		})
	}

	for i, decisionID := range elem.Decisions {
		color := model.DecisionBadgeColor("")
		href := ""
		tooltip := decisionID
		if d, ok := decisionMap[decisionID]; ok {
			color = model.DecisionBadgeColor(d.Status)
			href = d.FilePath
			tooltip = fmt.Sprintf("%s: %s", d.ID, d.Title)
		}
		icons = append(icons, docIcon{
			id:      fmt.Sprintf("%s%s-decision-%d", docLinkPrefix, cellID, i),
			glyph:   "⚖", // ⚖ — same glyph decision badges have always used
			fill:    color,
			stroke:  color,
			href:    href,
			tooltip: tooltip,
		})
	}

	for i, dl := range elem.DocLinks {
		glyph, fill, stroke := docLinkGlyph(dl.Type)
		tooltip := dl.Title
		if tooltip == "" {
			tooltip = dl.Type
		}
		icons = append(icons, docIcon{
			id:      fmt.Sprintf("%s%s-doc-%d", docLinkPrefix, cellID, i),
			glyph:   glyph,
			fill:    fill,
			stroke:  stroke,
			href:    dl.Href,
			tooltip: tooltip,
		})
	}

	return icons
}

// placeElementIconRow creates the given icons as a row anchored to the
// parent element's top-right corner, growing leftward as more icons are
// added — matching the positioning convention the older decision badges
// used (AddDecisionBadges).
func placeElementIconRow(root *etree.Element, parentCellID string, icons []docIcon, parentWidth float64) {
	if len(icons) == 0 {
		return
	}
	startX := parentWidth - (docIconSize+docIconGap)*float64(len(icons))
	for i, icon := range icons {
		x := startX + float64(i)*(docIconSize+docIconGap)
		createDocIcon(root, parentCellID, icon, x, docIconGap)
	}
}

// synchronizeDocLinkIcons updates the clickable documentation-link icons on
// every element of a page: a Link icon, clickable decision badges, and
// DocLinks icons (#543). Old icons are removed and recreated each sync —
// the row position is purely a function of the element's current width and
// icon count, so recomputing it every sync is idempotent (no drift).
func synchronizeDocLinkIcons(page *drawio.Page, m *model.BausteinsichtModel) {
	if m == nil {
		return
	}
	decisionMap := make(map[string]*model.DecisionRecord, len(m.Specification.Decisions))
	for i := range m.Specification.Decisions {
		decisionMap[m.Specification.Decisions[i].ID] = &m.Specification.Decisions[i]
	}

	root := page.Root()
	if root == nil {
		return
	}

	for _, obj := range root.SelectElements("object") {
		elemID := obj.SelectAttrValue("bausteinsicht_id", "")
		if elemID == "" {
			continue
		}
		elem, ok := findElementByID(m, elemID)
		if !ok || elem == nil {
			continue
		}
		cellID := obj.SelectAttrValue("id", "")
		if cellID == "" {
			continue
		}

		removeDocIcons(root, docLinkPrefix+cellID+"-")

		icons := buildElementDocIcons(cellID, elem, decisionMap)
		if len(icons) == 0 {
			continue
		}

		width := 100.0
		if cell := obj.SelectElement("mxCell"); cell != nil {
			if geo := cell.SelectElement("mxGeometry"); geo != nil {
				width = parseFloat(geo.SelectAttrValue("width", "100"))
			}
		}
		placeElementIconRow(root, cellID, icons, width)
	}
}

// synchronizeViewDocLinkIcons updates the clickable icon row for a view's
// page-level DocLinks (#543). The row is positioned once — below the
// metadata/legend boxes, mirroring their own placement convention, since
// this codebase has no notion of a "free diagram corner", only the max
// Y/X of existing content (see computeMaxY/computeContentWidth) — and its
// position is then reused on every subsequent sync (read back from an
// existing icon's geometry) rather than recomputed, so it never drifts.
func synchronizeViewDocLinkIcons(page *drawio.Page, viewID string, view model.View) {
	root := page.Root()
	if root == nil {
		return
	}
	prefix := docLinkPrefix + "view-" + viewID + "-"

	var existing []*etree.Element
	for _, obj := range root.SelectElements("object") {
		if strings.HasPrefix(obj.SelectAttrValue("id", ""), prefix) {
			existing = append(existing, obj)
		}
	}

	if len(view.DocLinks) == 0 {
		for _, obj := range existing {
			root.RemoveChild(obj)
		}
		return
	}

	x, y := metadataX, 0.0
	positionKnown := false
	if len(existing) > 0 {
		if cell := existing[0].SelectElement("mxCell"); cell != nil {
			if geo := cell.SelectElement("mxGeometry"); geo != nil {
				x = parseFloat(geo.SelectAttrValue("x", "0"))
				y = parseFloat(geo.SelectAttrValue("y", "0"))
				positionKnown = true
			}
		}
	}
	if !positionKnown {
		metaCellID := metadataPrefix + viewID
		legendCellID := legendPrefix + viewID
		y = computeMaxY(page, metaCellID, legendCellID) + infoBoxGap
		x = metadataX
	}

	for _, obj := range existing {
		root.RemoveChild(obj)
	}

	for i, dl := range view.DocLinks {
		glyph, fill, stroke := docLinkGlyph(dl.Type)
		tooltip := dl.Title
		if tooltip == "" {
			tooltip = dl.Type
		}
		icon := docIcon{
			id:      fmt.Sprintf("%s%d", prefix, i),
			glyph:   glyph,
			fill:    fill,
			stroke:  stroke,
			href:    dl.Href,
			tooltip: tooltip,
		}
		createDocIcon(root, "1", icon, x+float64(i)*(docIconSize+docIconGap), y)
	}
}
