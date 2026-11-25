package e2e

// TestLayoutLaysOutEveryPage covers #654 through the real CLI on the sample
// project of `init`: every page is laid out, elements do not overlap, elements
// inside a boundary stay inside it (relative coordinates), and running the
// command twice gives the same file.

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/beevik/etree"
)

type cellBox struct {
	id, parent string
	x, y, w, h float64
}

func readPageBoxes(t *testing.T, page *etree.Element) []cellBox {
	t.Helper()
	var boxes []cellBox
	for _, obj := range page.FindElements(".//object") {
		if obj.SelectAttr("bausteinsicht_id") == nil {
			continue
		}
		cell := obj.SelectElement("mxCell")
		geom := cell.SelectElement("mxGeometry")
		num := func(name string) float64 {
			v, err := strconv.ParseFloat(geom.SelectAttrValue(name, "0"), 64)
			if err != nil {
				t.Fatalf("bad %s on %s", name, obj.SelectAttrValue("id", ""))
			}
			return v
		}
		boxes = append(boxes, cellBox{obj.SelectAttrValue("id", ""), cell.SelectAttrValue("parent", ""), num("x"), num("y"), num("width"), num("height")})
	}
	return boxes
}

func TestLayoutLaysOutEveryPage(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")
	runCLI(t, bin, dir, "sync")

	out := runCLI(t, bin, dir, "layout", "--rank-dir", "LR")
	if !strings.Contains(out, "page(s)") {
		t.Errorf("expected a summary, got:\n%s", out)
	}

	drawioPath := filepath.Join(dir, "architecture.drawio")
	first, err := os.ReadFile(drawioPath)
	if err != nil {
		t.Fatal(err)
	}
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(first); err != nil {
		t.Fatal(err)
	}

	pages := doc.FindElements("//diagram")
	if len(pages) < 2 {
		t.Fatalf("the sample project should have several pages, got %d", len(pages))
	}
	nested := 0
	for _, page := range pages {
		boxes := readPageBoxes(t, page)
		byID := map[string]cellBox{}
		for _, b := range boxes {
			byID[b.id] = b
		}
		for i, a := range boxes {
			if a.parent != "1" {
				nested++
				p, ok := byID[a.parent]
				if !ok {
					continue
				}
				if a.x < 0 || a.y < 0 || a.x+a.w > p.w || a.y+a.h > p.h {
					t.Errorf("page %q: %s (%v,%v %vx%v) is outside its boundary %s (%vx%v)",
						page.SelectAttrValue("name", ""), a.id, a.x, a.y, a.w, a.h, p.id, p.w, p.h)
				}
			}
			for _, b := range boxes[i+1:] {
				if a.parent != b.parent {
					continue
				}
				if a.x < b.x+b.w && b.x < a.x+a.w && a.y < b.y+b.h && b.y < a.y+a.h {
					t.Errorf("page %q: %s and %s overlap", page.SelectAttrValue("name", ""), a.id, b.id)
				}
			}
		}
	}
	if nested == 0 {
		t.Error("the sample project should have elements inside a boundary; the test would not check #654's relative coordinates")
	}

	runCLI(t, bin, dir, "layout", "--rank-dir", "LR")
	second, _ := os.ReadFile(drawioPath)
	if string(first) != string(second) {
		t.Error("running layout twice changed the diagram: the order is not deterministic")
	}

	if out, code := runCLIAllowFail(t, bin, dir, "layout", "--view", "no-such-view"); code == 0 || !strings.Contains(out, "not found") {
		t.Errorf("unknown view: exit %d\n%s", code, out)
	}
}
