package stale

import (
	"fmt"
	"strings"

	"github.com/beevik/etree"
	"github.com/docToolchain/Bausteinsicht/internal/drawio"
	"github.com/docToolchain/Bausteinsicht/internal/overlay"
)

const (
	originalStrokeColorAttr = "data-original-stroke-color"
	originalStrokeWidthAttr = "data-original-stroke-width"
	originalTooltipAttr     = "data-original-tooltip"
)

// MarkInDrawio adds visual stale indicators to elements across all diagram pages.
// Original fill, stroke, and tooltip are preserved in data attributes
// so the marking is non-destructive and can be reversed with UnmarkInDrawio.
func MarkInDrawio(staleElements []StaleElement, drawioPath string) error {
	doc, err := drawio.LoadDocument(drawioPath)
	if err != nil {
		return fmt.Errorf("loading draw.io document: %w", err)
	}

	pages := doc.Pages()
	if len(pages) == 0 {
		return fmt.Errorf("no diagram page found in %s", drawioPath)
	}

	for _, page := range pages {
		root := page.Root()
		if root == nil {
			continue
		}

		idMap := make(map[string]*etree.Element)
		for _, obj := range root.SelectElements("object") {
			if bausteinsichtID := obj.SelectAttr("bausteinsicht_id"); bausteinsichtID != nil {
				idMap[bausteinsichtID.Value] = obj
			}
		}

		for _, staleElem := range staleElements {
			obj, exists := idMap[staleElem.ID]
			if !exists {
				continue
			}
			markStaleElement(obj, staleElem)
		}
	}

	if err := drawio.SaveDocument(drawioPath, doc); err != nil {
		return fmt.Errorf("saving draw.io document: %w", err)
	}
	return nil
}

// UnmarkInDrawio removes stale visual indicators from all diagram pages,
// restoring each element's original fill color from overlay.OriginalFillAttr.
// Returns the number of elements that were actually unmarked.
func UnmarkInDrawio(drawioPath string) (int, error) {
	doc, err := drawio.LoadDocument(drawioPath)
	if err != nil {
		return 0, fmt.Errorf("loading draw.io document: %w", err)
	}

	count := 0
	for _, page := range doc.Pages() {
		root := page.Root()
		if root == nil {
			continue
		}
		for _, obj := range root.SelectElements("object") {
			cell := obj.FindElement("mxCell")
			if cell == nil {
				continue
			}
			originalFillAttr := cell.SelectAttr(overlay.OriginalFillAttr)
			if originalFillAttr == nil {
				continue
			}
			style := cell.SelectAttrValue("style", "")
			if originalFillAttr.Value == "" {
				// fillColor was originally absent — remove it instead of restoring a value.
				style = removeStyleProperties(style, []string{"fillColor"})
			} else {
				style = setStyleProperty(style, "fillColor", originalFillAttr.Value)
			}

			// Restore original stroke color: if it was absent originally, remove the key.
			originalStroke := cell.SelectAttrValue(originalStrokeColorAttr, "")
			if originalStroke != "" {
				style = setStyleProperty(style, "strokeColor", originalStroke)
			} else {
				style = removeStyleProperties(style, []string{"strokeColor"})
			}

			// Restore original stroke width: if it was absent originally, remove the key.
			originalWidth := cell.SelectAttrValue(originalStrokeWidthAttr, "")
			if originalWidth != "" {
				style = setStyleProperty(style, "strokeWidth", originalWidth)
			} else {
				style = removeStyleProperties(style, []string{"strokeWidth"})
			}

			cell.CreateAttr("style", style)
			cell.RemoveAttr(overlay.OriginalFillAttr)
			cell.RemoveAttr(originalStrokeColorAttr)
			cell.RemoveAttr(originalStrokeWidthAttr)

			// Restore original tooltip (remove the stale tooltip if none was set before).
			originalTooltip := obj.SelectAttrValue(originalTooltipAttr, "")
			obj.RemoveAttr("tooltip")
			obj.RemoveAttr(originalTooltipAttr)
			if originalTooltip != "" {
				obj.CreateAttr("tooltip", originalTooltip)
			}
			count++
		}
	}

	if err := drawio.SaveDocument(drawioPath, doc); err != nil {
		return 0, fmt.Errorf("saving draw.io document: %w", err)
	}
	return count, nil
}

