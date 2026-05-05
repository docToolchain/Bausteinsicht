package template

// ShapeConfig defines the draw.io shape and dimensions for a kind.
type ShapeConfig struct {
	Shape  string
	Width  int
	Height int
}

// KindShapes maps element kinds to their shape configurations.
var KindShapes = map[string]ShapeConfig{
	"person":          {Shape: "mxgraph.archimate3.actor", Width: 60, Height: 80},
	"actor":           {Shape: "mxgraph.archimate3.actor", Width: 60, Height: 80},
	"system":          {Shape: "rounded=1", Width: 160, Height: 60},
	"service":         {Shape: "rounded=1", Width: 120, Height: 60},
	"container":       {Shape: "rounded=1;container=1", Width: 200, Height: 120},
	"database":        {Shape: "mxgraph.flowchart.database", Width: 60, Height: 80},
	"datastore":       {Shape: "mxgraph.flowchart.database", Width: 60, Height: 80},
	"cache":           {Shape: "mxgraph.flowchart.stored_data", Width: 80, Height: 60},
	"queue":           {Shape: "mxgraph.flowchart.process", Width: 120, Height: 60},
	"filestore":       {Shape: "mxgraph.flowchart.stored_data", Width: 80, Height: 60},
	"component":       {Shape: "rounded=1", Width: 120, Height: 60},
	"frontend":        {Shape: "rounded=1", Width: 120, Height: 60},
	"mobile":          {Shape: "mxgraph.iphone.phone3", Width: 60, Height: 100},
	"ui":              {Shape: "rounded=1", Width: 120, Height: 60},
	"external_system": {Shape: "dashed=1;dashPattern=5 5;rounded=1", Width: 160, Height: 60},
}

// ColorStyle defines fill and stroke colors for a style preset.
type ColorStyle struct {
	Fill   string
	Stroke string
}

