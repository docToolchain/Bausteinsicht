package layout

// Algorithm specifies the layout computation method.
type Algorithm int

const (
	Hierarchical Algorithm = iota
	ForceDirected
	Radial
)

// Config holds layout computation parameters.
type Config struct {
	Algorithm     Algorithm
	View          string // if empty, layout all views
	PreservePinned bool
	RankDir       string // TB (top-to-bottom) or LR (left-to-right)
}

// ElementPosition represents a positioned element.
type ElementPosition struct {
	ID     string
	X, Y   float64
	Width  float64
	Height float64
	Layer  int  // for hierarchical layout
	Pinned bool // read from draw.io
}

// LayoutResult holds computed positions.
type LayoutResult struct {
	Positions map[string]ElementPosition
	Algorithm Algorithm
	ViewKey   string
}
