package sync

import (
	"strconv"

	"github.com/docToolchain/Bauteinsicht/internal/drawio"
	"github.com/docToolchain/Bauteinsicht/internal/model"
)

const (
	newElementMarker = "strokeColor=#FF0000;dashed=1;"
	elementGap       = 40.0
	defaultWidth     = 120.0
	defaultHeight    = 60.0
)

// ForwardResult summarises the changes applied to a draw.io document.
type ForwardResult struct {
	ElementsCreated   int
	ElementsUpdated   int
	ElementsDeleted   int
	ConnectorsCreated int
	ConnectorsUpdated int
	ConnectorsDeleted int
	Warnings          []string
}

// ApplyForward applies ModelElementChanges and ModelRelationshipChanges from cs
// to doc, using templates for styles and m for element data.
func ApplyForward(
	cs *ChangeSet,
	doc *drawio.Document,
	templates *drawio.TemplateSet,
	m *model.BausteinsichtModel,
) *ForwardResult {
	result := &ForwardResult{}
	flat := model.FlattenElements(m)

	page := firstPage(doc)
	if page == nil {
		result.Warnings = append(result.Warnings, "no page found in document")
		return result
	}

	placement := computePlacement(page)

	for _, ch := range cs.ModelElementChanges {
		switch ch.Type {
		case Added:
			applyElementAdded(ch.ID, page, templates, flat, &placement, result)
		case Modified:
			applyElementModified(ch, page, flat, result)
		case Deleted:
			page.DeleteElement(ch.ID)
			result.ElementsDeleted++
		}
	}

	for _, ch := range cs.ModelRelationshipChanges {
		switch ch.Type {
		case Added:
			applyRelAdded(ch, page, templates, result)
		case Modified:
			page.UpdateConnectorLabel(ch.From, ch.To, ch.NewValue)
			result.ConnectorsUpdated++
		case Deleted:
			page.DeleteConnector(ch.From, ch.To)
			result.ConnectorsDeleted++
		}
	}

	return result
}

// firstPage returns the first page in doc, or nil if there are none.
func firstPage(doc *drawio.Document) *drawio.Page {
	pages := doc.Pages()
	if len(pages) == 0 {
		return nil
	}
	return pages[0]
}

// placement tracks where the next new element should be placed.
type placement struct {
	nextX float64
	nextY float64
}

// computePlacement scans existing elements on a page and returns a placement
// state positioned one row below all existing content.
func computePlacement(page *drawio.Page) placement {
	maxY := 0.0
	for _, obj := range page.FindAllElements() {
		cell := obj.FindElement("mxCell")
		if cell == nil {
			continue
		}
		geo := cell.FindElement("mxGeometry")
		if geo == nil {
			continue
		}
		y, _ := strconv.ParseFloat(geo.SelectAttrValue("y", "0"), 64)
		h, _ := strconv.ParseFloat(geo.SelectAttrValue("height", "0"), 64)
		if bottom := y + h; bottom > maxY {
			maxY = bottom
		}
	}

	startY := maxY
	if maxY > 0 {
		startY = maxY + elementGap
	}
	return placement{nextX: elementGap, nextY: startY}
}

// applyElementAdded creates a new element on page with a visual new-element marker.
func applyElementAdded(
	id string,
	page *drawio.Page,
	templates *drawio.TemplateSet,
	flat map[string]*model.Element,
	pl *placement,
	result *ForwardResult,
) {
	elem, ok := flat[id]
	if !ok {
		result.Warnings = append(result.Warnings, "element not found in model: "+id)
		return
	}

	ts, ok := templates.GetStyle(elem.Kind)
	if !ok {
		ts = drawio.TemplateStyle{Width: defaultWidth, Height: defaultHeight}
		result.Warnings = append(result.Warnings, "no template style for kind: "+elem.Kind)
	}

	style := ts.Style + newElementMarker

	width := ts.Width
	if width == 0 {
		width = defaultWidth
	}
	height := ts.Height
	if height == 0 {
		height = defaultHeight
	}

	data := drawio.ElementData{
		ID:          id,
		Kind:        elem.Kind,
		Title:       elem.Title,
		Technology:  elem.Technology,
		Description: elem.Description,
		X:           pl.nextX,
		Y:           pl.nextY,
		Width:       width,
		Height:      height,
	}

	if err := page.CreateElement(data, style); err != nil {
		result.Warnings = append(result.Warnings, "failed to create element "+id+": "+err.Error())
		return
	}

	pl.nextX += width + elementGap
	result.ElementsCreated++
}

// applyElementModified updates the changed field of an existing element.
func applyElementModified(
	ch ElementChange,
	page *drawio.Page,
	flat map[string]*model.Element,
	result *ForwardResult,
) {
	elem, ok := flat[ch.ID]
	if !ok {
		result.Warnings = append(result.Warnings, "element not found in model for update: "+ch.ID)
		return
	}

	data := drawio.ElementData{
		ID:          ch.ID,
		Title:       elem.Title,
		Technology:  elem.Technology,
		Description: elem.Description,
	}

	page.UpdateElement(ch.ID, data)
	result.ElementsUpdated++
}

// applyRelAdded creates a new connector on page.
func applyRelAdded(
	ch RelationshipChange,
	page *drawio.Page,
	templates *drawio.TemplateSet,
	result *ForwardResult,
) {
	style := templates.GetConnectorStyle()
	data := drawio.ConnectorData{
		From:  ch.From,
		To:    ch.To,
		Label: ch.NewValue,
	}
	page.CreateConnector(data, style)
	result.ConnectorsCreated++
}
