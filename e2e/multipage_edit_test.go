package e2e

import (
	"os"
	"strings"
	"testing"

	"github.com/beevik/etree"
)

const multiPageModel = `{
  "specification": { "elements": { "system": { "notation": "System", "container": true }, "db": { "notation": "DB" } },
    "relationships": { "uses": { "notation": "uses" } } },
  "model": { "shop": { "kind": "system", "title": "Shop", "children": { "db": { "kind": "db", "title": "DB" } } },
             "cust": { "kind": "db", "title": "Cust" } },
  "relationships": [ { "from": "cust", "to": "shop.db", "kind": "uses", "label": "reads" } ],
  "views": {
    "ctx":   { "title": "C", "include": ["*"] },
    "inner": { "title": "I", "scope": "shop", "include": ["shop.*"] },
    "excl":  { "title": "X", "include": ["**"] }
  }
}`

// editShopTitleOnPage changes the title of "shop" on one page only: the text
// cell <page>--shop-title where the element is drawn as a boundary or with
// sub-cells, or the HTML label of the object on a scoped page.
func editShopTitleOnPage(t *testing.T, path, pageID, newTitle string) bool {
	t.Helper()
	doc := etree.NewDocument()
	if err := doc.ReadFromFile(path); err != nil {
		t.Fatal(err)
	}
	changed := false
	for _, diagram := range doc.FindElements("//diagram") {
		if diagram.SelectAttrValue("id", "") != pageID {
			continue
		}
		for _, cell := range diagram.FindElements(".//mxCell") {
			if strings.HasSuffix(cell.SelectAttrValue("id", ""), "--shop-title") {
				cell.CreateAttr("value", newTitle)
				changed = true
			}
		}
		for _, obj := range diagram.FindElements(".//object[@bausteinsicht_id='shop']") {
			if label := obj.SelectAttrValue("label", ""); strings.Contains(label, "Shop") {
				obj.CreateAttr("label", strings.Replace(label, "Shop", newTitle, 1))
				changed = true
			}
		}
	}
	if err := doc.WriteToFile(path); err != nil {
		t.Fatal(err)
	}
	return changed
}

// TestSyncReverseTitleEditOnAnyPage (#684): an element that is drawn on several
// pages (here "shop" on three) can be edited on any of them. Before, only the
// first page of the file counted, so the edit was silently overwritten on some
// pages and, because the page order changed from run to run (#683), on some runs.
func TestSyncReverseTitleEditOnAnyPage(t *testing.T) {
	bin := buildBinary(t)
	for _, page := range []string{"view-ctx", "view-inner", "view-excl"} {
		t.Run(page, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, dir+"/architecture.jsonc", multiPageModel)
			runCLI(t, bin, dir, "sync")
			if !editShopTitleOnPage(t, dir+"/architecture.drawio", page, "Shop EDIT") {
				t.Fatalf("setup: the title of shop was not found on %s", page)
			}

			out := runCLI(t, bin, dir, "sync")
			data, err := os.ReadFile(dir + "/architecture.jsonc")
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(data), `"title": "Shop EDIT"`) && !strings.Contains(string(data), `"title":"Shop EDIT"`) {
				t.Fatalf("the edit on %s did not reach the model:\n%s\n%s", page, out, data)
			}
			// The other pages follow the model with the next sync.
			runCLI(t, bin, dir, "sync")
			if out := runCLI(t, bin, dir, "sync"); !strings.Contains(out, "Already in sync") {
				t.Errorf("the model and all pages must converge:\n%s", out)
			}
		})
	}
}
