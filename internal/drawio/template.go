package drawio

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/beevik/etree"
)

// CurrentTemplateVersion is the latest template format version supported.
const CurrentTemplateVersion = 1

// SubCellStyle holds style and geometry for a text sub-cell within an element.
type SubCellStyle struct {
	Style         string  // mxCell style string
	X, Y          float64 // position relative to parent
	Width, Height float64 // dimensions
}

// TemplateStyle holds the visual style and default dimensions for a draw.io element.
type TemplateStyle struct {
	Style  string  // mxCell style string
	Width  float64 // default width from mxGeometry
	Height float64 // default height from mxGeometry

	// Sub-cell styles for grouped text labels (title, technology, description).
	// Nil means no sub-cells defined (legacy template).
	TitleStyle *SubCellStyle
	TechStyle  *SubCellStyle
	DescStyle  *SubCellStyle
}

// TemplateSet holds all styles parsed from a draw.io template file.
type TemplateSet struct {
	Version    int                      // template format version (0 means unset/v1)
	elements   map[string]TemplateStyle // keyed by kind (actor, system, container, component)
	boundaries map[string]TemplateStyle // keyed by kind (system_boundary, container_boundary)
	connector  string                   // default connector style
}

// LoadTemplate parses a draw.io template file from disk.
func LoadTemplate(path string) (*TemplateSet, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- path from CLI flag
	if err != nil {
		return nil, fmt.Errorf("LoadTemplate %q: %w", path, err)
	}
	return LoadTemplateFromBytes(data)
}

// LoadTemplateFromBytes parses a draw.io template from raw XML bytes.
func LoadTemplateFromBytes(data []byte) (*TemplateSet, error) {
	tree := etree.NewDocument()
	if err := tree.ReadFromBytes(data); err != nil {
		return nil, fmt.Errorf("LoadTemplateFromBytes: %w", err)
	}

	// Validate that the document is a valid draw.io file with an <mxfile> root.
	root := tree.Root()
	if root == nil || root.Tag != "mxfile" {
		return nil, fmt.Errorf("LoadTemplateFromBytes: not a valid draw.io template (missing <mxfile> root element)")
	}

	// Read template version from <mxfile> root. Missing → version 1 (backward compat).
	version := 1
	if vStr := root.SelectAttrValue("bausteinsicht_template_version", ""); vStr != "" {
		v, err := strconv.Atoi(vStr)
		if err != nil {
			return nil, fmt.Errorf("LoadTemplateFromBytes: invalid template version %q: %w", vStr, err)
		}
		version = v
	}
	if version > CurrentTemplateVersion {
		return nil, fmt.Errorf("LoadTemplateFromBytes: template version %d not supported (max: %d)", version, CurrentTemplateVersion)
	}

	ts := &TemplateSet{
		Version:    version,
		elements:   make(map[string]TemplateStyle),
		boundaries: make(map[string]TemplateStyle),
	}

	// Build maps for looking up child cells.
	templateIDs := make(map[string]string) // template object id → kind
	groupToKind := make(map[string]string) // group cell id → kind (when template is inside a group)

	// Find all <object> elements with bausteinsicht_template attribute.
	for _, obj := range tree.FindElements("//object[@bausteinsicht_template]") {
		kind := obj.SelectAttrValue("bausteinsicht_template", "")
		if kind == "" {
			continue
		}

		cell := obj.FindElement("mxCell")
		if cell == nil {
			continue
		}

		style := cell.SelectAttrValue("style", "")
		width, height := parseGeometry(cell)

		objID := obj.SelectAttrValue("id", "")
		if objID != "" {
			templateIDs[objID] = kind
		}

		// If the template mxCell's parent is not "1", it's inside a group.
		// Map the group ID so sub-cells with that group parent are found too.
		cellParent := cell.SelectAttrValue("parent", "1")
		if cellParent != "1" && cellParent != "0" && cellParent != "" {
			groupToKind[cellParent] = kind
		}

		ts.categorize(kind, TemplateStyle{Style: style, Width: width, Height: height})
	}

	// Parse child mxCells that are sub-cells of template elements.
	// Sub-cells may be children of the template object ID directly,
	// or children of a group cell that contains the template object.
	for _, cell := range tree.FindElements("//mxCell[@parent]") {
		parentID := cell.SelectAttrValue("parent", "")
		kind, ok := templateIDs[parentID]
		if !ok {
			kind, ok = groupToKind[parentID]
		}
		if !ok {
			continue
		}
		cellID := cell.SelectAttrValue("id", "")
		if cellID == "" {
			continue
		}
		sub := parseSubCellStyle(cell)
		if sub == nil {
			continue
		}

		// Determine role from cell ID suffix or value heuristic.
		role := ""
		switch {
		case strings.HasSuffix(cellID, "-title"):
			role = "title"
		case strings.HasSuffix(cellID, "-tech"):
			role = "tech"
		case strings.HasSuffix(cellID, "-desc"):
			role = "desc"
		default:
			// Fallback: detect role by value attribute when ID doesn't follow convention
			// (e.g., draw.io-generated IDs from manual template editing).
			val := strings.TrimSpace(cell.SelectAttrValue("value", ""))
			switch {
			case strings.EqualFold(val, "title") || strings.HasSuffix(val, " Name"):
				role = "title"
			case strings.EqualFold(val, "[technology]"):
				role = "tech"
			case strings.EqualFold(val, "description"):
				role = "desc"
			}
		}
		if role != "" {
			ts.setSubCell(kind, role, sub)
		}
	}

	// Find relationship connector: bare <mxCell bausteinsicht_template="relationship">.
	for _, cell := range tree.FindElements("//mxCell[@bausteinsicht_template='relationship']") {
		ts.connector = cell.SelectAttrValue("style", "")
	}

	return ts, nil
}

