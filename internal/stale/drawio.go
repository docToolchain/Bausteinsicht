package stale

import (
	"fmt"
	"regexp"

	"github.com/beevik/etree"
	"github.com/docToolchain/Bausteinsicht/internal/drawio"
)

var staleStyleRe = regexp.MustCompile(`(fillColor|strokeColor|strokeWidth)=[^;]*;?`)

// MarkInDrawio adds visual indicators to stale elements in a draw.io diagram.
// Changes fill color and stroke to indicate staleness, with risk-level color coding.
func MarkInDrawio(staleElements []StaleElement, drawioPath string) error {
	// Load draw.io document
	doc, err := drawio.LoadDocument(drawioPath)
	if err != nil {
		return fmt.Errorf("loading draw.io document: %w", err)
	}

	// Get the first diagram page
	pages := doc.Pages()
	if len(pages) == 0 {
		return fmt.Errorf("no diagram page found in %s", drawioPath)
	}
	page := pages[0]

	// Get the root mxGraphModel element
	root := page.Root()
	if root == nil {
		return fmt.Errorf("no root element in page")
	}

	// Build ID to element map for quick lookup.
	// Elements are stored as <object bausteinsicht_id="..."> wrapping an <mxCell>.
	idMap := make(map[string]*etree.Element)
	for _, obj := range root.SelectElements("object") {
		if bausteinsichtID := obj.SelectAttr("bausteinsicht_id"); bausteinsichtID != nil {
			idMap[bausteinsichtID.Value] = obj
		}
	}

	// Mark each stale element
	for _, staleElem := range staleElements {
		elem, exists := idMap[staleElem.ID]
		if !exists {
			continue // Element not in diagram
		}

		// Add style properties for visual indication
		markStaleElement(elem, staleElem)
	}

	// Save modified document
	if err := drawio.SaveDocument(drawioPath, doc); err != nil {
		return fmt.Errorf("saving draw.io document: %w", err)
	}

	return nil
}

// markStaleElement modifies the draw.io element to mark it as stale.
// obj is the <object> element; style changes go on its <mxCell> child.
func markStaleElement(obj *etree.Element, staleElem StaleElement) {
	cell := obj.FindElement("mxCell")
	if cell == nil {
		return
	}

	riskColor := riskColor(staleElem.Risk)
	if riskColor == "" {
		riskColor = "#CCCCCC"
	}

	// Get current style from mxCell; strip previous stale marker properties
	// so re-runs are idempotent and don't accumulate strokeWidth entries.
	style := cell.SelectAttrValue("style", "")
	style = staleStyleRe.ReplaceAllString(style, "")
	style = fmt.Sprintf("%s;fillColor=%s;strokeColor=%s;strokeWidth=2", style, riskColor, riskColor)

	cell.CreateAttr("style", style)

	// Tooltip lives on the <object> element (consistent with how element.go places description).
	tooltip := fmt.Sprintf("⚠ STALE\nLast modified: %s\nNo status set\nNo ADR linked",
		staleElem.LastModified.Format("2006-01-02"))
	obj.CreateAttr("tooltip", tooltip)
}

// riskColor returns a color code for the risk level.
func riskColor(risk RiskLevel) string {
	switch risk {
	case RiskHigh:
		return "#FF6666" // Light red
	case RiskMedium:
		return "#FFBB66" // Light orange
	case RiskLow:
		return "#66DD66" // Light green
	default:
		return "#CCCCCC" // Grey
	}
}
