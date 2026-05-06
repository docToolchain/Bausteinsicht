package diagram

// KindStyle defines fill and stroke colors for element kinds.
type KindStyle struct {
	Fill   string
	Stroke string
}

// DefaultKindColors maps element kinds to consistent visual styles.
// Used by all diagram renderers (PlantUML, Mermaid, DOT, D2, HTML5).
var DefaultKindColors = map[string]KindStyle{
	"actor":           {Fill: "#dae8fc", Stroke: "#6c8ebf"},
	"person":          {Fill: "#dae8fc", Stroke: "#6c8ebf"},
	"system":          {Fill: "#f5f5f5", Stroke: "#666666"},
	"external_system": {Fill: "#e1d5e7", Stroke: "#9673a6"},
	"container":       {Fill: "#d5e8d4", Stroke: "#82b366"},
	"ui":              {Fill: "#d5e8d4", Stroke: "#82b366"},
	"mobile":          {Fill: "#d5e8d4", Stroke: "#82b366"},
	"datastore":       {Fill: "#fff2cc", Stroke: "#d6b656"},
	"database":        {Fill: "#fff2cc", Stroke: "#d6b656"},
	"queue":           {Fill: "#ffe6cc", Stroke: "#d5a74e"},
	"filestore":       {Fill: "#fff2cc", Stroke: "#d6b656"},
	"component":       {Fill: "#d5e8d4", Stroke: "#82b366"},
}

// ColorForKind returns the style for a given element kind.
// Falls back to a default gray color if the kind is not defined.
func ColorForKind(kind string) KindStyle {
	if style, ok := DefaultKindColors[kind]; ok {
		return style
	}
	return KindStyle{Fill: "#f5f5f5", Stroke: "#666666"}
}
