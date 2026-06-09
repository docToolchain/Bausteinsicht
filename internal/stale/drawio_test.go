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
	cell.CreateAttr("style", "rounded=1;fillColor=#FF6666;strokeColor=#FF6666;strokeWidth=2")

	elem := StaleElement{Risk: RiskMedium, LastModified: time.Now()}
	markStaleElement(obj, elem)

	style := cell.SelectAttrValue("style", "")
	// Should not accumulate duplicate strokeWidth entries.
	if strings.Count(style, "strokeWidth") > 1 {
		t.Errorf("duplicate strokeWidth in style: %s", style)
	}
	if !strings.Contains(style, "fillColor=#FFBB66") {
		t.Errorf("expected medium risk color in style: %s", style)
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
