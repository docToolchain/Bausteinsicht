package template

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	etree "github.com/beevik/etree"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// Generator creates a draw.io template from an element specification.
type Generator struct {
	spec   model.Specification
	style  string
	nextID int
}

// NewGenerator creates a new template generator.
func NewGenerator(spec model.Specification, style string) *Generator {
	if style == "" {
		style = DefaultStyle
	}
	return &Generator{
		spec:   spec,
		style:  style,
		nextID: 2,
	}
}

// Generate produces the draw.io template XML as a complete mxfile.
func (g *Generator) Generate() string {
	doc := etree.NewDocument()
	doc.CreateProcInst("xml", `version="1.0" encoding="UTF-8"`)

	mxfile := doc.CreateElement("mxfile")
	mxfile.CreateAttr("host", "app.diagrams.net")
	mxfile.CreateAttr("modified", "2026-01-01T00:00:00.000Z")
	mxfile.CreateAttr("agent", "Bausteinsicht")
	mxfile.CreateAttr("version", "1.0")
	mxfile.CreateAttr("type", "device")

	diagram := mxfile.CreateElement("diagram")
	diagram.CreateAttr("id", "1")
	diagram.CreateAttr("name", "Template")

	root := diagram.CreateElement("mxGraphModel")
	root.CreateAttr("dx", "0")
	root.CreateAttr("dy", "0")
	root.CreateAttr("grid", "1")
	root.CreateAttr("gridSize", "10")
	root.CreateAttr("guides", "1")
	root.CreateAttr("tooltips", "1")
	root.CreateAttr("connect", "1")
	root.CreateAttr("arrows", "1")
	root.CreateAttr("fold", "1")
	root.CreateAttr("page", "1")
	root.CreateAttr("pageScale", "1")
	root.CreateAttr("pageWidth", "827")
	root.CreateAttr("pageHeight", "1169")
	root.CreateAttr("background", "#ffffff")
	root.CreateAttr("math", "0")
	root.CreateAttr("shadow", "0")

	rootElem := root.CreateElement("root")
	cell0 := rootElem.CreateElement("mxCell")
	cell0.CreateAttr("id", "0")
	cell1 := rootElem.CreateElement("mxCell")
	cell1.CreateAttr("id", "1")
	cell1.CreateAttr("parent", "0")

	g.nextID = 2

	// Collect kinds in sorted order
	var kinds []string
	for kind := range g.spec.Elements {
		kinds = append(kinds, kind)
	}

	// Sort for consistent output
	sort.Strings(kinds)

	// Layout elements in grid (4 columns)
	layout := GridLayout(kinds, 4)

	for _, elem := range layout {
		g.addElement(rootElem, elem.Kind, elem.Position.X, elem.Position.Y)
	}

	doc.Indent(2)
	var buf bytes.Buffer
	if _, err := doc.WriteTo(&buf); err != nil {
		return ""
	}
	return buf.String()
}

func (g *Generator) addElement(parent *etree.Element, kind string, x, y int) {
	cfg := DefaultShapeConfig(kind)
	colors := ColorForKind(g.style, kind)
	elementSpec := g.spec.Elements[kind]

	// Create label with kind name and type
	kindTitle := strings.ToUpper(kind[:1]) + kind[1:]
	label := fmt.Sprintf("<b>%s</b><br/>[%s]", kindTitle, kind)

	// Build style string
	style := g.buildStyle(cfg, colors, elementSpec)

	cell := parent.CreateElement("mxCell")
	cell.CreateAttr("id", fmt.Sprintf("%d", g.nextID))
	g.nextID++
	cell.CreateAttr("value", label)           // Don't escape: draw.io uses html=1 and requires raw HTML markup for rich text
	cell.CreateAttr("style", style+";html=1") // Enable HTML rendering in draw.io
	cell.CreateAttr("vertex", "1")
	cell.CreateAttr("parent", "1")

	geometry := cell.CreateElement("mxGeometry")
	geometry.CreateAttr("x", fmt.Sprintf("%d", x))
	geometry.CreateAttr("y", fmt.Sprintf("%d", y))
	geometry.CreateAttr("width", fmt.Sprintf("%d", cfg.Width))
	geometry.CreateAttr("height", fmt.Sprintf("%d", cfg.Height))
	geometry.CreateAttr("as", "geometry")
}

func (g *Generator) buildStyle(cfg ShapeConfig, colors ColorStyle, _ model.ElementKind) string {
	parts := []string{
		fmt.Sprintf("fillColor=%s", colors.Fill),
		fmt.Sprintf("strokeColor=%s", colors.Stroke),
		"fontColor=#000000",
		"fontSize=12",
	}

	// Add shape if specified
	if cfg.Shape != "" {
		if strings.HasPrefix(cfg.Shape, "shape=") {
			parts = append(parts, cfg.Shape)
		} else if !strings.Contains(cfg.Shape, "=") {
			parts = append(parts, fmt.Sprintf("shape=%s", cfg.Shape))
		} else {
			parts = append(parts, cfg.Shape)
		}
	}

	return strings.Join(parts, ";")
}
