package overlay

import (
	"testing"
)

func TestNormalize_HigherIsBetter(t *testing.T) {
	metrics := []NormalizedMetric{
		{ElementID: "a", Value: 50},
		{ElementID: "b", Value: 100},
		{ElementID: "c", Value: 0},
	}

	result := Normalize(metrics, true)

	if result["c"] != 0 {
		t.Errorf("expected c=0 (lowest), got %f", result["c"])
	}
	if result["b"] != 1 {
		t.Errorf("expected b=1 (highest), got %f", result["b"])
	}
	if result["a"] != 0.5 {
		t.Errorf("expected a=0.5 (middle), got %f", result["a"])
	}
}

func TestNormalize_LowerIsBetter(t *testing.T) {
	metrics := []NormalizedMetric{
		{ElementID: "a", Value: 50},
		{ElementID: "b", Value: 100},
		{ElementID: "c", Value: 0},
	}

	result := Normalize(metrics, false)

	if result["c"] != 1 {
		t.Errorf("expected c=1 (lowest value, best), got %f", result["c"])
	}
	if result["b"] != 0 {
		t.Errorf("expected b=0 (highest value, worst), got %f", result["b"])
	}
	if result["a"] != 0.5 {
		t.Errorf("expected a=0.5 (middle), got %f", result["a"])
	}
}

func TestNormalize_AllSameValues(t *testing.T) {
	metrics := []NormalizedMetric{
		{ElementID: "a", Value: 50},
		{ElementID: "b", Value: 50},
		{ElementID: "c", Value: 50},
	}

	result := Normalize(metrics, true)

	if result["a"] != 0 || result["b"] != 0 || result["c"] != 0 {
		t.Errorf("expected all 0 when values are identical, got %v", result)
	}
}

func TestColorForValue(t *testing.T) {
	tests := []struct {
		normalized float64
		expected   string
	}{
		{0.1, "#d5e8d4"},  // Green
		{0.3, "#fff2cc"},  // Yellow
		{0.6, "#ffe6cc"},  // Orange
		{0.9, "#f8cecc"},  // Red
	}

	for _, tc := range tests {
		result := ColorForValue(tc.normalized, DefaultColorScheme)
		if result != tc.expected {
			t.Errorf("normalized=%f: expected %s, got %s", tc.normalized, tc.expected, result)
		}
	}
}

func TestIsMetricBetter(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"coverage", true},
		{"uptime", true},
		{"deploy_freq", true},
		{"error_rate", false},
		{"latency", false},
		{"p99_ms", false},
		{"unknown_metric", false},
	}

	for _, tc := range tests {
		result := IsMetricBetter(tc.name)
		if result != tc.expected {
			t.Errorf("%s: expected %v, got %v", tc.name, tc.expected, result)
		}
	}
}

func TestExtractMetric(t *testing.T) {
	metrics := []ElementMetric{
		{
			ElementID: "service-b",
			Values:    map[string]float64{"error_rate": 2.0, "coverage": 95},
		},
		{
			ElementID: "service-a",
			Values:    map[string]float64{"error_rate": 5.0, "coverage": 80},
		},
	}

	result, err := ExtractMetric(metrics, "error_rate")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 results, got %d", len(result))
	}

	if result[0].ElementID != "service-a" || result[1].ElementID != "service-b" {
		t.Errorf("expected sorted by elementID, got %+v", result)
	}

	if result[0].Value != 5.0 || result[1].Value != 2.0 {
		t.Errorf("expected correct values, got %+v", result)
	}
}
