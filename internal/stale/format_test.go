package stale

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestFormatText_NoStaleElements(t *testing.T) {
	result := DetectionResult{StaleElements: []StaleElement{}, TotalElements: 5}
	out := FormatText(result)
	if !strings.Contains(out, "No stale elements found") {
		t.Errorf("expected 'No stale elements found', got: %s", out)
	}
	if !strings.Contains(out, "5") {
		t.Errorf("expected total count 5 in output, got: %s", out)
	}
}

func TestFormatText_WithStaleElements(t *testing.T) {
	elems := []StaleElement{
		{
			ID:                "shop.api",
			Kind:              "system",
			DaysSinceModified: 120,
			MissingStatus:     true,
			MissingADR:        true,
			IncomingRelCount:  2,
			Risk:              RiskHigh,
		},
		{
			ID:                "shop.db",
			Kind:              "database",
			DaysSinceModified: 200,
			MissingStatus:     false,
			MissingADR:        false,
			IncomingRelCount:  0,
			Risk:              RiskLow,
		},
	}
	result := DetectionResult{StaleElements: elems, TotalElements: 10}
	out := FormatText(result)

	if !strings.Contains(out, "shop.api") {
		t.Error("expected shop.api in output")
	}
	if !strings.Contains(out, "shop.db") {
		t.Error("expected shop.db in output")
	}
	if !strings.Contains(out, "Risk summary") {
		t.Error("expected risk summary section")
	}
	if !strings.Contains(out, "Suggested actions") {
		t.Error("expected suggested actions section")
	}
}

func TestFormatText_MissingStatusAndADR(t *testing.T) {
	elems := []StaleElement{
		{
			ID:            "svc",
			Kind:          "service",
			MissingStatus: true,
			MissingADR:    true,
			Risk:          RiskMedium,
		},
	}
	out := FormatText(DetectionResult{StaleElements: elems, TotalElements: 1})
	if !strings.Contains(out, "No lifecycle status set") {
		t.Error("expected 'No lifecycle status set'")
	}
	if !strings.Contains(out, "No ADR linked") {
		t.Error("expected 'No ADR linked'")
	}
}

func TestFormatText_StatusSet(t *testing.T) {
	elems := []StaleElement{
		{ID: "svc", Kind: "service", MissingStatus: false, MissingADR: false, Risk: RiskLow},
	}
	out := FormatText(DetectionResult{StaleElements: elems, TotalElements: 1})
	if !strings.Contains(out, "Status set") {
		t.Error("expected 'Status set' in output")
	}
}

func TestFormatJSON(t *testing.T) {
	result := DetectionResult{
		StaleElements: []StaleElement{{ID: "x", Kind: "system", Risk: RiskHigh}},
		TotalElements: 3,
		Timestamp:     time.Now(),
	}
	out, err := FormatJSON(result)
	if err != nil {
		t.Fatalf("FormatJSON error: %v", err)
	}
	var decoded DetectionResult
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("FormatJSON produced invalid JSON: %v", err)
	}
	if decoded.TotalElements != 3 {
		t.Errorf("expected TotalElements=3, got %d", decoded.TotalElements)
	}
}

func TestRiskIcon(t *testing.T) {
	tests := []struct {
		risk RiskLevel
		want string
	}{
		{RiskHigh, "🔴"},
		{RiskMedium, "🟡"},
		{RiskLow, "✅"},
		{"unknown", "❓"},
	}
	for _, tt := range tests {
		got := riskIcon(tt.risk)
		if got != tt.want {
			t.Errorf("riskIcon(%q) = %q, want %q", tt.risk, got, tt.want)
		}
	}
}

func TestCountByRisk(t *testing.T) {
	elems := []StaleElement{
		{Risk: RiskHigh},
		{Risk: RiskHigh},
		{Risk: RiskMedium},
		{Risk: RiskLow},
	}
	if got := countByRisk(elems, RiskHigh); got != 2 {
		t.Errorf("countByRisk(High) = %d, want 2", got)
	}
	if got := countByRisk(elems, RiskMedium); got != 1 {
		t.Errorf("countByRisk(Medium) = %d, want 1", got)
	}
	if got := countByRisk(elems, RiskLow); got != 1 {
		t.Errorf("countByRisk(Low) = %d, want 1", got)
	}
}
