package template

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	etree "github.com/beevik/etree"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// templateVersion is the bausteinsicht_template_version written to generated
// templates. It must match drawio.CurrentTemplateVersion so LoadTemplate accepts
// the output.
const templateVersion = "1"

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
//
// The output matches the structure the sync engine's LoadTemplate expects:
//   - <object bausteinsicht_template="<kind>"> wrappers carrying the element style
//   - "-title"/"-tech"/"-desc" sub-cells for grouped text labels
//   - "<kind>_boundary" templates for container kinds
//   - a "relationship" connector template
//   - a bausteinsicht_template_version attribute on the <mxfile> root
//
// This makes a generated template directly usable as --template without falling
// back to default styles.
func (g *Generator) Generate() string {
	doc := etree.NewDocument()
	doc.CreateProcInst("xml", `version="1.0" encoding="UTF-8"`)

	mxfile := doc.CreateElement("mxfile")
	mxfile.CreateAttr("host", "app.diagrams.net")
	mxfile.CreateAttr("modified", "2026-01-01T00:00:00.000Z")
	mxfile.CreateAttr("agent", "Bausteinsicht")
	mxfile.CreateAttr("version", "1.0")
	mxfile.CreateAttr("type", "device")
	mxfile.CreateAttr("bausteinsicht_template_version", templateVersion)

	diagram := mxfile.CreateElement("diagram")
	diagram.CreateAttr("id", "template")
	diagram.CreateAttr("name", "Bausteinsicht Template")

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

	// Collect kinds in sorted order for deterministic output.
	var kinds []string
	for kind := range g.spec.Elements {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)

	// Layout element templates in a grid (4 columns).
	layout := GridLayout(kinds, 4)
	for _, elem := range layout {
		g.addElementTemplate(rootElem, elem.Kind, elem.Position.X, elem.Position.Y)
	}

	// Boundary templates for container kinds, laid out below the element grid.
	g.addBoundaryTemplates(rootElem, kinds, layout)

	// Relationship connector template (bare mxCell carrying the connector style).
	g.addConnectorTemplate(rootElem)

	doc.Indent(2)
	var buf bytes.Buffer
	if _, err := doc.WriteTo(&buf); err != nil {
		return ""
	}
	return buf.String()
}

// addElementTemplate writes one <object bausteinsicht_template="kind"> wrapper
// plus its title/tech/desc sub-cells.
func (g *Generator) addElementTemplate(parent *etree.Element, kind string, x, y int) {
	cfg := DefaultShapeConfig(kind)
	colors := ColorForKind(g.style, kind)
	style := g.buildStyle(cfg, colors)

	objID := "template-" + kind

	obj := parent.CreateElement("object")
	obj.CreateAttr("label", "")
	obj.CreateAttr("bausteinsicht_template", kind)
	obj.CreateAttr("id", objID)

	cell := obj.CreateElement("mxCell")
	cell.CreateAttr("parent", "1")
	cell.CreateAttr("style", style)
	cell.CreateAttr("vertex", "1")

	geo := cell.CreateElement("mxGeometry")
	geo.CreateAttr("x", fmt.Sprintf("%d", x))
	geo.CreateAttr("y", fmt.Sprintf("%d", y))
	geo.CreateAttr("width", fmt.Sprintf("%d", cfg.Width))
	geo.CreateAttr("height", fmt.Sprintf("%d", cfg.Height))
	geo.CreateAttr("as", "geometry")

	// Sub-cells: title, technology, description. Positions are relative to the
	// parent object's geometry, stacked vertically.
	titleH, techH := 24, 18
	descH := cfg.Height - titleH - techH
	if descH < 18 {
		descH = 18
	}

	kindTitle := strings.ToUpper(kind[:1]) + kind[1:]
	g.addSubCell(parent, objID, "title", kindTitle,
		fmt.Sprintf("text;html=1;fontSize=13;fontStyle=1;fontColor=%s;fillColor=none;strokeColor=none;align=center;verticalAlign=middle;", colors.Stroke),
		0, 0, cfg.Width, titleH)
	g.addSubCell(parent, objID, "tech", "[Technology]",
		"text;html=1;fontSize=10;fontStyle=2;fontColor=#666666;fillColor=none;strokeColor=none;align=center;verticalAlign=middle;",
		0, titleH, cfg.Width, techH)
	g.addSubCell(parent, objID, "desc", "Description",
		"text;html=1;whiteSpace=wrap;fontSize=10;fontColor=#555555;fillColor=none;strokeColor=none;align=center;verticalAlign=middle;",
		0, titleH+techH, cfg.Width, descH)
}

