package overlay

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/beevik/etree"
)

const (
	OriginalFillAttr = "data-original-fill"
	styleAttr        = "style"
	fillColorAttr    = "fillColor"
	fillColorPrefix  = "fillColor="
)

func LoadMetricsFile(path string) (*MetricsFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading metrics file: %w", err)
	}
	var mf MetricsFile
	if err := json.Unmarshal(data, &mf); err != nil {
		return nil, fmt.Errorf("parsing metrics file: %w", err)
	}
	return &mf, nil
}

func Apply(drawioPath string, metrics *MetricsFile, metricKey string, scheme ColorScheme) error {
	doc := etree.NewDocument()
	if err := doc.ReadFromFile(drawioPath); err != nil {
		return fmt.Errorf("reading draw.io file: %w", err)
	}

	extracted, err := ExtractMetric(metrics.Metrics, metricKey)
	if err != nil {
		return fmt.Errorf("extracting metric %q: %w", metricKey, err)
	}

	if len(extracted) == 0 {
		return fmt.Errorf("no elements found for metric %q", metricKey)
	}

	higherIsBetter := IsMetricBetter(metricKey)
	normalized := Normalize(extracted, higherIsBetter)

	root := doc.Root()

	// Bare mxCell children (hand-crafted or connector elements)
	for _, cell := range root.FindElements(".//mxGraphModel/root/mxCell") {
		elementID := cell.SelectAttrValue("id", "")
		if elementID == "" || elementID == "0" || elementID == "1" {
			continue
		}
		if normVal, ok := normalized[elementID]; ok {
			applyColor(cell, normVal, scheme)
		}
	}

	// <object>-wrapped elements (bausteinsicht sync output).
	// Match on bausteinsicht_id so metrics.json uses model IDs, not page-scoped draw.io IDs.
	for _, obj := range root.FindElements(".//mxGraphModel/root/object") {
		bsID := obj.SelectAttrValue("bausteinsicht_id", "")
		if bsID == "" {
			continue
		}
		if normVal, ok := normalized[bsID]; ok {
			inner := obj.FindElement("mxCell")
			if inner != nil {
				applyColor(inner, normVal, scheme)
			}
		}
	}

	if err := doc.WriteToFile(drawioPath); err != nil {
		return fmt.Errorf("writing draw.io file: %w", err)
	}
	return nil
}

func Remove(drawioPath string) error {
	doc := etree.NewDocument()
	if err := doc.ReadFromFile(drawioPath); err != nil {
		return fmt.Errorf("reading draw.io file: %w", err)
	}

	root := doc.Root()

	// Collect all candidate cells: direct mxCell children and mxCell inside <object> wrappers.
	cells := root.FindElements(".//mxGraphModel/root/mxCell")
	for _, obj := range root.FindElements(".//mxGraphModel/root/object") {
		if inner := obj.FindElement("mxCell"); inner != nil {
			cells = append(cells, inner)
		}
	}

	for _, cell := range cells {
		originalFill := cell.SelectAttrValue(OriginalFillAttr, "")
		if originalFill == "" {
			continue
		}
		style := cell.SelectAttrValue(styleAttr, "")
		style = updateStyleFill(style, originalFill)
		cell.CreateAttr(styleAttr, style)
		cell.RemoveAttr(OriginalFillAttr)
	}

	if err := doc.WriteToFile(drawioPath); err != nil {
		return fmt.Errorf("writing draw.io file: %w", err)
	}
	return nil
}

// applyColor sets a heatmap fill color on a draw.io mxCell element.
// Style and fillColor live on the mxCell itself (not on its mxGeometry child).
func applyColor(cell *etree.Element, normalized float64, scheme ColorScheme) {
	color := ColorForValue(normalized, scheme)

	style := cell.SelectAttrValue(styleAttr, "")
	originalFill := extractFillColor(style)
	if originalFill == "" {
		originalFill = "#ffffff"
	}

	if cell.SelectAttrValue(OriginalFillAttr, "") == "" {
		cell.CreateAttr(OriginalFillAttr, originalFill)
	}

	style = updateStyleFill(style, color)
	cell.CreateAttr(styleAttr, style)
}

// extractFillColor parses a draw.io style string and returns the fillColor value.
func extractFillColor(style string) string {
	for _, part := range parseStyleParts(style) {
		if startsWithKey(part, fillColorAttr) && len(part) > len(fillColorAttr)+1 {
			return part[len(fillColorAttr)+1:]
		}
	}
	return ""
}

func updateStyleFill(style, color string) string {
	if style == "" {
		return fillColorPrefix + color
	}

	result := ""
	hasKey := false
	for _, part := range parseStyleParts(style) {
		if len(part) > 0 && startsWithKey(part, fillColorAttr) {
			result += fillColorPrefix + color + ";"
			hasKey = true
		} else {
			if part != "" {
				result += part + ";"
			}
		}
	}
	if !hasKey {
		result += fillColorPrefix + color + ";"
	}
	return result
}

func parseStyleParts(style string) []string {
	var result []string
	var current string
	for _, ch := range style {
		if ch == ';' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func startsWithKey(part, key string) bool {
	return len(part) >= len(key) && part[:len(key)] == key
}
