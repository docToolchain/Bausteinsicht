package e2e

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// #718: long element titles were clipped ("Customer Relatio"). After sync and
// layout the title cell wraps and has room for all lines.
func TestLongTitlesWrapAndGrowTheShape(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "m.jsonc")
	if err := os.WriteFile(path, []byte(`{"specification":{"elements":{"actor":{"notation":"Actor"},"system":{"notation":"System"}},"relationships":{"uses":{"notation":"uses"}}},
 "model":{"u":{"kind":"actor","title":"Customer Relationship Management Back-Office Operator"},"s":{"kind":"system","title":"Core"}},
 "relationships":[{"from":"u","to":"s","kind":"uses","label":"submits requests"}],
 "views":{"v":{"title":"V","include":["*"]}}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	check := func(step string) {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(dir, "architecture.drawio"))
		if err != nil {
			t.Fatal(err)
		}
		cell := regexp.MustCompile(`<mxCell id="[^"]*u-title"[^>]*>\s*<mxGeometry[^>]*>`).FindString(string(data))
		if cell == "" {
			t.Fatalf("%s: no title cell of u:\n%s", step, data)
		}
		if !strings.Contains(cell, "whiteSpace=wrap") {
			t.Errorf("%s: the title cell does not wrap: %s", step, cell)
		}
		h := regexp.MustCompile(`height="([0-9.]+)"`).FindStringSubmatch(cell)
		if height, _ := strconv.ParseFloat(h[1], 64); height < 3*13 {
			t.Errorf("%s: title height %v cannot hold three lines", step, height)
		}
	}
	runCLI(t, bin, dir, "--model", path, "sync")
	check("after sync")
	runCLI(t, bin, dir, "--model", path, "layout")
	check("after layout")
}
