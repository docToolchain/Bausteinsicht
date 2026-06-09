package stale

import (
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
