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

// layoutResult holds computed positions and optional boundary dimensions.
type layoutResult struct {
	Positions      map[string]position
	BoundaryX      float64
	BoundaryY      float64
	BoundaryWidth  float64
	BoundaryHeight float64
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
	default: // "layered" or ""
		return computeLayeredLayout(ids, flat, templates, elementOrder, scopeID, relationships)
	}
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
	scopeChildren, actorExternals, otherExternals := groupElementsByScope(ids, scopeID, flat)

	// If no scope, treat everything as one group (actors first, then rest).
	if scopeID == "" {
		all := append(actorExternals, otherExternals...)
		all = append(all, scopeChildren...)
		placeLayered(all, flat, templates, kindTier, maxTier, cfg, cfg.startX, cfg.startY, result.Positions)
		return result
	}

	// Compute boundary dimensions first (needed for centering actors/externals).
	boundaryX := cfg.startX
	scopeChildPositions := computeScopeBoundary(&result, scopeChildren, flat, templates, cfg, actorExternals, relationships)

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
		centerRow(actorExternals, actorW, refWidth, result.Positions)
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
		centerRow(otherExternals, extW, refWidth, result.Positions)
	}

	return result
}

// groupElementsByScope splits ids into scoped children of scopeID, actor-kind
// externals, and other externals, preserving the input order within each group.
func groupElementsByScope(ids []string, scopeID string, flat map[string]*model.Element) (scopeChildren, actorExternals, otherExternals []string) {
	for _, id := range ids {
		if id == scopeID {
			continue
		}
		if scopeID != "" && isChildOf(id, scopeID) {
			scopeChildren = append(scopeChildren, id)
			continue
		}
		if isActorKind(id, flat) {
			actorExternals = append(actorExternals, id)
		} else {
			otherExternals = append(otherExternals, id)
		}
	}
	return scopeChildren, actorExternals, otherExternals
}

// computeScopeBoundary lays out scopeChildren inside the scope boundary and
// sets its computed width/height (clamped to a 400x300 minimum) on result.
// Returns the computed child positions, to be merged into result.Positions
// once the boundary's Y position is known.
func computeScopeBoundary(
	result *layoutResult,
	scopeChildren []string,
	flat map[string]*model.Element,
	templates *drawio.TemplateSet,
	cfg layoutConfig,
	actorExternals []string,
	relationships []model.Relationship,
) map[string]position {
	scopeChildPositions := make(map[string]position)
	if len(scopeChildren) == 0 {
		return scopeChildPositions
	}

	boundaryStartSize := 30.0
	innerX := cfg.padding
	innerY := boundaryStartSize + cfg.padding
	minContentWidth := minBoundaryContentWidth(scopeChildren, flat, templates, cfg)

	boundaryContentW, boundaryContentH := placeBFS(scopeChildren, flat, templates, cfg, innerX, innerY, minContentWidth, actorExternals, relationships, scopeChildPositions)
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
	return scopeChildPositions
}

// centerRow shifts the X position of each id in ids by the offset needed to
// center a row of width rowWidth within refWidth. No-op if rowWidth >= refWidth.
func centerRow(ids []string, rowWidth, refWidth float64, positions map[string]position) {
	if rowWidth >= refWidth {
		return
	}
	offset := (refWidth - rowWidth) / 2
	for _, id := range ids {
		if p, ok := positions[id]; ok {
			p.X += offset
			positions[id] = p
		}
	}
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
	tiers, tierKeys := groupIntoTiers(ids, flat, kindTier, maxTier)

	rows, maxRowWidth, endY := placeTierRows(tierKeys, tiers, flat, templates, cfg, originX, originY, positions)
	centerRows(rows, maxRowWidth, positions)

	contentWidth = maxRowWidth
	contentHeight = endY - originY - cfg.elementGap // subtract trailing gap
	if contentHeight < 0 {
		contentHeight = 0
	}
	return contentWidth, contentHeight
}

// layeredRow tracks one row's placed elements and geometry, computed by
// placeTierRows and consumed by centerRows.
type layeredRow struct {
	ids    []string
	width  float64
	height float64
	y      float64
}

