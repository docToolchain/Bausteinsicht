package template

// Position represents the X, Y coordinates of an element.
type Position struct {
	X int
	Y int
}

// Element holds a kind and its position.
type Element struct {
	Kind     string
	Position Position
}

// GridLayout arranges elements in a grid.
func GridLayout(kinds []string, cols int) []Element {
	if cols <= 0 {
		cols = 4
	}

	var elements []Element
	x, y := 40, 40
	colCount := 0
	maxHeight := 0

	for _, kind := range kinds {
		cfg := DefaultShapeConfig(kind)
		elements = append(elements, Element{
			Kind:     kind,
			Position: Position{X: x, Y: y},
		})

		if cfg.Height > maxHeight {
			maxHeight = cfg.Height
		}

		colCount++
		if colCount >= cols {
			x = 40
			y += maxHeight + 40
			colCount = 0
			maxHeight = 0
		} else {
			x += cfg.Width + 40
		}
	}

	return elements
}
