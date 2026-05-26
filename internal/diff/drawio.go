package diff

// DrawIO color definitions for diff visualization
const (
	ColorAdded   = "#d5e8d4"    // green
	ColorRemoved = "#f8cecc"    // red
	ColorChanged = "#ffe6cc"    // orange
	ColorUnchanged = "#ffffff"  // white (default)

	StrokeAdded   = "#82b366"   // dark green
	StrokeRemoved = "#b85450"   // dark red
	StrokeChanged = "#d6b656"   // dark orange
)

// AppliedChangeStyle returns the fill and stroke colors for a changed element
func GetChangeColors(changeType ChangeType) (fillColor, strokeColor string) {
	switch changeType {
	case ChangeAdded:
		return ColorAdded, StrokeAdded
	case ChangeRemoved:
		return ColorRemoved, StrokeRemoved
	case ChangeChanged:
		return ColorChanged, StrokeChanged
	default:
		return ColorUnchanged, "#999999"
	}
}

// ElementStyle describes visual styling for a draw.io element
type ElementStyle struct {
	FillColor   string
	StrokeColor string
	StrokeWidth float64
	Opacity     float64
	Label       string // For removed elements, add strikethrough indicator
}

// GetElementStyle returns the draw.io styling for a given element change
func GetElementStyle(change ElementChange) ElementStyle {
	fillColor, strokeColor := GetChangeColors(change.Type)

	style := ElementStyle{
		FillColor:   fillColor,
		StrokeColor: strokeColor,
		StrokeWidth: 2,
		Opacity:     1.0,
	}

	// For removed elements, add visual indication
	if change.Type == ChangeRemoved && change.AsIs != nil {
		style.Label = "~" + change.AsIs.Title // strikethrough indicator
	}

	return style
}
