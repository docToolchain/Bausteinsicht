package stale

import (
	"fmt"
	"strings"

	"github.com/beevik/etree"
	"github.com/docToolchain/Bausteinsicht/internal/drawio"
	"github.com/docToolchain/Bausteinsicht/internal/overlay"
)

// MarkInDrawio adds visual stale indicators to elements across all diagram pages.
// Original fill colors are preserved in a data attribute (overlay.OriginalFillAttr)
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
func UnmarkInDrawio(drawioPath string) error {
	doc, err := drawio.LoadDocument(drawioPath)
	if err != nil {
		return fmt.Errorf("loading draw.io document: %w", err)
	}

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
			originalFill := cell.SelectAttrValue(overlay.OriginalFillAttr, "")
			if originalFill == "" {
				continue
			}
			style := cell.SelectAttrValue("style", "")
			style = setStyleProperty(style, "fillColor", originalFill)
			style = removeStyleProperties(style, []string{"strokeColor", "strokeWidth"})
			cell.CreateAttr("style", style)
			cell.RemoveAttr(overlay.OriginalFillAttr)
			obj.RemoveAttr("tooltip")
		}
	}

	if err := drawio.SaveDocument(drawioPath, doc); err != nil {
		return fmt.Errorf("saving draw.io document: %w", err)
	}
	return nil
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

	// Preserve the original fill only on the first marking pass.
	if cell.SelectAttrValue(overlay.OriginalFillAttr, "") == "" {
		originalFill := extractStyleProperty(style, "fillColor")
		if originalFill == "" {
			originalFill = "#ffffff"
		}
		cell.CreateAttr(overlay.OriginalFillAttr, originalFill)
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
