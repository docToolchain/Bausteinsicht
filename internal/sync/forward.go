package sync

import (
	"strconv"
	"strings"

	"github.com/beevik/etree"
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
	// Build drill-down link map: elementID → "data:page/id,view-<viewID>"
	// An element gets a link when a view's scope matches that element.
	drillDownLinks := make(map[string]string)
	for vID, v := range m.Views {
		if v.Scope != "" {
			drillDownLinks[v.Scope] = "data:page/id,view-" + vID
		}
	}

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
			// Include scope element in the filter so connectors targeting the
			// boundary element are rendered (#217).
			elemSet[scopeID] = true
		}

		applyChangesToPage(cs, page, templates, flat, elemSet, viewID, scopeID, result)

		// Populate resolved elements that aren't already on the page.
		// For new pages this handles elements already in sync state (#184,
		// #188, #189). For existing pages this handles elements newly
		// included via view include/exclude changes that don't appear in
		// the ChangeSet (#231).
		populateNewPage(page, viewID, scopeID, templates, flat, elemSet, result)

		// Populate connectors for relationships whose endpoints are both
		// on the page but whose connector doesn't exist yet. This handles
		// relationships involving newly populated elements (#231).
		populateConnectors(page, viewID, m, elemSet, templates, result)

		// Reconciliation: remove elements on the page that are no longer
		// in the resolved view (e.g., after exclude list changes). #102
		reconcileViewPage(page, elemSet, flat, scopeID, viewID, result)

		// Set drill-down links on elements that have a detail view. (#198)
		applyDrillDownLinks(page, drillDownLinks)

		// Create back-navigation button on detail views (views with scope). (#198)
		if scopeID != "" {
			createBackNavButton(page, viewID, scopeID, m)
		}
	}
}

// populateNewPage creates elements on a view page for all elements in
// the view's resolved set that aren't already present.
// For new pages, this populates elements already in sync state (#184, #188).
// For existing pages, this adds elements newly included via view changes (#231).
func populateNewPage(
	page *drawio.Page,
	viewID string,
	scopeID string,
	templates *drawio.TemplateSet,
	flat map[string]*model.Element,
	elemSet map[string]bool,
	result *ForwardResult,
) {
	pl := computePlacement(page)
	for id := range elemSet {
		if id == scopeID {
			continue // Scope boundary handled separately.
		}
		if page.FindElement(id) != nil {
			continue // Already created by applyChangesToPage.
		}
		applyElementAdded(id, viewID, scopeID, page, templates, flat, &pl, result)
	}
}

