package sync

import (
	"math"
	"sort"

	"github.com/docToolchain/Bauteinsicht/internal/drawio"
	"github.com/docToolchain/Bauteinsicht/internal/model"
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
	elementGap: 40,
	padding:    60,
	startX:     40,
	startY:     40,
}

// layoutResult holds computed positions and optional boundary dimensions.
type layoutResult struct {
	Positions     map[string]position
	BoundaryWidth float64
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
) layoutResult {
	switch layout {
	case "grid":
		return computeGridLayout(ids, flat, templates)
	case "none":
		return computeNoneLayout(ids, flat, templates)
	default: // "layered" or ""
		return computeLayeredLayout(ids, flat, templates, elementOrder, scopeID)
	}
}

// computeLayeredLayout arranges elements in horizontal rows grouped by kind.
// Kinds are ordered according to elementOrder (from specification.elements).
// Elements within each layer are sorted alphabetically.
func computeLayeredLayout(
	ids []string,
	flat map[string]*model.Element,
	templates *drawio.TemplateSet,
	elementOrder []string,
	scopeID string,
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
	var scopeChildren []string
	var externals []string
	for _, id := range ids {
		if id == scopeID {
			continue
		}
		if scopeID != "" && isChildOf(id, scopeID) {
			scopeChildren = append(scopeChildren, id)
		} else {
			externals = append(externals, id)
		}
	}

	// Place scoped children inside the boundary area.
	if scopeID != "" && len(scopeChildren) > 0 {
		boundaryStartSize := 30.0 // header height of the boundary swimlane
		innerX := cfg.padding
		innerY := boundaryStartSize + cfg.padding

		contentW, contentH := placeLayered(scopeChildren, flat, templates, kindTier, maxTier, cfg, innerX, innerY, result.Positions)

		result.BoundaryWidth = contentW + 2*cfg.padding
		result.BoundaryHeight = contentH + boundaryStartSize + 2*cfg.padding

		// Minimum boundary size.
		if result.BoundaryWidth < 400 {
			result.BoundaryWidth = 400
		}
		if result.BoundaryHeight < 300 {
			result.BoundaryHeight = 300
		}
	}

	// Place external elements below the boundary (or at startX/startY if no boundary).
	if len(externals) > 0 {
		extStartY := cfg.startY
		if result.BoundaryHeight > 0 {
			extStartY = cfg.startY + result.BoundaryHeight + cfg.elementGap
		}
		placeLayered(externals, flat, templates, kindTier, maxTier, cfg, cfg.startX, extStartY, result.Positions)
	}

	return result
}

// placeLayered places elements in layered rows and returns the content width and height.
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

	// Place layer by layer.
	curY := originY
	maxRowWidth := 0.0

	for _, tier := range tierKeys {
		elems := tiers[tier]
		sort.Strings(elems)

		curX := originX
		rowHeight := 0.0

		for _, id := range elems {
			w, h := elementSize(id, flat, templates)

			// Row wrapping.
			if curX > originX && curX+w > cfg.pageWidth-cfg.startX {
				curY += rowHeight + cfg.elementGap
				curX = originX
				rowHeight = 0
			}

			positions[id] = position{X: curX, Y: curY}
			curX += w + cfg.elementGap
			if curX-cfg.elementGap-originX > maxRowWidth {
				maxRowWidth = curX - cfg.elementGap - originX
			}
			if h > rowHeight {
				rowHeight = h
			}
		}

		curY += rowHeight + cfg.elementGap
	}

	contentWidth = maxRowWidth
	contentHeight = curY - originY - cfg.elementGap // subtract trailing gap
	if contentHeight < 0 {
		contentHeight = 0
	}
	return contentWidth, contentHeight
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
