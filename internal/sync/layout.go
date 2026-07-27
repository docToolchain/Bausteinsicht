package sync

import (
	"math"
	"sort"
	"strings"

	"github.com/docToolchain/Bausteinsicht/internal/drawio"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

type position struct {
	X, Y float64
}

type layoutConfig struct {
	pageWidth  float64 // 1169 (A4 landscape)
	elementGap float64 // 40
	padding    float64 // 60 (boundary inner padding)
	startX     float64 // 40
	startY     float64 // 40
}

var defaultLayoutConfig = layoutConfig{
	pageWidth:  1169,
	elementGap: 60,
	padding:    60,
	startX:     40,
	startY:     40,
}

type sizePair struct{ W, H float64 }

// layoutResult holds computed positions and optional boundary dimensions.
type layoutResult struct {
	Positions      map[string]position
	BoundaryX      float64
	BoundaryY      float64
	BoundaryWidth  float64
	BoundaryHeight float64
	ContainerSizes map[string]sizePair // "nested" layout: per-container computed size
	ParentOf       map[string]string   // "nested" layout: element ID → immediate container ID
}

// computeLayout returns positions for elements to place on a fresh page.
// It only computes positions; it does not modify the document.
func computeLayout(
	ids []string,
	flat map[string]*model.Element,
	templates *drawio.TemplateSet,
	elementOrder []string,
	scopeID string,
	layout string,
	relationships []model.Relationship,
) layoutResult {
	switch layout {
	case "grid":
		return computeGridLayout(ids, flat, templates)
	case "none":
		return computeNoneLayout(ids, flat, templates)
	case "nested":
		return computeNestedLayout(ids, flat, templates, scopeID)
	default: // "layered" or ""
		return computeLayeredLayout(ids, flat, templates, elementOrder, scopeID, relationships)
	}
}

// computeNestedLayout lays out a scope's subtree as true multi-level containment:
// each container element gets its immediate children placed inside it (relative
// coordinates), and every container is sized to fit its children. This is what
// makes draw.io render concentric hexagonal rings when zone kinds carry a
// hexagon-container style. Positions are relative to each element's IMMEDIATE
// parent; forward sync re-parents cells accordingly (see populateNewPage).
func computeNestedLayout(
	ids []string,
	flat map[string]*model.Element,
	templates *drawio.TemplateSet,
	scopeID string,
) layoutResult {
	res := layoutResult{
		Positions:      make(map[string]position),
		ContainerSizes: make(map[string]sizePair),
		ParentOf:       make(map[string]string),
	}
	if scopeID == "" {
		// Nested mode targets scoped views; without a scope, fall back to grid.
		return computeGridLayout(ids, flat, templates)
	}

	idset := make(map[string]bool, len(ids)+1)
	for _, id := range ids {
		idset[id] = true
	}
	idset[scopeID] = true

	immediateChildren := func(id string) []string {
		prefix := id + "."
		var cs []string
		for cid := range idset {
			if strings.HasPrefix(cid, prefix) && !strings.Contains(cid[len(prefix):], ".") {
				cs = append(cs, cid)
			}
		}
		sort.Strings(cs)
		return cs
	}

	const (
		pad    = 26.0 // inner padding of a container
		titleH = 46.0 // room for the container's title band (clears crossing edge labels)
		gap    = 40.0 // gap between siblings (also gives edges room for their labels)
	)

	// layoutOf arranges a container's children as concentric vertical bands:
	// a TOP row (leaves with metadata.side != "bottom"), the inner ring(s) in
	// the MIDDLE, and a BOTTOM row (leaves with metadata.side == "bottom"),
	// each horizontally centered. This gives the classic hexagonal look with
	// driving/inbound on top and driven/outbound at the bottom. A hexagon
	// margin widens the container so a hexagon zone does not clip content at
	// its slanted sides. Recursion yields true concentric rings.
	rowDims := func(ids []string, size map[string]sizePair) (w, h float64) {
		for i, k := range ids {
			w += size[k].W
			if i > 0 {
				w += gap
			}
			if size[k].H > h {
				h = size[k].H
			}
		}
		return
	}

	var layoutOf func(id string) sizePair
	layoutOf = func(id string) sizePair {
		kids := immediateChildren(id)
		if len(kids) == 0 {
			w, h := elementSize(id, flat, templates)
			return sizePair{W: w, H: h}
		}

		var subs, top, bottom []string
		size := make(map[string]sizePair, len(kids))
		for _, k := range kids {
			if len(immediateChildren(k)) > 0 {
				subs = append(subs, k)
				size[k] = layoutOf(k) // size inner rings first
				continue
			}
			w, h := elementSize(k, flat, templates)
			size[k] = sizePair{W: w, H: h}
			if el, ok := flat[k]; ok && el.Metadata["side"] == "bottom" {
				bottom = append(bottom, k)
			} else {
				top = append(top, k)
			}
		}

		topW, topH := rowDims(top, size)
		botW, botH := rowDims(bottom, size)
		var innerW, innerH float64
		for i, k := range subs {
			if size[k].W > innerW {
				innerW = size[k].W
			}
			innerH += size[k].H
			if i > 0 {
				innerH += gap
			}
		}

		localPad := pad
		if len(subs) == 0 { // innermost ring (e.g. domain core): give it more room
			localPad = pad * 2.5
		}

		contentW := math.Max(topW, math.Max(innerW, botW))
		if contentW == 0 {
			contentW = defaultWidth
		}
		if len(subs) == 0 && contentW < 320 { // make the domain core visually substantial
			contentW = 320
		}
		bandH, bands := 0.0, 0
		for _, h := range []float64{topH, innerH, botH} {
			if h > 0 {
				bandH += h
				bands++
			}
		}
		contentH := bandH
		if bands > 1 {
			contentH += float64(bands-1) * gap
		}

		hexMargin := contentH * 0.30
		x0 := localPad + hexMargin
		y := titleH + localPad

		placeRow := func(ids []string) {
			if len(ids) == 0 {
				return
			}
			rowW, rowH := rowDims(ids, size)
			cx := x0 + (contentW-rowW)/2
			for _, k := range ids {
				sz := size[k]
				res.Positions[k] = position{X: cx, Y: y + (rowH-sz.H)/2}
				res.ParentOf[k] = id
				cx += sz.W + gap
			}
			y += rowH + gap
		}
		placeRow(top)
		if len(subs) > 0 && len(top) > 0 {
			y += gap // extra clearance so an edge crossing into the inner ring clears its title band
		}
		for _, k := range subs { // inner ring(s), horizontally centered
			sz := size[k]
			res.Positions[k] = position{X: x0 + (contentW-sz.W)/2, Y: y}
			res.ParentOf[k] = id
			y += sz.H + gap
		}
		if len(subs) > 0 && len(bottom) > 0 {
			y += gap
		}
		placeRow(bottom)

		result := sizePair{W: contentW + 2*(localPad+hexMargin), H: y - gap + localPad}
		res.ContainerSizes[id] = result
		return result
	}

	top := layoutOf(scopeID)
	res.BoundaryX = defaultLayoutConfig.startX
	res.BoundaryWidth = top.W
	res.BoundaryHeight = top.H
	delete(res.ContainerSizes, scopeID) // the scope is sized via Boundary* above

	// Elements outside the scope (externals) are placed around the ring hexagon:
	// actor-like ones above, the rest below, each centered to the boundary width.
	var actors, others []string
	for _, id := range ids {
		if id == scopeID || isChildOf(id, scopeID) {
			continue
		}
		if isActorKind(id, flat) {
			actors = append(actors, id)
		} else {
			others = append(others, id)
		}
	}
	const extGap = 60.0
	placeExternRow := func(rowIDs []string, y float64) float64 {
		if len(rowIDs) == 0 {
			return 0
		}
		sort.Strings(rowIDs)
		widths := make([]float64, len(rowIDs))
		heights := make([]float64, len(rowIDs))
		totalW, maxH := 0.0, 0.0
		for i, id := range rowIDs {
			w, h := elementSize(id, flat, templates)
			widths[i] = w
			heights[i] = h
			totalW += w
			if i > 0 {
				totalW += extGap
			}
			if h > maxH {
				maxH = h
			}
		}
		cx := res.BoundaryX + (res.BoundaryWidth-totalW)/2
		for i, id := range rowIDs {
			res.Positions[id] = position{X: cx, Y: y + (maxH-heights[i])/2}
			cx += widths[i] + extGap
		}
		return maxH
	}

	y := defaultLayoutConfig.startY
	if h := placeExternRow(actors, y); h > 0 {
		y += h + extGap
	}
	res.BoundaryY = y
	placeExternRow(others, y+res.BoundaryHeight+extGap)
	return res
}

// computeLayeredLayout arranges elements in horizontal rows grouped by kind.
// Kinds are ordered according to elementOrder (from specification.elements).
// Elements within each layer are sorted alphabetically.
//
// For scoped views the layout is:
//  1. Actor-like externals (kinds whose notation contains "Actor") → top rows
//  2. Scope boundary with children → middle
//  3. Non-actor externals → bottom rows
func computeLayeredLayout(
	ids []string,
	flat map[string]*model.Element,
	templates *drawio.TemplateSet,
	elementOrder []string,
	scopeID string,
	relationships []model.Relationship,
) layoutResult {
	cfg := defaultLayoutConfig
	result := layoutResult{Positions: make(map[string]position)}

	// Build kind → tier mapping from elementOrder.
	kindTier := make(map[string]int)
	for i, k := range elementOrder {
		kindTier[k] = i
	}
	maxTier := len(elementOrder) // unknown kinds go here

	// Separate scoped children from external elements.
	// For externals, further split into actors (top) and non-actors (bottom).
	var scopeChildren []string
	var actorExternals []string
	var otherExternals []string
	for _, id := range ids {
		if id == scopeID {
			continue
		}
		if scopeID != "" && isChildOf(id, scopeID) {
			scopeChildren = append(scopeChildren, id)
		} else {
			if isActorKind(id, flat) {
				actorExternals = append(actorExternals, id)
			} else {
				otherExternals = append(otherExternals, id)
			}
		}
	}

	// If no scope, treat everything as one group (actors first, then rest).
	if scopeID == "" {
		all := append(actorExternals, otherExternals...)
		all = append(all, scopeChildren...)
		placeLayered(all, flat, templates, kindTier, maxTier, cfg, cfg.startX, cfg.startY, result.Positions)
		return result
	}

	// Compute boundary dimensions first (needed for centering actors/externals).
	boundaryX := cfg.startX
	var boundaryContentW, boundaryContentH float64
	boundaryStartSize := 30.0

	scopeChildPositions := make(map[string]position)
	if len(scopeChildren) > 0 {
		innerX := cfg.padding
		innerY := boundaryStartSize + cfg.padding
		minContentWidth := minBoundaryContentWidth(scopeChildren, flat, templates, cfg)

		boundaryContentW, boundaryContentH = placeBFS(scopeChildren, flat, templates, cfg, innerX, innerY, minContentWidth, actorExternals, relationships, scopeChildPositions)

		if boundaryContentW < minContentWidth {
			boundaryContentW = minContentWidth
		}

		result.BoundaryWidth = boundaryContentW + 2*cfg.padding
		result.BoundaryHeight = boundaryContentH + boundaryStartSize + 2*cfg.padding

		if result.BoundaryWidth < 400 {
			result.BoundaryWidth = 400
		}
		if result.BoundaryHeight < 300 {
			result.BoundaryHeight = 300
		}
	}

	// Reference width for centering: the wider of boundary or page content.
	refWidth := result.BoundaryWidth
	if refWidth < 400 {
		refWidth = 400
	}

	curY := cfg.startY

	// 1. Place actor externals above the boundary, centered to boundary width.
	// Reserve the first row even when there are no actors so users can add them later.
	if len(actorExternals) > 0 {
		actorW, actorH := placeLayered(actorExternals, flat, templates, kindTier, maxTier, cfg, cfg.startX, curY, result.Positions)
		// Center actor row relative to boundary width.
		if actorW < refWidth {
			offset := (refWidth - actorW) / 2
			for _, id := range actorExternals {
				if p, ok := result.Positions[id]; ok {
					p.X += offset
					result.Positions[id] = p
				}
			}
		}
		curY += actorH + cfg.elementGap
	} else {
		// No actors — still reserve space for a potential actor row.
		curY += defaultHeight + cfg.elementGap
	}

	// 2. Place scope boundary and its children.
	boundaryY := curY
	if len(scopeChildren) > 0 {
		result.BoundaryX = boundaryX
		result.BoundaryY = boundaryY

		// Copy pre-computed child positions into result.
		for id, pos := range scopeChildPositions {
			result.Positions[id] = pos
		}

		curY = boundaryY + result.BoundaryHeight + cfg.elementGap
	}

	// 3. Place non-actor externals below the boundary, centered.
	if len(otherExternals) > 0 {
		extW, _ := placeLayered(otherExternals, flat, templates, kindTier, maxTier, cfg, cfg.startX, curY, result.Positions)
		if extW < refWidth {
			offset := (refWidth - extW) / 2
			for _, id := range otherExternals {
				if p, ok := result.Positions[id]; ok {
					p.X += offset
					result.Positions[id] = p
				}
			}
		}
	}

	return result
}

// isActorKind returns true if the element's kind contains "actor" (case-insensitive).
func isActorKind(id string, flat map[string]*model.Element) bool {
	elem := flat[id]
	if elem == nil {
		return false
	}
	k := elem.Kind
	return k == "actor" || k == "Actor" || k == "user" || k == "User" ||
		k == "person" || k == "Person"
}

// minBoundaryContentWidth computes the minimum content width to fit at least
// 3 average-sized elements side by side.
func minBoundaryContentWidth(ids []string, flat map[string]*model.Element, templates *drawio.TemplateSet, cfg layoutConfig) float64 {
	if len(ids) == 0 {
		return 0
	}
	// Use the most common element width as reference.
	totalW := 0.0
	for _, id := range ids {
		w, _ := elementSize(id, flat, templates)
		totalW += w
	}
	avgW := totalW / float64(len(ids))

	cols := 3
	if len(ids) < cols {
		cols = len(ids)
	}
	return float64(cols)*avgW + float64(cols-1)*cfg.elementGap
}

// placeLayered places elements in layered rows, centered horizontally,
// and returns the content width and height.
func placeLayered(
	ids []string,
	flat map[string]*model.Element,
	templates *drawio.TemplateSet,
	kindTier map[string]int,
	maxTier int,
	cfg layoutConfig,
	originX, originY float64,
	positions map[string]position,
) (contentWidth, contentHeight float64) {
	// Group by tier.
	tiers := make(map[int][]string)
	for _, id := range ids {
		elem := flat[id]
		if elem == nil {
			continue
		}
		tier, ok := kindTier[elem.Kind]
		if !ok {
			tier = maxTier
		}
		tiers[tier] = append(tiers[tier], id)
	}

	// Sort tier keys.
	tierKeys := make([]int, 0, len(tiers))
	for k := range tiers {
		tierKeys = append(tierKeys, k)
	}
	sort.Ints(tierKeys)

	// First pass: place left-aligned and track row membership for centering.
	type rowInfo struct {
		ids    []string
		width  float64
		height float64
		y      float64
	}
	var rows []rowInfo
	curY := originY
	maxRowWidth := 0.0

	for _, tier := range tierKeys {
		elems := tiers[tier]
		sort.Strings(elems)

		curX := originX
		rowHeight := 0.0
		var currentRow []string

		for _, id := range elems {
			w, h := elementSize(id, flat, templates)

			// Row wrapping.
			if curX > originX && curX+w > cfg.pageWidth-cfg.startX {
				rowWidth := curX - cfg.elementGap - originX
				rows = append(rows, rowInfo{ids: currentRow, width: rowWidth, height: rowHeight, y: curY})
				if rowWidth > maxRowWidth {
					maxRowWidth = rowWidth
				}
				curY += rowHeight + cfg.elementGap
				curX = originX
				rowHeight = 0
				currentRow = nil
			}

			positions[id] = position{X: curX, Y: curY}
			currentRow = append(currentRow, id)
			curX += w + cfg.elementGap
			if h > rowHeight {
				rowHeight = h
			}
		}

		// Finish last row of this tier.
		if len(currentRow) > 0 {
			rowWidth := curX - cfg.elementGap - originX
			rows = append(rows, rowInfo{ids: currentRow, width: rowWidth, height: rowHeight, y: curY})
			if rowWidth > maxRowWidth {
				maxRowWidth = rowWidth
			}
		}
		curY += rowHeight + cfg.elementGap
	}

	// Second pass: center each row within the max row width.
	for _, row := range rows {
		if row.width >= maxRowWidth {
			continue
		}
		offset := (maxRowWidth - row.width) / 2
		for _, id := range row.ids {
			p := positions[id]
			p.X += offset
			positions[id] = p
		}
	}

	contentWidth = maxRowWidth
	contentHeight = curY - originY - cfg.elementGap // subtract trailing gap
	if contentHeight < 0 {
		contentHeight = 0
	}
	return contentWidth, contentHeight
}

// placeBFS places scope children using relationship-based BFS ordering:
//  1. Row 1: elements connected to actors (external seeds)
//  2. Row 2: elements connected to row 1
//  3. Row N: elements connected to row N-1
//  4. Remaining: any unconnected elements at the end
//
// Within each row, the next element is chosen by adjacency to the last placed
// element (greedy neighbor selection), with alphabetical fallback.
func placeBFS(
	ids []string,
	flat map[string]*model.Element,
	templates *drawio.TemplateSet,
	cfg layoutConfig,
	originX, originY float64,
	minRowWidth float64,
	actorIDs []string,
	relationships []model.Relationship,
	positions map[string]position,
) (contentWidth, contentHeight float64) {
	// Build an adjacency map among the scope children.
	idSet := make(map[string]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}

	// adj maps each scope child to its neighbors (other scope children
	// or external elements it is connected to).
	adj := make(map[string]map[string]bool)
	for _, id := range ids {
		adj[id] = make(map[string]bool)
	}

	actorSet := make(map[string]bool, len(actorIDs))
	for _, id := range actorIDs {
		actorSet[id] = true
	}

	// connectedToActor tracks which scope children have a direct or
	// indirect (via lifted) relationship to an actor.
	connectedToActor := make(map[string]bool)

	for _, rel := range relationships {
		from, to := rel.From, rel.To

		// Check if this relationship connects a scope child to an actor.
		if idSet[from] && actorSet[to] {
			connectedToActor[from] = true
		}
		if idSet[to] && actorSet[from] {
			connectedToActor[to] = true
		}

		// Also check lifted relationships: if an actor connects to a
		// parent of a scope child (e.g., actor→onlineshop.frontend where
		// frontend is a scope child via onlineshop.frontend).
		fromInScope := idSet[from] || hasChildInSet(from, idSet)
		toInScope := idSet[to] || hasChildInSet(to, idSet)
		_ = fromInScope
		_ = toInScope

		// Build adjacency between scope children.
		fromResolved := resolveToScopeChild(from, idSet)
		toResolved := resolveToScopeChild(to, idSet)
		if fromResolved != "" && toResolved != "" && fromResolved != toResolved {
			if adj[fromResolved] != nil && adj[toResolved] != nil {
				adj[fromResolved][toResolved] = true
				adj[toResolved][fromResolved] = true
			}
		}

		// Track actor connections via resolved scope children.
		if fromResolved != "" && actorSet[to] {
			connectedToActor[fromResolved] = true
		}
		if toResolved != "" && actorSet[from] {
			connectedToActor[toResolved] = true
		}
	}

	// BFS: assign elements to rows.
	placed := make(map[string]bool)
	var rows [][]string

	// Row 0: elements connected to actors.
	var row0 []string
	for _, id := range ids {
		if connectedToActor[id] {
			row0 = append(row0, id)
			placed[id] = true
		}
	}
	sort.Strings(row0)
	if len(row0) > 0 {
		rows = append(rows, orderByAdjacency(row0, adj))
	}

	// Subsequent rows: BFS from previous row.
	for len(rows) > 0 {
		if len(placed) >= len(ids) {
			break
		}
		prevRow := rows[len(rows)-1]
		var nextRow []string
		for _, prev := range prevRow {
			for neighbor := range adj[prev] {
				if !placed[neighbor] {
					nextRow = append(nextRow, neighbor)
					placed[neighbor] = true
				}
			}
		}
		if len(nextRow) == 0 {
			break
		}
		sort.Strings(nextRow)
		rows = append(rows, orderByAdjacency(nextRow, adj))
	}

	// Remaining: elements not reached by BFS (no relationships).
	var remaining []string
	for _, id := range ids {
		if !placed[id] {
			remaining = append(remaining, id)
		}
	}
	if len(remaining) > 0 {
		sort.Strings(remaining)
		rows = append(rows, remaining)
	}

	// Place rows.
	curY := originY
	maxWidth := 0.0

	for _, row := range rows {
		curX := originX
		rowHeight := 0.0

		for _, id := range row {
			w, h := elementSize(id, flat, templates)

			// Row wrapping.
			if curX > originX && curX+w > originX+minRowWidth {
				curY += rowHeight + cfg.elementGap
				curX = originX
				rowHeight = 0
			}

			positions[id] = position{X: curX, Y: curY}
			curX += w + cfg.elementGap
			usedWidth := curX - cfg.elementGap - originX
			if usedWidth > maxWidth {
				maxWidth = usedWidth
			}
			if h > rowHeight {
				rowHeight = h
			}
		}

		curY += rowHeight + cfg.elementGap
	}

	contentWidth = maxWidth
	contentHeight = curY - originY - cfg.elementGap
	if contentHeight < 0 {
		contentHeight = 0
	}
	return contentWidth, contentHeight
}

// orderByAdjacency reorders elements so that each next element is a neighbor
// of the previously placed one (greedy). Falls back to original order.
func orderByAdjacency(ids []string, adj map[string]map[string]bool) []string {
	if len(ids) <= 1 {
		return ids
	}

	remaining := make(map[string]bool, len(ids))
	for _, id := range ids {
		remaining[id] = true
	}

	result := make([]string, 0, len(ids))
	// Start with the first element (alphabetically).
	current := ids[0]
	result = append(result, current)
	delete(remaining, current)

	for len(remaining) > 0 {
		// Find a neighbor of current that is in remaining.
		var next string
		var candidates []string
		for neighbor := range adj[current] {
			if remaining[neighbor] {
				candidates = append(candidates, neighbor)
			}
		}
		if len(candidates) > 0 {
			sort.Strings(candidates)
			next = candidates[0]
		} else {
			// No neighbor found — pick alphabetically first remaining.
			var fallback []string
			for id := range remaining {
				fallback = append(fallback, id)
			}
			if len(fallback) == 0 {
				break
			}
			sort.Strings(fallback)
			next = fallback[0]
		}
		result = append(result, next)
		delete(remaining, next)
		current = next
	}

	return result
}

// resolveToScopeChild resolves an element ID to a scope child ID.
// If the ID is directly in the set, returns it. If a parent is in the set,
// returns the parent. Returns "" if no match.
func resolveToScopeChild(id string, scopeChildren map[string]bool) string {
	if scopeChildren[id] {
		return id
	}
	// Walk up the hierarchy to find a scope child ancestor.
	for {
		dot := strings.LastIndex(id, ".")
		if dot < 0 {
			return ""
		}
		id = id[:dot]
		if scopeChildren[id] {
			return id
		}
	}
}

// hasChildInSet returns true if any key in the set is a child of id.
func hasChildInSet(id string, set map[string]bool) bool {
	prefix := id + "."
	for k := range set {
		if strings.HasPrefix(k, prefix) {
			return true
		}
	}
	return false
}

// computeGridLayout arranges all elements in a simple grid.
func computeGridLayout(
	ids []string,
	flat map[string]*model.Element,
	templates *drawio.TemplateSet,
) layoutResult {
	cfg := defaultLayoutConfig
	result := layoutResult{Positions: make(map[string]position)}

	sorted := make([]string, len(ids))
	copy(sorted, ids)
	sort.Strings(sorted)

	// Determine max element width for column calculation.
	maxW := 0.0
	for _, id := range sorted {
		w, _ := elementSize(id, flat, templates)
		if w > maxW {
			maxW = w
		}
	}

	columns := int(math.Floor((cfg.pageWidth - 2*cfg.startX) / (maxW + cfg.elementGap)))
	if columns < 1 {
		columns = 1
	}

	col, row := 0, 0
	for _, id := range sorted {
		_, h := elementSize(id, flat, templates)
		_ = h
		x := cfg.startX + float64(col)*(maxW+cfg.elementGap)
		y := cfg.startY + float64(row)*(defaultHeight+cfg.elementGap)
		result.Positions[id] = position{X: x, Y: y}
		col++
		if col >= columns {
			col = 0
			row++
		}
	}

	return result
}

// computeNoneLayout uses the legacy horizontal row placement.
func computeNoneLayout(
	ids []string,
	flat map[string]*model.Element,
	templates *drawio.TemplateSet,
) layoutResult {
	cfg := defaultLayoutConfig
	result := layoutResult{Positions: make(map[string]position)}

	sorted := make([]string, len(ids))
	copy(sorted, ids)
	sort.Strings(sorted)

	curX := cfg.startX
	for _, id := range sorted {
		w, _ := elementSize(id, flat, templates)
		result.Positions[id] = position{X: curX, Y: cfg.startY}
		curX += w + cfg.elementGap
	}

	return result
}

// elementSize returns the width and height for an element, based on its
// template style or default dimensions.
func elementSize(id string, flat map[string]*model.Element, templates *drawio.TemplateSet) (float64, float64) {
	elem := flat[id]
	if elem == nil {
		return defaultWidth, defaultHeight
	}
	ts, ok := templates.GetStyle(elem.Kind)
	if !ok {
		return defaultWidth, defaultHeight
	}
	w := ts.Width
	if w == 0 {
		w = defaultWidth
	}
	h := ts.Height
	if h == 0 {
		h = defaultHeight
	}
	return w, h
}