// GetStyle returns the TemplateStyle for a given element kind.
func (t *TemplateSet) GetStyle(kind string) (TemplateStyle, bool) {
	s, ok := t.elements[kind]
	return s, ok
}

// GetBoundaryStyle returns the TemplateStyle for a given boundary kind.
func (t *TemplateSet) GetBoundaryStyle(kind string) (TemplateStyle, bool) {
	s, ok := t.boundaries[kind]
	return s, ok
}

// GetConnectorStyle returns the default connector style string.
func (t *TemplateSet) GetConnectorStyle() string {
	return t.connector
}

// GetAllStyles returns a copy of all element styles keyed by kind.
func (t *TemplateSet) GetAllStyles() map[string]TemplateStyle {
	out := make(map[string]TemplateStyle, len(t.elements))
	for k, v := range t.elements {
		out[k] = v
	}
	return out
}

// categorize places the style into the appropriate map based on its kind.
func (t *TemplateSet) categorize(kind string, style TemplateStyle) {
	if strings.HasSuffix(kind, "_boundary") {
		t.boundaries[kind] = style
	} else {
		t.elements[kind] = style
	}
}

// setSubCell assigns a parsed sub-cell style to the appropriate field on a TemplateStyle.
func (t *TemplateSet) setSubCell(kind, role string, sub *SubCellStyle) {
	// Look up in both maps.
	if ts, ok := t.elements[kind]; ok {
		setSubCellOnStyle(&ts, role, sub)
		t.elements[kind] = ts
	}
	if ts, ok := t.boundaries[kind]; ok {
		setSubCellOnStyle(&ts, role, sub)
		t.boundaries[kind] = ts
	}
}

func setSubCellOnStyle(ts *TemplateStyle, role string, sub *SubCellStyle) {
	switch role {
	case "title":
		ts.TitleStyle = sub
	case "tech":
		ts.TechStyle = sub
	case "desc":
		ts.DescStyle = sub
	}
}

// parseSubCellStyle parses style and geometry from a child mxCell element.
func parseSubCellStyle(cell *etree.Element) *SubCellStyle {
	style := cell.SelectAttrValue("style", "")
	if style == "" {
		return nil
	}
	geo := cell.FindElement("mxGeometry")
	if geo == nil {
		return nil
	}
	x, _ := strconv.ParseFloat(geo.SelectAttrValue("x", "0"), 64)
	y, _ := strconv.ParseFloat(geo.SelectAttrValue("y", "0"), 64)
	w, _ := strconv.ParseFloat(geo.SelectAttrValue("width", "0"), 64)
	h, _ := strconv.ParseFloat(geo.SelectAttrValue("height", "0"), 64)
	return &SubCellStyle{Style: style, X: x, Y: y, Width: w, Height: h}
}

// parseGeometry extracts width and height from an mxCell's nested mxGeometry element.
func parseGeometry(cell *etree.Element) (float64, float64) {
	geo := cell.FindElement("mxGeometry")
	if geo == nil {
		return 0, 0
	}
	w, _ := strconv.ParseFloat(geo.SelectAttrValue("width", "0"), 64)
	h, _ := strconv.ParseFloat(geo.SelectAttrValue("height", "0"), 64)
	return w, h
}
