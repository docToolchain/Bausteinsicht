package stale

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/beevik/etree"
)

func TestRiskColor(t *testing.T) {
	tests := []struct {
		risk RiskLevel
		want string
	}{
		{RiskHigh, "#FF6666"},
		{RiskMedium, "#FFBB66"},
		{RiskLow, "#66DD66"},
		{"unknown", "#CCCCCC"},
	}
	for _, tt := range tests {
		got := riskColor(tt.risk)
		if got != tt.want {
			t.Errorf("riskColor(%q) = %q, want %q", tt.risk, got, tt.want)
		}
	}
}

func TestMarkStaleElement_NoCell(t *testing.T) {
	obj := etree.NewElement("object")
	// No mxCell child — should be a no-op, not panic.
	markStaleElement(obj, StaleElement{Risk: RiskHigh, LastModified: time.Now()})
}

func TestMarkStaleElement_SetsStyleAndTooltip(t *testing.T) {
	obj := etree.NewElement("object")
	cell := obj.CreateElement("mxCell")
	cell.CreateAttr("style", "rounded=1;")

	elem := StaleElement{
		Risk:         RiskHigh,
		LastModified: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
	}
	markStaleElement(obj, elem)

	style := cell.SelectAttrValue("style", "")
	if !strings.Contains(style, "fillColor=#FF6666") {
		t.Errorf("style missing fillColor: %s", style)
	}
	if !strings.Contains(style, "strokeColor=#FF6666") {
		t.Errorf("style missing strokeColor: %s", style)
	}
	if !strings.Contains(style, "strokeWidth=2") {
		t.Errorf("style missing strokeWidth: %s", style)
	}

	tooltip := obj.SelectAttrValue("tooltip", "")
	if !strings.Contains(tooltip, "STALE") {
		t.Errorf("tooltip missing STALE marker: %s", tooltip)
	}
	if !strings.Contains(tooltip, "2025-01-15") {
		t.Errorf("tooltip missing date: %s", tooltip)
	}
}

func TestMarkStaleElement_IdempotentStyle(t *testing.T) {
	obj := etree.NewElement("object")
	cell := obj.CreateElement("mxCell")
	cell.CreateAttr("style", "rounded=1;fillColor=#dae8fc;")

	highElem := StaleElement{Risk: RiskHigh, LastModified: time.Now()}
	// First mark: should store original fill and apply risk color.
	markStaleElement(obj, highElem)

	originalFill := cell.SelectAttrValue("data-original-fill", "__missing__")
	if originalFill != "#dae8fc" {
		t.Errorf("first mark: expected original fill #dae8fc stored, got %q", originalFill)
	}

	// Second mark (different risk): data-original-fill must NOT be overwritten.
	medElem := StaleElement{Risk: RiskMedium, LastModified: time.Now()}
	markStaleElement(obj, medElem)

	originalFillAfter := cell.SelectAttrValue("data-original-fill", "__missing__")
	if originalFillAfter != "#dae8fc" {
		t.Errorf("second mark overwrote data-original-fill: got %q, want #dae8fc", originalFillAfter)
	}

	style := cell.SelectAttrValue("style", "")
	// Should not accumulate duplicate strokeWidth entries.
	if strings.Count(style, "strokeWidth") > 1 {
		t.Errorf("duplicate strokeWidth in style: %s", style)
	}
	if !strings.Contains(style, "fillColor=#FFBB66") {
		t.Errorf("expected medium risk color in style after second mark: %s", style)
	}
}

func TestMarkStaleElement_LowRisk(t *testing.T) {
	obj := etree.NewElement("object")
	cell := obj.CreateElement("mxCell")
	cell.CreateAttr("style", "")

	markStaleElement(obj, StaleElement{Risk: RiskLow, LastModified: time.Now()})

	style := cell.SelectAttrValue("style", "")
	if !strings.Contains(style, "fillColor=#66DD66") {
		t.Errorf("expected low risk color: %s", style)
	}
}

