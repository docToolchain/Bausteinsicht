package e2e

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const pageOrderModel = `{"specification":{"elements":{"actor":{"notation":"Actor"}}},
 "model":{"u":{"kind":"actor","title":"U"}},
 "views":{"ctx":{"title":"C","include":["*"]},"inner":{"title":"I","include":["*"]},"deep":{"title":"D","include":["*"]},"empty":{"title":"E","include":["*"]},"excl":{"title":"X","include":["*"]}}}`

// TestSyncWritesViewPagesInAFixedOrder (#683): a fresh sync wrote the view pages
// in Go's random map order, so draw.io opened on a random page and committed
// .drawio files changed on every regeneration.
func TestSyncWritesViewPagesInAFixedOrder(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "m.jsonc"), pageOrderModel)
	re := regexp.MustCompile(`<diagram[^>]* id="view-([a-z]+)"`)

	var first string
	for run := 1; run <= 6; run++ {
		_ = os.Remove(filepath.Join(dir, "architecture.drawio"))
		_ = os.Remove(filepath.Join(dir, ".bausteinsicht-sync"))
		runCLI(t, bin, dir, "--model", "m.jsonc", "sync")
		data, err := os.ReadFile(filepath.Join(dir, "architecture.drawio"))
		if err != nil {
			t.Fatal(err)
		}
		var ids []string
		for _, m := range re.FindAllStringSubmatch(string(data), -1) {
			ids = append(ids, m[1])
		}
		got := strings.Join(ids, " ")
		if run == 1 {
			first = got
			if got != "ctx deep empty excl inner" {
				t.Errorf("run 1: pages %q, want the keys in alphabetical order", got)
			}
			continue
		}
		if got != first {
			t.Errorf("run %d: pages %q differ from run 1: %q", run, got, first)
		}
	}
}