// StylePresets defines the visual presets for different kinds.
var StylePresets = map[string]map[string]ColorStyle{
	"default": {
		"person":          {Fill: "#dae8fc", Stroke: "#6c8ebf"},
		"actor":           {Fill: "#dae8fc", Stroke: "#6c8ebf"},
		"system":          {Fill: "#d5e8d4", Stroke: "#82b366"},
		"service":         {Fill: "#d5e8d4", Stroke: "#82b366"},
		"container":       {Fill: "#d5e8d4", Stroke: "#82b366"},
		"database":        {Fill: "#fff2cc", Stroke: "#d6b656"},
		"datastore":       {Fill: "#fff2cc", Stroke: "#d6b656"},
		"cache":           {Fill: "#fff2cc", Stroke: "#d6b656"},
		"queue":           {Fill: "#f8cecc", Stroke: "#b85450"},
		"filestore":       {Fill: "#fff2cc", Stroke: "#d6b656"},
		"component":       {Fill: "#d5e8d4", Stroke: "#82b366"},
		"frontend":        {Fill: "#e1d5e7", Stroke: "#9673a6"},
		"mobile":          {Fill: "#e1d5e7", Stroke: "#9673a6"},
		"ui":              {Fill: "#e1d5e7", Stroke: "#9673a6"},
		"external_system": {Fill: "#f5f5f5", Stroke: "#999999"},
	},
	"c4": {
		"person":          {Fill: "#08427b", Stroke: "#08427b"},
		"actor":           {Fill: "#08427b", Stroke: "#08427b"},
		"system":          {Fill: "#1168bd", Stroke: "#0b4884"},
		"service":         {Fill: "#1168bd", Stroke: "#0b4884"},
		"container":       {Fill: "#1168bd", Stroke: "#0b4884"},
		"database":        {Fill: "#438dd5", Stroke: "#3c7fc0"},
		"datastore":       {Fill: "#438dd5", Stroke: "#3c7fc0"},
		"cache":           {Fill: "#438dd5", Stroke: "#3c7fc0"},
		"queue":           {Fill: "#999999", Stroke: "#666666"},
		"filestore":       {Fill: "#438dd5", Stroke: "#3c7fc0"},
		"component":       {Fill: "#438dd5", Stroke: "#3c7fc0"},
		"frontend":        {Fill: "#1168bd", Stroke: "#0b4884"},
		"mobile":          {Fill: "#1168bd", Stroke: "#0b4884"},
		"ui":              {Fill: "#1168bd", Stroke: "#0b4884"},
		"external_system": {Fill: "#999999", Stroke: "#666666"},
	},
	"minimal": {
		"person":          {Fill: "#ffffff", Stroke: "#999999"},
		"actor":           {Fill: "#ffffff", Stroke: "#999999"},
		"system":          {Fill: "#ffffff", Stroke: "#999999"},
		"service":         {Fill: "#ffffff", Stroke: "#999999"},
		"container":       {Fill: "#ffffff", Stroke: "#999999"},
		"database":        {Fill: "#ffffff", Stroke: "#999999"},
		"datastore":       {Fill: "#ffffff", Stroke: "#999999"},
		"cache":           {Fill: "#ffffff", Stroke: "#999999"},
		"queue":           {Fill: "#ffffff", Stroke: "#999999"},
		"filestore":       {Fill: "#ffffff", Stroke: "#999999"},
		"component":       {Fill: "#ffffff", Stroke: "#999999"},
		"frontend":        {Fill: "#ffffff", Stroke: "#999999"},
		"mobile":          {Fill: "#ffffff", Stroke: "#999999"},
		"ui":              {Fill: "#ffffff", Stroke: "#999999"},
		"external_system": {Fill: "#ffffff", Stroke: "#999999"},
	},
	"dark": {
		"person":          {Fill: "#ffb74d", Stroke: "#ff8a00"},
		"actor":           {Fill: "#ffb74d", Stroke: "#ff8a00"},
		"system":          {Fill: "#4dd0e1", Stroke: "#00acc1"},
		"service":         {Fill: "#4dd0e1", Stroke: "#00acc1"},
		"container":       {Fill: "#4dd0e1", Stroke: "#00acc1"},
		"database":        {Fill: "#81c784", Stroke: "#66bb6a"},
		"datastore":       {Fill: "#81c784", Stroke: "#66bb6a"},
		"cache":           {Fill: "#81c784", Stroke: "#66bb6a"},
		"queue":           {Fill: "#e57373", Stroke: "#ef5350"},
		"filestore":       {Fill: "#81c784", Stroke: "#66bb6a"},
		"component":       {Fill: "#4dd0e1", Stroke: "#00acc1"},
		"frontend":        {Fill: "#ba68c8", Stroke: "#ab47bc"},
		"mobile":          {Fill: "#ba68c8", Stroke: "#ab47bc"},
		"ui":              {Fill: "#ba68c8", Stroke: "#ab47bc"},
		"external_system": {Fill: "#bbdefb", Stroke: "#64b5f6"},
	},
}

// DefaultStyle is the default visual preset.
const DefaultStyle = "default"

// ColorForKind returns the color style for a kind in a given preset.
// Falls back to default preset if not found.
func ColorForKind(preset, kind string) ColorStyle {
	if colors, ok := StylePresets[preset]; ok {
		if color, ok := colors[kind]; ok {
			return color
		}
	}
	// Fall back to default preset
	if colors, ok := StylePresets[DefaultStyle]; ok {
		if color, ok := colors[kind]; ok {
			return color
		}
	}
	// Ultimate fallback
	return ColorStyle{Fill: "#d5e8d4", Stroke: "#82b366"}
}

// DefaultShapeConfig returns the shape config for a kind.
// Falls back to rounded rectangle if not found.
func DefaultShapeConfig(kind string) ShapeConfig {
	if cfg, ok := KindShapes[kind]; ok {
		return cfg
	}
	return ShapeConfig{Shape: "rounded=1", Width: 120, Height: 60}
}
