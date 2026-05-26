package layout

import (
	"fmt"

	"github.com/beevik/etree"
	"github.com/docToolchain/Bausteinsicht/internal/drawio"
)

// Apply applies layout positions to draw.io diagram.
// Reads pinned status from existing draw.io elements and respects PreservePinned setting.
func Apply(doc *drawio.Document, result LayoutResult, preservePinned bool) error {
	if len(result.Positions) == 0 {
		return fmt.Errorf("layout result is empty")
	}

	// Build map of existing draw.io elements with their pinned status
	pinnedMap := readPinnedStatus(doc)

	// For each computed position, update the draw.io element
	for elemID, pos := range result.Positions {
		// Skip pinned elements if preservePinned is enabled
		if preservePinned && pinnedMap[elemID] {
			continue
		}

		// Find and update element in draw.io
		if err := updateElementPosition(doc, elemID, pos); err != nil {
			continue
		}
	}

	return nil
}

// readPinnedStatus reads the bausteinsicht-pinned property from draw.io elements.
func readPinnedStatus(doc *drawio.Document) map[string]bool {
	pinned := make(map[string]bool)

	for _, page := range doc.Pages() {
		if root := page.Root(); root != nil {
			walkElements(root, func(elem *etree.Element) {
				if id, ok := getAttr(elem, "bausteinsicht_id"); ok {
					if pinValue, ok := getAttr(elem, "bausteinsicht-pinned"); ok && pinValue == "true" {
						pinned[id] = true
					}
				}
			})
		}
	}

	return pinned
}

// updateElementPosition updates the x, y, width, height of a draw.io element.
func updateElementPosition(doc *drawio.Document, elemID string, pos ElementPosition) error {
	for _, page := range doc.Pages() {
		if root := page.Root(); root != nil {
			found := false
			walkElements(root, func(elem *etree.Element) {
				if found {
					return
				}
				if id, ok := getAttr(elem, "bausteinsicht_id"); ok && id == elemID {
					// Find mxGeometry child and update coordinates
					for _, child := range elem.ChildElements() {
						if child.Tag == "mxGeometry" {
							child.CreateAttr("x", fmt.Sprintf("%.0f", pos.X))
							child.CreateAttr("y", fmt.Sprintf("%.0f", pos.Y))
							child.CreateAttr("width", fmt.Sprintf("%.0f", pos.Width))
							child.CreateAttr("height", fmt.Sprintf("%.0f", pos.Height))
							found = true
							break
						}
					}
				}
			})
			if found {
				return nil
			}
		}
	}

	return fmt.Errorf("element %s not found in diagram", elemID)
}

// walkElements recursively walks through all elements in the tree.
func walkElements(elem *etree.Element, fn func(*etree.Element)) {
	fn(elem)
	for _, child := range elem.ChildElements() {
		walkElements(child, fn)
	}
}

// getAttr extracts attribute value from element safely.
func getAttr(elem *etree.Element, name string) (string, bool) {
	for _, attr := range elem.Attr {
		if attr.Key == name {
			return attr.Value, true
		}
	}
	return "", false
}
