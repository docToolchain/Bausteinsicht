package drawio

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/beevik/etree"
)

// TemplateStyle holds the visual style and default dimensions for a draw.io element.
type TemplateStyle struct {
	Style  string  // mxCell style string
	Width  float64 // default width from mxGeometry
	Height float64 // default height from mxGeometry
}

// TemplateSet holds all styles parsed from a draw.io template file.
type TemplateSet struct {
	elements   map[string]TemplateStyle // keyed by kind (actor, system, container, component)
	boundaries map[string]TemplateStyle // keyed by kind (system_boundary, container_boundary)
	connector  string                   // default connector style
}

// LoadTemplate parses a draw.io template file from disk.
func LoadTemplate(path string) (*TemplateSet, error) {
	data, err := os.ReadFile(path)
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

	ts := &TemplateSet{
		elements:   make(map[string]TemplateStyle),
		boundaries: make(map[string]TemplateStyle),
	}

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

		ts.categorize(kind, TemplateStyle{Style: style, Width: width, Height: height})
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

// categorize places the style into the appropriate map based on its kind.
func (t *TemplateSet) categorize(kind string, style TemplateStyle) {
	if strings.HasSuffix(kind, "_boundary") {
		t.boundaries[kind] = style
	} else {
		t.elements[kind] = style
	}
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