// groupIntoTiers groups ids by their kind's tier (elements with unknown
// kinds fall into maxTier) and returns the tiers plus their sorted keys.
func groupIntoTiers(ids []string, flat map[string]*model.Element, kindTier map[string]int, maxTier int) (tiers map[int][]string, tierKeys []int) {
	tiers = make(map[int][]string)
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

	tierKeys = make([]int, 0, len(tiers))
	for k := range tiers {
		tierKeys = append(tierKeys, k)
	}
	sort.Ints(tierKeys)
	return tiers, tierKeys
}

// placeTierRows places each tier's elements left-aligned in row order,
// wrapping to a new row when the page width would be exceeded. Returns the
// resulting rows (for later centering), the widest row's width, and the Y
// position immediately after the last row.
func placeTierRows(
	tierKeys []int,
	tiers map[int][]string,
	flat map[string]*model.Element,
	templates *drawio.TemplateSet,
	cfg layoutConfig,
	originX, originY float64,
	positions map[string]position,
) (rows []layeredRow, maxRowWidth float64, endY float64) {
	curY := originY

	for _, tier := range tierKeys {
		elems := tiers[tier]
		sort.Strings(elems)

		var tierRows []layeredRow
		tierRows, curY = placeOneTier(elems, flat, templates, cfg, originX, curY, positions)
		for _, row := range tierRows {
			rows = append(rows, row)
			if row.width > maxRowWidth {
				maxRowWidth = row.width
			}
		}
	}

	return rows, maxRowWidth, curY
}