// markStaleElement applies a risk-color fill to the mxCell of an <object> element.
// On the first call, the original fillColor is saved in overlay.OriginalFillAttr so
// subsequent runs remain idempotent and the change is reversible via UnmarkInDrawio.
func markStaleElement(obj *etree.Element, staleElem StaleElement) {
	cell := obj.FindElement("mxCell")
	if cell == nil {
		return
	}

	color := riskColor(staleElem.Risk)

	style := cell.SelectAttrValue("style", "")

	// Preserve original styling only on the first marking pass (idempotent).
	// Store "" when fillColor was originally absent so UnmarkInDrawio can remove it.
	if cell.SelectAttr(overlay.OriginalFillAttr) == nil {
		originalFill := extractStyleProperty(style, "fillColor")
		cell.CreateAttr(overlay.OriginalFillAttr, originalFill)
		cell.CreateAttr(originalStrokeColorAttr, extractStyleProperty(style, "strokeColor"))
		cell.CreateAttr(originalStrokeWidthAttr, extractStyleProperty(style, "strokeWidth"))
		obj.CreateAttr(originalTooltipAttr, obj.SelectAttrValue("tooltip", ""))
	}

	style = setStyleProperty(style, "fillColor", color)
	style = setStyleProperty(style, "strokeColor", color)
	style = setStyleProperty(style, "strokeWidth", "2")
	cell.CreateAttr("style", style)

	tooltip := fmt.Sprintf("⚠ STALE\nLast modified: %s\nNo status set\nNo ADR linked",
		staleElem.LastModified.Format("2006-01-02"))
	obj.CreateAttr("tooltip", tooltip)
}

// riskColor returns the fill color for a given risk level.
func riskColor(risk RiskLevel) string {
	switch risk {
	case RiskHigh:
		return "#FF6666"
	case RiskMedium:
		return "#FFBB66"
	case RiskLow:
		return "#66DD66"
	default:
		return "#CCCCCC"
	}
}

// extractStyleProperty returns the value of key in a draw.io semicolon-separated
// style string, or "" if the key is absent.
func extractStyleProperty(style, key string) string {
	for _, part := range splitStyleParts(style) {
		if eqIdx := strings.IndexByte(part, '='); eqIdx > 0 {
			if part[:eqIdx] == key {
				return part[eqIdx+1:]
			}
		}
	}
	return ""
}

// setStyleProperty updates or inserts key=value in a draw.io style string.
func setStyleProperty(style, key, value string) string {
	parts := splitStyleParts(style)
	found := false
	var result []string
	for _, part := range parts {
		if eqIdx := strings.IndexByte(part, '='); eqIdx > 0 && part[:eqIdx] == key {
			result = append(result, key+"="+value)
			found = true
		} else if part != "" {
			result = append(result, part)
		}
	}
	if !found {
		result = append(result, key+"="+value)
	}
	if len(result) == 0 {
		return ""
	}
	return strings.Join(result, ";") + ";"
}

// removeStyleProperties strips the given keys from a draw.io style string.
func removeStyleProperties(style string, keys []string) string {
	keySet := make(map[string]bool, len(keys))
	for _, k := range keys {
		keySet[k] = true
	}
	parts := splitStyleParts(style)
	var result []string
	for _, part := range parts {
		if eqIdx := strings.IndexByte(part, '='); eqIdx > 0 {
			if !keySet[part[:eqIdx]] {
				result = append(result, part)
			}
		} else if part != "" {
			result = append(result, part)
		}
	}
	if len(result) == 0 {
		return ""
	}
	return strings.Join(result, ";") + ";"
}

// splitStyleParts splits a draw.io style string on semicolons, trimming empty parts.
func splitStyleParts(style string) []string {
	raw := strings.Split(strings.TrimRight(style, ";"), ";")
	result := raw[:0]
	for _, p := range raw {
		if p = strings.TrimSpace(p); p != "" {
			result = append(result, p)
		}
	}
	return result
}
