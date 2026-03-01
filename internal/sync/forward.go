package sync

import (
	"strconv"
	"strings"

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
// When the model defines views, elements and relationships are placed on their
// corresponding view pages. Without views, falls back to the first page.
func ApplyForward(
	cs *ChangeSet,
	doc *drawio.Document,
	templates *drawio.TemplateSet,
	m *model.BausteinsichtModel,
) *ForwardResult {
	result := &ForwardResult{}
	flat := model.FlattenElements(m)

	if len(m.Views) == 0 {
		applyForwardToPage(cs, doc, templates, flat, nil, result)
		return result
	}

	applyForwardPerView(cs, doc, templates, flat, m, result)
	return result
}

// applyForwardToPage applies all changes to a single page (legacy/no-views mode).
func applyForwardToPage(
	cs *ChangeSet,
	doc *drawio.Document,
	templates *drawio.TemplateSet,
	flat map[string]*model.Element,
	elemFilter map[string]bool,
	result *ForwardResult,
) {
	page := firstPage(doc)
	if page == nil {
		result.Warnings = append(result.Warnings, "no page found in document")
		return
	}
	applyChangesToPage(cs, page, templates, flat, elemFilter, "", "", result)

	// Reconcile orphaned elements: remove any element on the page whose
	// bausteinsicht_id is not present in the current model. This handles
	// cases where the sync state is missing or the model was emptied. (#110)
	reconcileOrphanedElements(page, flat, result)
}

// applyForwardPerView iterates over model views and applies changes per page.
func applyForwardPerView(
	cs *ChangeSet,
	doc *drawio.Document,
	templates *drawio.TemplateSet,
	flat map[string]*model.Element,
	m *model.BausteinsichtModel,
	result *ForwardResult,
) {
	for viewID, view := range m.Views {
		pageID := "view-" + viewID
		page := doc.GetPage(pageID)
		if page == nil {
			result.Warnings = append(result.Warnings,
				"no page found for view: "+viewID)
			continue
		}

		viewCopy := view
		resolved, err := model.ResolveView(m, &viewCopy)
		if err != nil {
			result.Warnings = append(result.Warnings,
				"resolving view "+viewID+": "+err.Error())
			continue
		}

		elemSet := make(map[string]bool, len(resolved))
		for _, id := range resolved {
			elemSet[id] = true
		}

		scopeID := view.Scope
		if scopeID != "" {
			createScopeBoundary(scopeID, viewID, page, templates, flat, result)
		}

		applyChangesToPage(cs, page, templates, flat, elemSet, viewID, scopeID, result)

		// Reconciliation: remove elements on the page that are no longer
		// in the resolved view (e.g., after exclude list changes). #102
		reconcileViewPage(page, elemSet, flat, scopeID, viewID, result)
	}
}

// scopedCellID returns a page-scoped cell ID to ensure file-wide uniqueness.
// If viewID is empty, returns the raw element ID (legacy mode).
func scopedCellID(viewID, elemID string) string {
	if viewID == "" {
		return elemID
	}
	return viewID + "--" + elemID
}

// isChildOf returns true if id is a direct or nested child of parentID.
// Example: isChildOf("shop.api", "shop") → true
func isChildOf(id, parentID string) bool {
	return strings.HasPrefix(id, parentID+".")
}

// liftEndpoint returns id if it is in elemFilter. Otherwise it walks up the
// parent chain (by removing the last dot-segment) until a parent is found in
// the filter. Returns "" if no ancestor is present on the page.
func liftEndpoint(id string, elemFilter map[string]bool) string {
	if elemFilter[id] {
		return id
	}
	for {
		dot := strings.LastIndex(id, ".")
		if dot < 0 {
			return ""
		}
		id = id[:dot]
		if elemFilter[id] {
			return id
		}
	}
}

// createScopeBoundary creates a boundary/swimlane element for the scope element
// of a view (e.g., the parent system in a container view).
func createScopeBoundary(
	scopeID string,
	viewID string,
	page *drawio.Page,
	templates *drawio.TemplateSet,
	flat map[string]*model.Element,
	result *ForwardResult,
) {
	// Skip if already present on the page.
	if page.FindElement(scopeID) != nil {
		return
	}

	elem, ok := flat[scopeID]
	if !ok {
		result.Warnings = append(result.Warnings, "scope element not found in model: "+scopeID)
		return
	}

	boundaryKind := elem.Kind + "_boundary"
	ts, ok := templates.GetBoundaryStyle(boundaryKind)
	if !ok {
		ts = drawio.TemplateStyle{Width: 400, Height: 300}
		result.Warnings = append(result.Warnings, "no boundary template for kind: "+boundaryKind)
	}

	style := ts.Style
	width := ts.Width
	if width == 0 {
		width = 400
	}
	height := ts.Height
	if height == 0 {
		height = 300
	}

	data := drawio.ElementData{
		ID:          scopeID,
		CellID:      scopedCellID(viewID, scopeID),
		Kind:        boundaryKind,
		Title:       elem.Title,
		Technology:  elem.Technology,
		Description: elem.Description,
		X:           elementGap,
		Y:           elementGap,
		Width:       width,
		Height:      height,
	}

	if err := page.CreateElement(data, style); err != nil {
		result.Warnings = append(result.Warnings, "failed to create scope boundary "+scopeID+": "+err.Error())
	}
}

// applyChangesToPage applies element and relationship changes to a single page.
// If elemFilter is nil, all changes are applied. Otherwise only elements in the
// filter set are processed, and relationships are only created when both
// endpoints are in the filter set.
// viewID is used to scope cell IDs for file-wide uniqueness (empty = legacy).
// scopeID identifies the boundary element for parenting children (empty = no scope).
func applyChangesToPage(
	cs *ChangeSet,
	page *drawio.Page,
	templates *drawio.TemplateSet,
	flat map[string]*model.Element,
	elemFilter map[string]bool,
	viewID string,
	scopeID string,
	result *ForwardResult,
) {
	pl := computePlacement(page)

	for _, ch := range cs.ModelElementChanges {
		switch ch.Type {
		case Added:
			if elemFilter != nil && !elemFilter[ch.ID] {
				continue
			}
			applyElementAdded(ch.ID, viewID, scopeID, page, templates, flat, &pl, result)
		case Modified:
			if elemFilter != nil && !elemFilter[ch.ID] && ch.ID != scopeID {
				continue
			}
			applyElementModified(ch, page, templates, flat, result)
		case Deleted:
			// Deleted elements are removed from all pages where they exist,
			// regardless of the current view filter — a deleted element is
			// no longer in the model, so it can't appear in any view's
			// resolved set. We just check if it exists on this page.
			cellID := scopedCellID(viewID, ch.ID)
			if page.FindElement(ch.ID) != nil {
				// Delete connectors referencing this element's cell ID before
				// removing the element itself. (#101)
				result.ConnectorsDeleted += countConnectorsFor(page, cellID)
				page.DeleteConnectorsFor(cellID)
				page.DeleteElement(ch.ID)
				result.ElementsDeleted++
			}
		}
	}

	liftedSeen := make(map[string]bool)

	// Process relationships in two passes: direct first, then lifted.
	// This ensures that when a direct relationship (e.g., api→db) and a
	// lifted relationship (e.g., api.catalog→db lifted to api→db) map to
	// the same pair, the direct one's label is used for the connector.
	for pass := 0; pass < 2; pass++ {
		for _, ch := range cs.ModelRelationshipChanges {
			switch ch.Type {
			case Deleted:
				if pass != 0 {
					continue
				}
				// For deletions, use scoped cell IDs to find and remove the
				// connector. The original endpoints may no longer be in the
				// view's element filter, so we bypass lifting entirely.
				fromRef := scopedCellID(viewID, ch.From)
				toRef := scopedCellID(viewID, ch.To)
				if page.FindConnector(fromRef, toRef) != nil {
					page.DeleteConnector(fromRef, toRef)
					result.ConnectorsDeleted++
				}
			default:
				from := ch.From
				to := ch.To
				if elemFilter != nil {
					from = liftEndpoint(from, elemFilter)
					to = liftEndpoint(to, elemFilter)
					if from == "" || to == "" || from == to {
						continue
					}
				}
				isLifted := from != ch.From || to != ch.To
				if pass == 0 && isLifted {
					continue // First pass: only direct relationships
				}
				if pass == 1 && !isLifted {
					continue // Second pass: only lifted relationships
				}
				lifted := RelationshipChange{From: from, To: to, Type: ch.Type, NewValue: ch.NewValue}
				pairKey := from + "->" + to
				switch ch.Type {
				case Added:
					if liftedSeen[pairKey] {
						continue
					}
					liftedSeen[pairKey] = true
					applyRelAdded(lifted, viewID, page, templates, result)
				case Modified:
					page.UpdateConnectorLabel(from, to, ch.NewValue)
					result.ConnectorsUpdated++
				}
			}
		}
	}
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
// If scopeID is set and the element is a child of the scope, it is parented to the
// scope boundary cell.
func applyElementAdded(
	id string,
	viewID string,
	scopeID string,
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
		CellID:      scopedCellID(viewID, id),
		Kind:        elem.Kind,
		Title:       elem.Title,
		Technology:  elem.Technology,
		Description: elem.Description,
		X:           pl.nextX,
		Y:           pl.nextY,
		Width:       width,
		Height:      height,
	}

	// Parent children of the scope element to the boundary cell.
	if scopeID != "" && isChildOf(id, scopeID) {
		data.ParentID = scopedCellID(viewID, scopeID)
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
	templates *drawio.TemplateSet,
	flat map[string]*model.Element,
	result *ForwardResult,
) {
	elem, ok := flat[ch.ID]
	if !ok {
		result.Warnings = append(result.Warnings, "element not found in model for update: "+ch.ID)
		return
	}

	// Handle kind changes separately (only updates attribute and style).
	if ch.Field == "kind" {
		ts, ok := templates.GetStyle(elem.Kind)
		if ok {
			page.UpdateElementKind(ch.ID, elem.Kind, ts.Style)
		} else {
			page.UpdateElementKind(ch.ID, elem.Kind, "")
		}
		result.ElementsUpdated++
		return
	}

	// When a specific field is known, read the current draw.io values and only
	// override the changed field. This prevents overwriting draw.io-side changes
	// to other fields during concurrent modification. (#109)
	title := elem.Title
	technology := elem.Technology
	description := elem.Description

	if ch.Field != "" {
		obj := page.FindElement(ch.ID)
		if obj != nil {
			label := obj.SelectAttrValue("label", "")
			curTitle, curTech, curDesc := drawio.ParseLabel(label)
			curTooltip := obj.SelectAttrValue("tooltip", "")
			if curDesc == "" {
				curDesc = curTooltip
			}

			// Start from current draw.io values, override only the changed field.
			title = curTitle
			technology = curTech
			description = curDesc

			switch ch.Field {
			case "title":
				title = elem.Title
			case "technology":
				technology = elem.Technology
			case "description":
				description = elem.Description
			}
		}
	}

	data := drawio.ElementData{
		ID:          ch.ID,
		Title:       title,
		Technology:  technology,
		Description: description,
	}
	page.UpdateElement(ch.ID, data)
	result.ElementsUpdated++
}

// countConnectorsFor counts the number of connectors on page where source or
// target matches cellID. This is used to increment ConnectorsDeleted before
// calling page.DeleteConnectorsFor.
func countConnectorsFor(page *drawio.Page, cellID string) int {
	n := 0
	for _, c := range page.FindAllConnectors() {
		src := c.SelectAttrValue("source", "")
		tgt := c.SelectAttrValue("target", "")
		if src == cellID || tgt == cellID {
			n++
		}
	}
	return n
}

// applyRelAdded creates a new connector on page.
func applyRelAdded(
	ch RelationshipChange,
	viewID string,
	page *drawio.Page,
	templates *drawio.TemplateSet,
	result *ForwardResult,
) {
	style := templates.GetConnectorStyle()
	data := drawio.ConnectorData{
		From:      ch.From,
		To:        ch.To,
		Label:     ch.NewValue,
		SourceRef: scopedCellID(viewID, ch.From),
		TargetRef: scopedCellID(viewID, ch.To),
	}
	page.CreateConnector(data, style)
	result.ConnectorsCreated++
}

// reconcileViewPage removes elements from the page that are not in the
// resolved view filter. This handles cases where view include/exclude rules
// change without corresponding model element changes (no ChangeSet entries).
// Elements not present in the flat model map are preserved — they were
// manually added by the user in draw.io and should not be deleted. (#115)
func reconcileViewPage(
	page *drawio.Page,
	elemFilter map[string]bool,
	flat map[string]*model.Element,
	scopeID string,
	viewID string,
	result *ForwardResult,
) {
	if elemFilter == nil {
		return
	}

	for _, obj := range page.FindAllElements() {
		id := obj.SelectAttrValue("bausteinsicht_id", "")
		if id == "" {
			continue
		}

		// Skip the scope boundary element — it's rendered separately
		// and is not subject to the normal element filter.
		if id == scopeID {
			continue
		}

		if elemFilter[id] {
			continue
		}

		// Preserve elements not in the model — they were manually added
		// by the user in draw.io and should not be deleted. (#115)
		if _, inModel := flat[id]; !inModel {
			continue
		}

		// Element is on the page but not in the view's resolved set.
		// Remove its connectors first (using scoped cell ID), then the element.
		// On view pages, connectors reference scoped cell IDs, not raw element IDs.
		cellID := scopedCellID(viewID, id)
		result.ConnectorsDeleted += countConnectorsFor(page, cellID)
		page.DeleteConnectorsFor(cellID)

		page.DeleteElement(id)
		result.ElementsDeleted++
	}
}

// reconcileOrphanedElements removes elements from the page whose
// bausteinsicht_id does not exist in the current model. This is the
// no-views equivalent of reconcileViewPage and handles cases where
// sync state is missing or the model was emptied. (#110)
func reconcileOrphanedElements(
	page *drawio.Page,
	flat map[string]*model.Element,
	result *ForwardResult,
) {
	for _, obj := range page.FindAllElements() {
		id := obj.SelectAttrValue("bausteinsicht_id", "")
		if id == "" {
			continue
		}
		if _, inModel := flat[id]; inModel {
			continue
		}
		// Element is on the page but not in the model — remove it.
		cellID := id // no view scoping in legacy mode
		result.ConnectorsDeleted += countConnectorsFor(page, cellID)
		page.DeleteConnectorsFor(cellID)
		page.DeleteElement(id)
		result.ElementsDeleted++
	}
}