// placeOneTier places a single tier's elements left-aligned starting at
// (originX, startY), wrapping to a new row when the page width would be
// exceeded. Returns the resulting rows and the Y position after the last row.
func placeOneTier(
	elems []string,
	flat map[string]*model.Element,
	templates *drawio.TemplateSet,
	cfg layoutConfig,
	originX, startY float64,
	positions map[string]position,
) (rows []layeredRow, endY float64) {
	curY := startY
	curX := originX
	rowHeight := 0.0
	var currentRow []string

	for _, id := range elems {
		w, h := elementSize(id, flat, templates)

		// Row wrapping.
		if curX > originX && curX+w > cfg.pageWidth-cfg.startX {
			rowWidth := curX - cfg.elementGap - originX
			rows = append(rows, layeredRow{ids: currentRow, width: rowWidth, height: rowHeight, y: curY})
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
		rows = append(rows, layeredRow{ids: currentRow, width: rowWidth, height: rowHeight, y: curY})
	}
	curY += rowHeight + cfg.elementGap

	return rows, curY
}

// centerRows shifts each row's elements so the row is horizontally centered
// within maxRowWidth.
func centerRows(rows []layeredRow, maxRowWidth float64, positions map[string]position) {
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
	adj, connectedToActor := buildScopeAdjacency(ids, actorIDs, relationships)
	rows := assignBFSRows(ids, adj, connectedToActor)
	return placeBFSRows(rows, flat, templates, cfg, originX, originY, minRowWidth, positions)
}

// buildScopeAdjacency builds an adjacency map among ids (scope children)
// from relationships. adj maps each scope child to the other scope children
// it is (directly or, via lifting, indirectly) connected to.
// connectedToActor tracks which scope children have a direct or lifted
// relationship to one of actorIDs — e.g. actor→onlineshop.frontend lifts to
// the scope child onlineshop.frontend via resolveToScopeChild.
func buildScopeAdjacency(ids []string, actorIDs []string, relationships []model.Relationship) (adj map[string]map[string]bool, connectedToActor map[string]bool) {
	idSet := make(map[string]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}

	adj = make(map[string]map[string]bool)
	for _, id := range ids {
		adj[id] = make(map[string]bool)
	}

	actorSet := make(map[string]bool, len(actorIDs))
	for _, id := range actorIDs {
		actorSet[id] = true
	}

	connectedToActor = make(map[string]bool)
	for _, rel := range relationships {
		addRelationshipToAdjacency(rel, idSet, actorSet, adj, connectedToActor)
	}

	return adj, connectedToActor
}

// addRelationshipToAdjacency updates adj and connectedToActor with the
// contribution of a single relationship: direct and lifted (resolved via
// resolveToScopeChild) actor connections, and adjacency between scope
// children.
func addRelationshipToAdjacency(
	rel model.Relationship,
	idSet map[string]bool,
	actorSet map[string]bool,
	adj map[string]map[string]bool,
	connectedToActor map[string]bool,
) {
	from, to := rel.From, rel.To

	// Check if this relationship connects a scope child to an actor.
	if idSet[from] && actorSet[to] {
		connectedToActor[from] = true
	}
	if idSet[to] && actorSet[from] {
		connectedToActor[to] = true
	}

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

// assignBFSRows assigns ids to rows via relationship-based BFS:
//  1. Row 1: elements connected to actors (external seeds)
//  2. Row 2: elements connected to row 1
//  3. Row N: elements connected to row N-1
//  4. Remaining: any unconnected elements at the end
//
// Row *membership* only; within-row order is decided afterwards by
// minimizeCrossings, which needs every row assigned first so it can weigh
// each row against both of its neighbors.
func assignBFSRows(ids []string, adj map[string]map[string]bool, connectedToActor map[string]bool) [][]string {
	placed := make(map[string]bool)
	var rows [][]string

	if row0 := seedActorRow(ids, connectedToActor, placed); len(row0) > 0 {
		rows = append(rows, row0)
	}

	// Subsequent rows: BFS from previous row.
	for len(rows) > 0 && len(placed) < len(ids) {
		nextRow := nextBFSRow(rows[len(rows)-1], adj, placed)
		if len(nextRow) == 0 {
			break
		}
		rows = append(rows, nextRow)
	}

	// Remaining: elements not reached by BFS (no relationships).
	if remaining := unplacedSorted(ids, placed); len(remaining) > 0 {
		rows = append(rows, remaining)
	}

	return minimizeCrossings(rows, adj)
}

// seedActorRow returns, sorted, the ids directly or indirectly connected to
// an actor (row 0 of the BFS), marking each as placed.
func seedActorRow(ids []string, connectedToActor map[string]bool, placed map[string]bool) []string {
	var row0 []string
	for _, id := range ids {
		if connectedToActor[id] {
			row0 = append(row0, id)
			placed[id] = true
		}
	}
	sort.Strings(row0)
	return row0
}

// nextBFSRow returns, sorted, the not-yet-placed neighbors of prevRow's
// elements, marking each as placed.
func nextBFSRow(prevRow []string, adj map[string]map[string]bool, placed map[string]bool) []string {
	var nextRow []string
	for _, prev := range prevRow {
		for neighbor := range adj[prev] {
			if !placed[neighbor] {
				nextRow = append(nextRow, neighbor)
				placed[neighbor] = true
			}
		}
	}
	sort.Strings(nextRow)
	return nextRow
}

// unplacedSorted returns, sorted, the ids not yet marked placed.
func unplacedSorted(ids []string, placed map[string]bool) []string {
	var remaining []string
	for _, id := range ids {
		if !placed[id] {
			remaining = append(remaining, id)
		}
	}
	sort.Strings(remaining)
	return remaining
}

// placeBFSRows places each BFS row left-aligned starting at (originX,
// originY), wrapping to a new line when minRowWidth would be exceeded, and
// returns the overall content width/height.
func placeBFSRows(
	rows [][]string,
	flat map[string]*model.Element,
	templates *drawio.TemplateSet,
	cfg layoutConfig,
	originX, originY float64,
	minRowWidth float64,
	positions map[string]position,
) (contentWidth, contentHeight float64) {
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

// minimizeCrossings reorders elements *within* each row (row membership is
// unchanged — only the left-to-right position within a row moves) to reduce
// edge crossings between adjacent rows. Uses the barycenter method from
// Sugiyama-style layered graph drawing: repeatedly sweep top-down and
// bottom-up, reordering each row by the average position of its neighbors
// in the already-reordered adjacent row (see sortByBarycenter). Barycenter
// sweeps are not guaranteed to strictly improve every iteration, so this
// tracks total crossings (countCrossings) across all sweeps — including the
// untouched input — and returns whichever arrangement had the fewest.
//
// #607: replaces the previous greedy orderByAdjacency, which only chained
// each row's own elements to each other and never considered where a row's
// neighbors actually sit in the adjacent row — it could not reduce
// crossings between rows, only within one.
func minimizeCrossings(rows [][]string, adj map[string]map[string]bool) [][]string {
	if len(rows) <= 1 {
		return rows
	}

	best := copyRows(rows)
	bestCrossings := countCrossings(best, adj)

	current := copyRows(rows)
	const sweeps = 4
	for s := 0; s < sweeps; s++ {
		if s%2 == 0 {
			for i := 1; i < len(current); i++ {
				current[i] = sortByBarycenter(current[i], current[i-1], adj)
			}
		} else {
			for i := len(current) - 2; i >= 0; i-- {
				current[i] = sortByBarycenter(current[i], current[i+1], adj)
			}
		}
		if c := countCrossings(current, adj); c < bestCrossings {
			bestCrossings = c
			best = copyRows(current)
		}
	}

	return best
}

// copyRows returns a deep copy of rows so sweep iterations can be mutated
// freely without disturbing the caller's original (needed as the "best so
// far" baseline in minimizeCrossings).
func copyRows(rows [][]string) [][]string {
	out := make([][]string, len(rows))
	for i, row := range rows {
		out[i] = append([]string(nil), row...)
	}
	return out
}

// sortByBarycenter reorders row by the average position of each element's
// neighbors (per adj) in refRow. Elements with no neighbor in refRow have no
// signal to sort by, so they sort by their own original index — keeping
// them close to where they already were rather than letting them drift to
// an arbitrary edge on every sweep.
func sortByBarycenter(row, refRow []string, adj map[string]map[string]bool) []string {
	refPos := make(map[string]int, len(refRow))
	for i, id := range refRow {
		refPos[id] = i
	}
	origPos := make(map[string]int, len(row))
	for i, id := range row {
		origPos[id] = i
	}

	type scored struct {
		id    string
		score float64
	}
	scoredRow := make([]scored, len(row))
	for i, id := range row {
		sum, count := 0, 0
		for neighbor := range adj[id] {
			if pos, ok := refPos[neighbor]; ok {
				sum += pos
				count++
			}
		}
		if count > 0 {
			scoredRow[i] = scored{id: id, score: float64(sum) / float64(count)}
		} else {
			scoredRow[i] = scored{id: id, score: float64(origPos[id])}
		}
	}

	sort.SliceStable(scoredRow, func(a, b int) bool { return scoredRow[a].score < scoredRow[b].score })

	result := make([]string, len(row))
	for i, s := range scoredRow {
		result[i] = s.id
	}
	return result
}

// countCrossings sums, over every pair of adjacent rows, the number of edge
// pairs that cross when drawn as straight lines between the two rows — the
// standard adjacent-layer crossing count used to evaluate layered graph
// drawings.
func countCrossings(rows [][]string, adj map[string]map[string]bool) int {
	total := 0
	for i := 0; i < len(rows)-1; i++ {
		total += countCrossingsBetweenRows(rows[i], rows[i+1], adj)
	}
	return total
}

// countCrossingsBetweenRows counts crossings among edges strictly between
// upper and lower (edges within a single row, or elsewhere in the graph,
// are out of scope here — those aren't drawn between these two rows).
// Two edges (u1,l1) and (u2,l2), u1 before u2 in upper, cross iff their
// lower endpoints are in the opposite order (l1 after l2).
func countCrossingsBetweenRows(upper, lower []string, adj map[string]map[string]bool) int {
	return countPairwiseCrossings(collectRowEdges(upper, lower, adj))
}

// rowEdge is one edge between a row-index in the upper row and a row-index
// in the lower row (positions, not element IDs — crossing detection only
// needs relative order).
type rowEdge struct{ u, l int }

// collectRowEdges finds every adjacency between upper and lower and records
// it as a (upper-index, lower-index) pair.
func collectRowEdges(upper, lower []string, adj map[string]map[string]bool) []rowEdge {
	lowerPos := make(map[string]int, len(lower))
	for i, id := range lower {
		lowerPos[id] = i
	}

	var edges []rowEdge
	for ui, u := range upper {
		for neighbor := range adj[u] {
			if li, ok := lowerPos[neighbor]; ok {
				edges = append(edges, rowEdge{u: ui, l: li})
			}
		}
	}
	return edges
}

// countPairwiseCrossings counts how many pairs of edges cross: (u1,l1) and
// (u2,l2) with u1 before u2 cross iff their lower endpoints are inverted
// (l1 after l2).
func countPairwiseCrossings(edges []rowEdge) int {
	crossings := 0
	for i := 0; i < len(edges); i++ {
		for j := i + 1; j < len(edges); j++ {
			if edgesCross(edges[i], edges[j]) {
				crossings++
			}
		}
	}
	return crossings
}

func edgesCross(a, b rowEdge) bool {
	return (a.u < b.u && a.l > b.l) || (a.u > b.u && a.l < b.l)
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
