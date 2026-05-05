package stale

import (
	"fmt"
	"regexp"

	"github.com/beevik/etree"
	"github.com/docToolchain/Bausteinsicht/internal/drawio"
)

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

	// Build ID to element map for quick lookup
	idMap := make(map[string]*etree.Element)
	for _, cell := range root.SelectElements("mxCell") {
		if bausteinsichtID := cell.SelectAttr("bausteinsicht_id"); bausteinsichtID != nil {
			idMap[bausteinsichtID.Value] = cell
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

// markStaleElement modifies the style attribute to mark stale elements.
func markStaleElement(elem *etree.Element, staleElem StaleElement) {
	// Get current style
	styleAttr := elem.SelectAttr("style")
	var style string
	if styleAttr != nil {
		style = styleAttr.Value
	}

	// Add grey fill and risk-based stroke color
	riskColor := riskColor(staleElem.Risk)
	if riskColor == "" {
		riskColor = "#CCCCCC" // Default grey
	}

	// Append or update style properties
	if style == "" {
		style = fmt.Sprintf("fillColor=%s;strokeColor=%s;strokeWidth=2", riskColor, riskColor)
	} else {
		// Remove existing fillColor and strokeColor if present
		re := regexp.MustCompile(`(fillColor|strokeColor)=[^;]*;?`)
		style = re.ReplaceAllString(style, "")
		style = fmt.Sprintf("%s;fillColor=%s;strokeColor=%s;strokeWidth=2", style, riskColor, riskColor)
	}

	elem.CreateAttr("style", style)

	// Add tooltip with staleness info
	tooltip := fmt.Sprintf("⚠ STALE\\nLast modified: %s\\nNo status set\\nNo ADR linked",
		staleElem.LastModified.Format("2006-01-02"))
	elem.CreateAttr("tooltip", tooltip)
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
