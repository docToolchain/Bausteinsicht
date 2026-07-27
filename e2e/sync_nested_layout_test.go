package e2e

// TestSyncNestedLayout verifies the "nested" view layout end-to-end: running
// the real `sync` command must write draw.io XML in which the model's container
// nesting is reproduced as actual mxCell containment — intermediate container
// cells re-parented under the scope, leaf cells re-parented under their
// immediate container, and the container geometry resized to fit its content.
//
// The view deliberately includes only the deep leaf IDs and omits their
// intermediate container (a hand-written, non-wildcard include list). This also
// exercises the robustness fix (#571 review): the nested layout closes the
// resolved set under the missing ancestor containers, so the container is still
// placed and the leaves are not silently dropped.

import (
	"path/filepath"
	"strconv"
	"testing"

	"github.com/beevik/etree"
	"github.com/docToolchain/Bausteinsicht/internal/drawio"
)

const nestedLayoutModel = `{
  "specification": {
    "elements": {
      "context": { "notation": "Context", "container": true },
      "zone":    { "notation": "Zone",    "container": true },
      "part":    { "notation": "Part" }
    }
  },
  "model": {
    "root": {
      "kind": "context", "title": "Root",
      "children": {
        "mid": {
          "kind": "zone", "title": "Mid",
          "children": {
            "top":    { "kind": "part", "title": "Top Part",    "metadata": { "side": "top" } },
            "bottom": { "kind": "part", "title": "Bottom Part", "metadata": { "side": "bottom" } }
          }
        }
      }
    }
  },
  "views": {
    "rings": {
      "title": "Rings",
      "scope": "root",
      "include": ["root.mid.top", "root.mid.bottom"],
      "layout": "nested"
    }
  }
}`

func TestSyncNestedLayout(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "architecture.jsonc")
	drawioPath := filepath.Join(dir, "architecture.drawio")

	writeFile(t, modelPath, nestedLayoutModel)
	runCLI(t, bin, dir, "sync")

	doc, err := drawio.LoadDocument(drawioPath)
	if err != nil {
		t.Fatalf("load generated draw.io: %v", err)
	}

	// Locate the page holding the nested elements.
	var page *drawio.Page
	for _, p := range doc.Pages() {
		if p.FindElement("root.mid.top") != nil {
			page = p
			break
		}
	}
	if page == nil {
		t.Fatal("nested elements not found on any page")
	}

	// The scope boundary and the intermediate container are distinct cells.
	scope := page.FindElement("root")
	if scope == nil {
		t.Fatal("scope element 'root' not on page")
	}
	scopeCellID := scope.SelectAttrValue("id", "")

	// Robustness: the intermediate container was NOT in the include list but
	// must still be placed (ancestor-closure), otherwise its leaves are dropped.
	container := page.FindElement("root.mid")
	if container == nil {
		t.Fatal("intermediate container 'root.mid' was dropped — nested layout must pull in ancestor containers")
	}
	containerCellID := container.SelectAttrValue("id", "")

	// The container is re-parented under the scope boundary.
	cCell := container.SelectElement("mxCell")
	if cCell == nil {
		t.Fatal("container 'root.mid' has no mxCell")
	}
	if got := cCell.SelectAttrValue("parent", ""); got != scopeCellID {
		t.Errorf("container parent = %q, want scope cell %q", got, scopeCellID)
	}

	// The container geometry is resized to fit its content (larger than a bare
	// default leaf box of 120x60), i.e. setCellSize applied the computed size.
	w, h := geometryWH(t, cCell)
	if w <= 120 || h <= 60 {
		t.Errorf("container not resized to fit content: got %gx%g, want > 120x60", w, h)
	}

	// Each leaf is re-parented under its immediate container, NOT flattened to
	// the scope boundary.
	for _, leaf := range []string{"root.mid.top", "root.mid.bottom"} {
		obj := page.FindElement(leaf)
		if obj == nil {
			t.Fatalf("leaf %q not on page", leaf)
		}
		cell := obj.SelectElement("mxCell")
		if cell == nil {
			t.Fatalf("leaf %q has no mxCell", leaf)
		}
		parent := cell.SelectAttrValue("parent", "")
		if parent != containerCellID {
			t.Errorf("leaf %q parent = %q, want container cell %q", leaf, parent, containerCellID)
		}
		if parent == scopeCellID {
			t.Errorf("leaf %q was flattened to the scope boundary instead of nested under its container", leaf)
		}
	}
}

// geometryWH reads width/height off a cell's mxGeometry child.
func geometryWH(t *testing.T, cell *etree.Element) (float64, float64) {
	t.Helper()
	geo := cell.SelectElement("mxGeometry")
	if geo == nil {
		t.Fatal("cell has no mxGeometry")
	}
	w, _ := strconv.ParseFloat(geo.SelectAttrValue("width", "0"), 64)
	h, _ := strconv.ParseFloat(geo.SelectAttrValue("height", "0"), 64)
	return w, h
}
