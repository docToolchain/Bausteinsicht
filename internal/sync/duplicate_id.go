package sync

import (
	"fmt"

	"github.com/docToolchain/Bausteinsicht/internal/drawio"
)

// stripDuplicateElementIDs finds element cells that repeat a bausteinsicht_id
// already used on the same page. Copying a shape in draw.io copies its
// attributes, so the copy claims to be the element; the first cell on the page
// stays the element, the copy loses the attribute and is an unmanaged shape
// from then on (the usual handling of shapes drawn in draw.io). One warning
// per copy names the ID, the page and the cell (#722).
func stripDuplicateElementIDs(doc *drawio.Document) []string {
	var warnings []string
	for _, page := range doc.Pages() {
		seen := make(map[string]bool)
		for _, obj := range page.FindAllElements() {
			id := obj.SelectAttrValue(attrBausteinsichtID, "")
			if !seen[id] {
				seen[id] = true
				continue
			}
			obj.RemoveAttr(attrBausteinsichtID)
			warnings = append(warnings, fmt.Sprintf(
				"duplicate bausteinsicht_id %q on page %q: the copy (cell %q) is now an unmanaged shape",
				id, page.Name(), obj.SelectAttrValue("id", "")))
		}
	}
	return warnings
}