// populateConnectors creates connectors for all model relationships whose
// (possibly lifted) endpoints are both on the page but no connector exists yet.
// This ensures relationships involving newly populated elements are rendered (#231).
func populateConnectors(
	page *drawio.Page,
	viewID string,
	m *model.BausteinsichtModel,
	elemSet map[string]bool,
	templates *drawio.TemplateSet,
	result *ForwardResult,
) {
	liftedSeen := make(map[string]bool)
	for i, rel := range m.Relationships {
		from := liftEndpoint(rel.From, elemSet)
		to := liftEndpoint(rel.To, elemSet)
		if from == "" || to == "" {
			continue
		}
		// Skip self-referencing lifted relationships.
		if from == to && (from != rel.From || to != rel.To) {
			continue
		}

		isLifted := from != rel.From || to != rel.To
		pairKey := from + "->" + to
		if isLifted {
			if liftedSeen[pairKey] {
				continue
			}
			liftedSeen[pairKey] = true
		} else {
			liftedSeen[pairKey] = true
		}

		srcRef := scopedCellID(viewID, from)
		tgtRef := scopedCellID(viewID, to)
		if page.FindConnector(srcRef, tgtRef, i) != nil {
			continue // Already exists.
		}
		style := templates.GetConnectorStyle()
		data := drawio.ConnectorData{
			From:      from,
			To:        to,
			Label:     rel.Label,
			SourceRef: srcRef,
			TargetRef: tgtRef,
			Index:     i,
		}
		page.CreateConnector(data, style)
		result.ConnectorsCreated++
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
		// Boundaries don't use sub-cells — they use the swimlane header for the label.
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
				if page.FindConnector(fromRef, toRef, ch.Index) != nil {
					page.DeleteConnector(fromRef, toRef, ch.Index)
					result.ConnectorsDeleted++
				}
			default:
				from := ch.From
				to := ch.To
				if elemFilter != nil {
					from = liftEndpoint(from, elemFilter)
					to = liftEndpoint(to, elemFilter)
					if from == "" || to == "" || (from == to && (from != ch.From || to != ch.To)) {
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
				lifted := RelationshipChange{From: from, To: to, Index: ch.Index, Type: ch.Type, NewValue: ch.NewValue}
				switch ch.Type {
				case Added:
					// Only deduplicate lifted relationships. When multiple
					// child relationships (e.g., a.x→b.z and a.y→b.z) are
					// lifted to the same parent pair (a→b), only one connector
					// should be created. Direct (non-lifted) relationships
					// with the same pair must not be deduplicated. (#142)
					pairKey := from + "->" + to
					if isLifted {
						// Skip lifted relationships when a direct relationship
						// or another lifted relationship already covers this
						// pair. (#142, #197)
						if liftedSeen[pairKey] {
							continue
						}
						liftedSeen[pairKey] = true
					} else {
						// Record direct relationships so lifted ones targeting
						// the same pair are suppressed in pass 1. (#197)
						liftedSeen[pairKey] = true
					}
					applyRelAdded(lifted, viewID, page, templates, result)
				case Modified:
					page.UpdateConnectorLabel(from, to, ch.Index, ch.NewValue)
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
	// Skip if element already exists on the page (prevents duplicates on sync state reset). (#141)
	if page.FindElement(id) != nil {
		return
	}

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

	style := mergeStyles(ts.Style, newElementMarker)

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
		SubCells:    subCellsFromTemplate(ts),
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

// subCellsFromTemplate creates SubCellTemplates from a TemplateStyle.
// Returns nil if the template has no sub-cell definitions.
func subCellsFromTemplate(ts drawio.TemplateStyle) *drawio.SubCellTemplates {
	if ts.TitleStyle == nil {
		return nil
	}
	return &drawio.SubCellTemplates{
		Title: ts.TitleStyle,
		Tech:  ts.TechStyle,
		Desc:  ts.DescStyle,
	}
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
			// Use ReadElementFields which handles both sub-cells and HTML labels.
			curTitle, curTech, curDesc := page.ReadElementFields(obj)
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

// applyRelAdded creates a new connector on page. If the connector already
// exists (e.g., sync state was deleted), it is skipped to avoid duplicates. (#119)
func applyRelAdded(
	ch RelationshipChange,
	viewID string,
	page *drawio.Page,
	templates *drawio.TemplateSet,
	result *ForwardResult,
) {
	srcRef := scopedCellID(viewID, ch.From)
	tgtRef := scopedCellID(viewID, ch.To)

	// Skip if connector already exists to prevent duplicates. (#119)
	if page.FindConnector(srcRef, tgtRef, ch.Index) != nil {
		return
	}

	style := templates.GetConnectorStyle()
	data := drawio.ConnectorData{
		From:      ch.From,
		To:        ch.To,
		Label:     ch.NewValue,
		SourceRef: srcRef,
		TargetRef: tgtRef,
		Index:     ch.Index,
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

// mergeStyles merges overlay style properties into a base style string.
// If both base and overlay define the same key (e.g., strokeColor), the
// overlay value wins. This prevents duplicate keys in the style string (#187).
func mergeStyles(base, overlay string) string {
	if overlay == "" {
		return base
	}
	if base == "" {
		return overlay
	}

	// Parse overlay keys.
	overlayKeys := make(map[string]string)
	for _, part := range strings.Split(overlay, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if idx := strings.IndexByte(part, '='); idx > 0 {
			overlayKeys[part[:idx]] = part
		} else {
			overlayKeys[part] = part
		}
	}

	// Build result: base properties (skipping those overridden) + overlay.
	var sb strings.Builder
	for _, part := range strings.Split(base, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key := part
		if idx := strings.IndexByte(part, '='); idx > 0 {
			key = part[:idx]
		}
		if _, overridden := overlayKeys[key]; overridden {
			continue
		}
		sb.WriteString(part)
		sb.WriteByte(';')
	}
	for _, v := range overlayKeys {
		sb.WriteString(v)
		sb.WriteByte(';')
	}
	return sb.String()
}

// applyDrillDownLinks sets the link attribute on elements that have a detail
// view (a view whose scope matches the element's bausteinsicht_id). (#198)
func applyDrillDownLinks(page *drawio.Page, links map[string]string) {
	for _, obj := range page.FindAllElements() {
		bid := obj.SelectAttrValue("bausteinsicht_id", "")
		if link, ok := links[bid]; ok {
			setAttrOn(obj, "link", link)
		}
	}
}

// setAttrOn sets or creates an attribute on an etree element.
func setAttrOn(el *etree.Element, key, value string) {
	for i, a := range el.Attr {
		if a.Key == key {
			el.Attr[i].Value = value
			return
		}
	}
	el.CreateAttr(key, value)
}

// createBackNavButton adds a small navigation button to a detail view page
// that links back to the parent view. The parent view is the one that
// contains the scope element. (#198)
func createBackNavButton(
	page *drawio.Page,
	viewID string,
	scopeID string,
	m *model.BausteinsichtModel,
) {
	navCellID := "nav-back-" + viewID

	// Don't create if already exists.
	root := page.Root()
	if root == nil {
		return
	}
	for _, obj := range root.SelectElements("object") {
		if obj.SelectAttrValue("id", "") == navCellID {
			return
		}
	}

	// Find the parent view: a view that includes the scope element.
	var parentViewID string
	var parentTitle string
	for vID, v := range m.Views {
		if vID == viewID {
			continue
		}
		viewCopy := v
		resolved, err := model.ResolveView(m, &viewCopy)
		if err != nil {
			continue
		}
		for _, id := range resolved {
			if id == scopeID {
				parentViewID = vID
				parentTitle = v.Title
				break
			}
		}
		if parentViewID != "" {
			break
		}
	}

	if parentViewID == "" {
		return // No parent view found.
	}

	obj := root.CreateElement("object")
	obj.CreateAttr("label", "&larr; "+parentTitle)
	obj.CreateAttr("id", navCellID)
	obj.CreateAttr("link", "data:page/id,view-"+parentViewID)

	cell := obj.CreateElement("mxCell")
	cell.CreateAttr("style", "rounded=1;fillColor=#f8cecc;strokeColor=#b85450;html=1;fontSize=10;")
	cell.CreateAttr("vertex", "1")
	cell.CreateAttr("parent", "1")

	geo := cell.CreateElement("mxGeometry")
	geo.CreateAttr("x", "20")
	geo.CreateAttr("y", "20")
	geo.CreateAttr("width", "140")
	geo.CreateAttr("height", "30")
	geo.CreateAttr("as", "geometry")
}
