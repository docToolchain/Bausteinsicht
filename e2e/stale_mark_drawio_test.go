package e2e

// TestStaleMarkDrawio verifies `stale --mark-drawio` / `--unmark-drawio`
// (visual badge marking of stale elements in the draw.io diagram) and
// `stale --format json`. TestStaleDetection (#496) only covers the plain
// text-detection path; the draw.io marking round-trip and JSON output had no
// prior E2E coverage.

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/beevik/etree"
	"github.com/docToolchain/Bausteinsicht/internal/drawio"
	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/docToolchain/Bausteinsicht/internal/overlay"
)

func TestStaleMarkDrawio(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	runCLI(t, bin, dir, "init")

	// Add an element far past the staleness threshold and sync it into draw.io.
	modelPath := filepath.Join(dir, "architecture.jsonc")
	m, err := model.Load(modelPath)
	if err != nil {
		t.Fatalf("model.Load: %v", err)
	}
	m.Model["legacy-service"] = model.Element{
		Title:        "Legacy Service",
		Kind:         "system",
		LastModified: "2020-01-01T00:00:00Z",
	}
	// The sample model's views use explicit include lists (not wildcards), so
	// a newly added top-level element must be added to a view's include list
	// to actually be synced into draw.io.
	contextView := m.Views["context"]
	contextView.Include = append(contextView.Include, "legacy-service")
	m.Views["context"] = contextView
	if err := model.Save(modelPath, m); err != nil {
		t.Fatalf("model.Save: %v", err)
	}
	runCLI(t, bin, dir, "sync", "--model", "architecture.jsonc")

	drawioPath := filepath.Join(dir, "architecture.drawio")

	// ── stale --format json: verify the actual stale-element entry ─────────────
	jsonOut := runCLI(t, bin, dir, "stale", "--model", "architecture.jsonc", "--format", "json")
	var result struct {
		StaleElements []struct {
			ID string `json:"id"`
		} `json:"staleElements"`
	}
	if err := json.Unmarshal([]byte(jsonOut), &result); err != nil {
		t.Fatalf("stale --format json: invalid JSON: %v\noutput: %s", err, jsonOut)
	}
	foundLegacy := false
	for _, e := range result.StaleElements {
		if e.ID == "legacy-service" {
			foundLegacy = true
			break
		}
	}
	if !foundLegacy {
		t.Fatalf("stale json: expected \"legacy-service\" in staleElements, got: %+v", result.StaleElements)
	}

	// ── stale --mark-drawio: badge marker attribute appears on the element ─────
	markOut := runCLI(t, bin, dir, "stale", "--model", "architecture.jsonc", "--mark-drawio")
	if !strings.Contains(markOut, "Marked") {
		t.Errorf("stale --mark-drawio: expected a \"Marked N stale elements\" message, got: %s", markOut)
	}

	doc, err := drawio.LoadDocument(drawioPath)
	if err != nil {
		t.Fatalf("LoadDocument after mark-drawio: %v", err)
	}
	obj := findElementInDoc(doc, "legacy-service")
	if obj == nil {
		t.Fatal("legacy-service object not found in draw.io after sync")
	}
	cell := obj.FindElement("mxCell")
	if cell == nil {
		t.Fatal("legacy-service object has no mxCell child")
	}
	// The marker attributes are stored on the mxCell, not the <object>
	// (internal/stale/drawio.go's markStaleElement); overlay.OriginalFillAttr's
	// mere *presence* (not a non-empty value) signals "marked", since an
	// element with no original fillColor still gets the attribute with an
	// empty value so UnmarkInDrawio knows to remove the key rather than
	// restore a color.
	if cell.SelectAttr(overlay.OriginalFillAttr) == nil {
		t.Error("stale --mark-drawio: expected the original-fill marker attribute on legacy-service's mxCell after marking")
	}
	style := cell.SelectAttrValue("style", "")
	if !strings.Contains(style, "fillColor=") {
		t.Errorf("stale --mark-drawio: expected a risk-color fillColor in style after marking, got: %s", style)
	}

	// ── stale --unmark-drawio: marker attribute removed ─────────────────────────
	runCLI(t, bin, dir, "stale", "--model", "architecture.jsonc", "--unmark-drawio")
	doc2, err := drawio.LoadDocument(drawioPath)
	if err != nil {
		t.Fatalf("LoadDocument after unmark-drawio: %v", err)
	}
	obj2 := findElementInDoc(doc2, "legacy-service")
	if obj2 == nil {
		t.Fatal("legacy-service object not found in draw.io after unmark")
	}
	cell2 := obj2.FindElement("mxCell")
	if cell2 == nil {
		t.Fatal("legacy-service object has no mxCell child after unmark")
	}
	if cell2.SelectAttr(overlay.OriginalFillAttr) != nil {
		t.Error("stale --unmark-drawio: original-fill marker attribute still present on legacy-service's mxCell after unmarking")
	}
}

// findElementInDoc searches all pages of doc for the element with the given
// bausteinsicht_id.
func findElementInDoc(doc *drawio.Document, id string) *etree.Element {
	for _, p := range doc.Pages() {
		if el := p.FindElement(id); el != nil {
			return el
		}
	}
	return nil
}
