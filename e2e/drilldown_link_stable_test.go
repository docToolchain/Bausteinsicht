package e2e

// TestDrillDownLinkIsStableAcrossSyncs covers #661: when several views share a
// scope, the drill-down link of the scope element used to depend on Go's map
// iteration order, so it flipped between views from one sync to the next.

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/beevik/etree"
)

// scopeLinks returns, per page name, the link of the element with the given
// model ID.
func scopeLinks(t *testing.T, drawioPath, elementID string) map[string]string {
	t.Helper()
	data, err := os.ReadFile(drawioPath)
	if err != nil {
		t.Fatal(err)
	}
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(data); err != nil {
		t.Fatal(err)
	}
	links := map[string]string{}
	for _, page := range doc.FindElements("//diagram") {
		for _, obj := range page.FindElements(".//object") {
			if obj.SelectAttrValue("bausteinsicht_id", "") == elementID {
				links[page.SelectAttrValue("name", "")] = obj.SelectAttrValue("link", "")
			}
		}
	}
	return links
}

func TestDrillDownLinkIsStableAcrossSyncs(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")
	// Two more views with the same scope as the sample's container view, each
	// showing less of its subtree.
	runCLI(t, bin, dir, "add", "view", "zz-api", "--scope", "onlineshop", "--include", "onlineshop.api", "--title", "ZZ API")
	runCLI(t, bin, dir, "add", "view", "aa-db", "--scope", "onlineshop", "--include", "onlineshop.cache", "--title", "AA DB")
	runCLI(t, bin, dir, "sync")

	drawioPath := filepath.Join(dir, "architecture.drawio")
	first := scopeLinks(t, drawioPath, "onlineshop")
	if len(first) == 0 {
		t.Fatal("no page shows the scope element")
	}
	var distinct []string
	seen := map[string]bool{}
	for _, l := range first {
		if !seen[l] {
			seen[l] = true
			distinct = append(distinct, l)
		}
	}
	sort.Strings(distinct)
	if len(distinct) != 1 || !strings.HasSuffix(distinct[0], "view-containers") {
		t.Fatalf("every page must link the scope element to the overview view-containers, got %v", distinct)
	}

	for i := 0; i < 8; i++ {
		runCLI(t, bin, dir, "sync")
		again := scopeLinks(t, drawioPath, "onlineshop")
		for page, link := range first {
			if again[page] != link {
				t.Fatalf("sync %d: the link on page %q changed from %q to %q", i+2, page, link, again[page])
			}
		}
	}
}
