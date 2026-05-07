package overlay

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/beevik/etree"
)

const OriginalFillAttr = "data-original-fill"

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
	for _, page := range root.FindElements(".//mxGraphModel/root/mxCell") {
		elementID := page.SelectAttrValue("id", "")
		if elementID == "" || elementID == "0" || elementID == "1" {
			continue
		}

		if normVal, ok := normalized[elementID]; ok {
			applyColor(page, normVal, scheme)
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
	for _, cell := range root.FindElements(".//mxGraphModel/root/mxCell") {
		originalFill := cell.SelectAttrValue(OriginalFillAttr, "")
		if originalFill != "" {
			geometry := cell.FindElement("mxGeometry")
			if geometry != nil {
				style := geometry.SelectAttrValue("style", "")
				if style != "" {
					style = updateStyleFill(style, originalFill)
					geometry.CreateAttr("style", style)
				}
			}
			cell.RemoveAttr(OriginalFillAttr)
		}
	}

	if err := doc.WriteToFile(drawioPath); err != nil {
		return fmt.Errorf("writing draw.io file: %w", err)
	}
	return nil
}

func applyColor(cell *etree.Element, normalized float64, scheme ColorScheme) {
	color := ColorForValue(normalized, scheme)

	geometry := cell.FindElement("mxGeometry")
	if geometry == nil {
		return
	}

	style := geometry.SelectAttrValue("style", "")
	originalFill := geometry.SelectAttrValue("fillColor", "")

	if originalFill == "" {
		originalFill = "#ffffff"
	}
	if cell.SelectAttrValue(OriginalFillAttr, "") == "" {
		cell.CreateAttr(OriginalFillAttr, originalFill)
	}

	style = updateStyleFill(style, color)
	geometry.CreateAttr("style", style)
}

func updateStyleFill(style, color string) string {
	if style == "" {
		return "fillColor=" + color
	}

	result := ""
	hasKey := false
	for _, part := range parseStyleParts(style) {
		if len(part) > 0 && startsWithKey(part, "fillColor") {
			result += "fillColor=" + color + ";"
			hasKey = true
		} else {
			if part != "" {
				result += part + ";"
			}
		}
	}
	if !hasKey {
		result += "fillColor=" + color + ";"
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
