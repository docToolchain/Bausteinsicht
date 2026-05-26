package layout

import (
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// HierarchicalLayout computes layer assignments via longest-path algorithm,
// then positions elements in layers with horizontal alignment.
type HierarchicalLayout struct {
	model     *model.BausteinsichtModel
	rankDir   string // TB or LR
	spacing   float64
	layerGap  float64
}

// NewHierarchicalLayout creates a hierarchical layout engine.
func NewHierarchicalLayout(m *model.BausteinsichtModel, rankDir string) *HierarchicalLayout {
	if rankDir == "" {
		rankDir = "TB"
	}
	return &HierarchicalLayout{
		model:    m,
		rankDir:  rankDir,
		spacing:  20,  // pixels between elements in same layer
		layerGap: 100, // pixels between layers
	}
}

// Compute calculates positions for all elements.
func (h *HierarchicalLayout) Compute() LayoutResult {
	outgoing := h.buildOutgoingMap()

	// Assign layers using longest-path algorithm
	layers := h.assignLayers(outgoing)

	// Position elements based on layers
	positions := h.positionElements(layers)

	return LayoutResult{
		Positions: positions,
		Algorithm: Hierarchical,
	}
}

// buildOutgoingMap creates a map of outgoing relationships per element.
func (h *HierarchicalLayout) buildOutgoingMap() map[string][]string {
	outgoing := make(map[string][]string)
	for _, rel := range h.model.Relationships {
		outgoing[rel.From] = append(outgoing[rel.From], rel.To)
	}
	return outgoing
}

// assignLayers assigns each element to a layer using longest-path algorithm.
func (h *HierarchicalLayout) assignLayers(outgoing map[string][]string) map[int][]string {
	flat, _ := model.FlattenElements(h.model)

	// Compute longest path from each node
	depths := make(map[string]int)
	for id := range flat {
		depths[id] = h.longestPath(id, outgoing, make(map[string]bool))
	}

	// Group elements by layer
	layers := make(map[int][]string)
	maxLayer := 0
	for id, depth := range depths {
		layers[depth] = append(layers[depth], id)
		if depth > maxLayer {
			maxLayer = depth
		}
	}

	return layers
}

// longestPath computes longest outgoing path from a node (memoized).
func (h *HierarchicalLayout) longestPath(id string, outgoing map[string][]string, visited map[string]bool) int {
	if visited[id] {
		return 0 // cycle detected, break here
	}

	targets := outgoing[id]
	if len(targets) == 0 {
		return 0
	}

	visited[id] = true
	maxDepth := 0
	for _, target := range targets {
		depth := h.longestPath(target, outgoing, visited)
		if depth > maxDepth {
			maxDepth = depth
		}
	}
	delete(visited, id)

	return maxDepth + 1
}

// positionElements places elements horizontally within each layer.
func (h *HierarchicalLayout) positionElements(layers map[int][]string) map[string]ElementPosition {
	positions := make(map[string]ElementPosition)

	for layer, ids := range layers {
		// Default sizes
		elemWidth := 160.0
		elemHeight := 60.0

		// Calculate layer positions
		var x, y float64
		if h.rankDir == "TB" {
			// Top-to-bottom: layer determines Y, elements spread horizontally
			y = float64(layer) * h.layerGap
			x = 50.0
		} else {
			// Left-to-right: layer determines X, elements spread vertically
			x = float64(layer) * h.layerGap
			y = 50.0
		}

		// Position elements in this layer
		for i, id := range ids {
			elemX, elemY := x, y
			if h.rankDir == "TB" {
				elemX = x + float64(i)*(elemWidth+h.spacing)
			} else {
				elemY = y + float64(i)*(elemHeight+h.spacing)
			}

			positions[id] = ElementPosition{
				ID:     id,
				X:      elemX,
				Y:      elemY,
				Width:  elemWidth,
				Height: elemHeight,
				Layer:  layer,
			}
		}
	}

	return positions
}