// addSubCell writes a "-title"/"-tech"/"-desc" text sub-cell parented to objID.
func (g *Generator) addSubCell(parent *etree.Element, objID, role, value, style string, x, y, w, h int) {
	cell := parent.CreateElement("mxCell")
	cell.CreateAttr("id", objID+"-"+role)
	cell.CreateAttr("parent", objID)
	cell.CreateAttr("style", style)
	cell.CreateAttr("value", value)
	cell.CreateAttr("vertex", "1")

	geo := cell.CreateElement("mxGeometry")
	geo.CreateAttr("x", fmt.Sprintf("%d", x))
	geo.CreateAttr("y", fmt.Sprintf("%d", y))
	geo.CreateAttr("width", fmt.Sprintf("%d", w))
	geo.CreateAttr("height", fmt.Sprintf("%d", h))
	geo.CreateAttr("as", "geometry")
}

// addBoundaryTemplates writes a "<kind>_boundary" template for every container
// kind in the spec, placed in a row beneath the element grid.
func (g *Generator) addBoundaryTemplates(parent *etree.Element, kinds []string, layout []Element) {
	// Find the lowest point of the element grid so boundaries sit below it.
	y := 40
	for _, elem := range layout {
		cfg := DefaultShapeConfig(elem.Kind)
		bottom := elem.Position.Y + cfg.Height
		if bottom > y {
			y = bottom
		}
	}
	y += 60

	x := 40
	for _, kind := range kinds {
		if !g.spec.Elements[kind].Container {
			continue
		}
		boundaryKind := kind + "_boundary"
		colors := ColorForKind(g.style, kind)
		style := fmt.Sprintf(
			"swimlane;startSize=30;fillColor=none;strokeColor=%s;fontColor=%s;fontStyle=1;rounded=1;arcSize=5;whiteSpace=wrap;html=1;container=1;collapsible=0;dashed=1;dashPattern=8 4;fontSize=13;",
			colors.Stroke, colors.Stroke)

		obj := parent.CreateElement("object")
		obj.CreateAttr("label", strings.ToUpper(kind[:1])+kind[1:]+" Boundary")
		obj.CreateAttr("bausteinsicht_template", boundaryKind)
		obj.CreateAttr("id", "template-"+boundaryKind)

		cell := obj.CreateElement("mxCell")
		cell.CreateAttr("parent", "1")
		cell.CreateAttr("style", style)
		cell.CreateAttr("vertex", "1")

		geo := cell.CreateElement("mxGeometry")
		geo.CreateAttr("x", fmt.Sprintf("%d", x))
		geo.CreateAttr("y", fmt.Sprintf("%d", y))
		geo.CreateAttr("width", "300")
		geo.CreateAttr("height", "200")
		geo.CreateAttr("as", "geometry")

		x += 340
	}
}

// addConnectorTemplate writes the relationship connector template: a bare mxCell
// carrying bausteinsicht_template="relationship".
func (g *Generator) addConnectorTemplate(parent *etree.Element) {
	cell := parent.CreateElement("mxCell")
	cell.CreateAttr("id", "template-relationship")
	cell.CreateAttr("bausteinsicht_template", "relationship")
	cell.CreateAttr("parent", "1")
	cell.CreateAttr("edge", "1")
	cell.CreateAttr("value", "<b>label</b>")
	cell.CreateAttr("style", "edgeStyle=orthogonalEdgeStyle;rounded=1;orthogonalLoop=1;jettySize=auto;html=1;endArrow=block;endFill=1;strokeColor=#707070;fontColor=#707070;fontSize=11;strokeWidth=1.5;")

	geo := cell.CreateElement("mxGeometry")
	geo.CreateAttr("relative", "1")
	geo.CreateAttr("as", "geometry")
}

// buildStyle composes the mxCell style string for an element template.
func (g *Generator) buildStyle(cfg ShapeConfig, colors ColorStyle) string {
	parts := []string{
		fmt.Sprintf("fillColor=%s", colors.Fill),
		fmt.Sprintf("strokeColor=%s", colors.Stroke),
		"fontColor=#000000",
		"fontSize=12",
		"whiteSpace=wrap",
		"html=1",
		"align=center",
		"verticalAlign=middle",
		"container=0",
	}

	// Add shape if specified.
	if cfg.Shape != "" {
		switch {
		case strings.HasPrefix(cfg.Shape, "shape="):
			parts = append(parts, cfg.Shape)
		case !strings.Contains(cfg.Shape, "="):
			parts = append(parts, fmt.Sprintf("shape=%s", cfg.Shape))
		default:
			parts = append(parts, cfg.Shape)
		}
	}

	return strings.Join(parts, ";")
}