// drawioXML creates a minimal valid draw.io file containing one <object> element
// with the given bausteinsicht_id attribute.
func drawioXML(bausteinsichtID string) string {
	return `<?xml version="1.0" encoding="UTF-8"?><mxfile><diagram id="d1" name="Page"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><object bausteinsicht_id="` + bausteinsichtID + `" label="API"><mxCell id="cell1" style="rounded=1;" vertex="1" parent="1"/></object></root></mxGraphModel></diagram></mxfile>`
}

func writeTempDrawio(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "test-*.drawio")
	if err != nil {
		t.Fatalf("create temp drawio: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("write temp drawio: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close temp drawio: %v", err)
	}
	return f.Name()
}

func TestMarkInDrawio_FileNotFound(t *testing.T) {
	err := MarkInDrawio([]StaleElement{{ID: "x", Risk: RiskHigh}}, "/nonexistent/file.drawio")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestMarkInDrawio_ElementInDiagram(t *testing.T) {
	path := writeTempDrawio(t, drawioXML("shop.api"))

	staleElems := []StaleElement{
		{ID: "shop.api", Risk: RiskHigh, LastModified: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	if err := MarkInDrawio(staleElems, path); err != nil {
		t.Fatalf("MarkInDrawio: %v", err)
	}

	// Reload the saved file and verify the style was applied.
	tree := etree.NewDocument()
	if err := tree.ReadFromFile(path); err != nil {
		t.Fatalf("reading saved file: %v", err)
	}
	obj := tree.FindElement("//object[@bausteinsicht_id='shop.api']")
	if obj == nil {
		t.Fatal("object element not found after save")
	}
	cell := obj.FindElement("mxCell")
	if cell == nil {
		t.Fatal("mxCell not found")
	}
	style := cell.SelectAttrValue("style", "")
	if !strings.Contains(style, "fillColor=#FF6666") {
		t.Errorf("expected high-risk fill color in style, got: %s", style)
	}
	tooltip := obj.SelectAttrValue("tooltip", "")
	if !strings.Contains(tooltip, "STALE") {
		t.Errorf("expected STALE in tooltip, got: %s", tooltip)
	}
}

func TestMarkInDrawio_ElementNotInDiagram(t *testing.T) {
	path := writeTempDrawio(t, drawioXML("other.element"))

	// Request marking an element that does NOT exist in the diagram.
	err := MarkInDrawio([]StaleElement{{ID: "shop.api", Risk: RiskHigh}}, path)
	if err != nil {
		t.Fatalf("MarkInDrawio with unknown element ID: %v", err)
	}
}

func TestUnmarkInDrawio_RestoresOriginalFill(t *testing.T) {
	// Build a draw.io XML that already has stale markup applied (including data-original-fill).
	xml := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<mxfile><diagram id="d1" name="Page"><mxGraphModel><root>` +
		`<mxCell id="0"/><mxCell id="1" parent="0"/>` +
		`<object bausteinsicht_id="shop.api" label="API" tooltip="⚠ STALE&#10;Last modified: 2025-01-15&#10;No status set&#10;No ADR linked">` +
		`<mxCell id="cell1" style="rounded=1;fillColor=#FF6666;strokeColor=#FF6666;strokeWidth=2;" data-original-fill="#dae8fc" vertex="1" parent="1"/>` +
		`</object>` +
		`</root></mxGraphModel></diagram></mxfile>`
	path := writeTempDrawio(t, xml)

	count, err := UnmarkInDrawio(path)
	if err != nil {
		t.Fatalf("UnmarkInDrawio: %v", err)
	}
	if count != 1 {
		t.Errorf("UnmarkInDrawio: expected 1 unmarked element, got %d", count)
	}

	tree := etree.NewDocument()
	if err := tree.ReadFromFile(path); err != nil {
		t.Fatalf("reading saved file: %v", err)
	}
	obj := tree.FindElement("//object[@bausteinsicht_id='shop.api']")
	if obj == nil {
		t.Fatal("object element not found after unmark")
	}
	cell := obj.FindElement("mxCell")
	if cell == nil {
		t.Fatal("mxCell not found")
	}
	style := cell.SelectAttrValue("style", "")
	if !strings.Contains(style, "fillColor=#dae8fc") {
		t.Errorf("expected original fill color restored, got style: %s", style)
	}
	if strings.Contains(style, "strokeColor") {
		t.Errorf("strokeColor should have been removed, got style: %s", style)
	}
	if strings.Contains(style, "strokeWidth") {
		t.Errorf("strokeWidth should have been removed, got style: %s", style)
	}
	if cell.SelectAttrValue("data-original-fill", "") != "" {
		t.Error("data-original-fill attribute should have been removed")
	}
	if obj.SelectAttrValue("tooltip", "") != "" {
		t.Error("tooltip should have been removed")
	}
}

func TestUnmarkInDrawio_NoMarkedElements(t *testing.T) {
	// A file with no data-original-fill — UnmarkInDrawio should be a no-op.
	path := writeTempDrawio(t, drawioXML("shop.api"))
	count, err := UnmarkInDrawio(path)
	if err != nil {
		t.Fatalf("UnmarkInDrawio on unmarked file: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 unmarked elements, got %d", count)
	}
}

func TestUnmarkInDrawio_FileNotFound(t *testing.T) {
	_, err := UnmarkInDrawio("/nonexistent/file.drawio")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

// TestUnmarkInDrawio_NoOriginalFillColor verifies that when fillColor was absent
// before marking (stored as "" in data-original-fill), unmark removes fillColor
// from the style instead of setting it to a fallback value.
func TestUnmarkInDrawio_NoOriginalFillColor(t *testing.T) {
	// style has no fillColor originally; mark added fillColor=#FF6666.
	// data-original-fill="" means fillColor was absent before marking.
	xml := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<mxfile><diagram id="d1" name="Page"><mxGraphModel><root>` +
		`<mxCell id="0"/><mxCell id="1" parent="0"/>` +
		`<object bausteinsicht_id="elem1" label="E1">` +
		`<mxCell id="c1" style="rounded=1;fillColor=#FF6666;strokeColor=#FF6666;strokeWidth=2;" ` +
		`data-original-fill="" vertex="1" parent="1"/>` +
		`</object>` +
		`</root></mxGraphModel></diagram></mxfile>`
	path := writeTempDrawio(t, xml)

	count, err := UnmarkInDrawio(path)
	if err != nil {
		t.Fatalf("UnmarkInDrawio: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 unmarked element, got %d", count)
	}

	tree := etree.NewDocument()
	if err := tree.ReadFromFile(path); err != nil {
		t.Fatalf("reading saved file: %v", err)
	}
	cell := tree.FindElement("//mxCell[@id='c1']")
	if cell == nil {
		t.Fatal("mxCell not found after unmark")
	}
	style := cell.SelectAttrValue("style", "")
	if strings.Contains(style, "fillColor") {
		t.Errorf("fillColor should have been removed (was absent before marking), got style: %s", style)
	}
}

func TestRemoveStyleProperties_RemovesKeys(t *testing.T) {
	style := "rounded=1;fillColor=#FF6666;strokeColor=#FF6666;strokeWidth=2;"
	got := removeStyleProperties(style, []string{"strokeColor", "strokeWidth"})
	if strings.Contains(got, "strokeColor") {
		t.Errorf("strokeColor not removed from: %s", got)
	}
	if strings.Contains(got, "strokeWidth") {
		t.Errorf("strokeWidth not removed from: %s", got)
	}
	if !strings.Contains(got, "rounded=1") {
		t.Errorf("rounded=1 should remain: %s", got)
	}
	if !strings.Contains(got, "fillColor=#FF6666") {
		t.Errorf("fillColor should remain: %s", got)
	}
}

func TestRemoveStyleProperties_EmptyKeys(t *testing.T) {
	style := "rounded=1;fillColor=#abc;"
	got := removeStyleProperties(style, nil)
	if !strings.Contains(got, "rounded=1") || !strings.Contains(got, "fillColor=#abc") {
		t.Errorf("removeStyleProperties with no keys should be identity, got: %s", got)
	}
}

func TestRemoveStyleProperties_EmptyStyle(t *testing.T) {
	got := removeStyleProperties("", []string{"fillColor"})
	if got != "" {
		t.Errorf("expected empty string, got: %s", got)
	}
}
