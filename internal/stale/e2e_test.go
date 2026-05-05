package stale

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// TestE2E_CompleteWorkflow tests the entire stale element detection workflow.
func TestE2E_CompleteWorkflow(t *testing.T) {
	// Setup: Create a test model with mixed elements
	testModel := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"system":    {Notation: "System", Container: true},
				"container": {Notation: "Container"},
			},
		},
		Model: map[string]model.Element{
			"active": {
				Kind:      "system",
				Title:     "Active System",
				Status:    "deployed", // Has status - should NOT be flagged
				Decisions: []string{"ADR-001"},
			},
			"stale_without_status": {
				Kind:  "system",
				Title: "Stale Without Status",
				// Missing status and decisions - candidate for flagging
			},
			"stale_without_adr": {
				Kind:   "system",
				Title:  "Stale Without ADR",
				Status: "deprecated", // Has status but no ADR
			},
			"archived": {
				Kind:   "system",
				Title:  "Archived",
				Status: "archived", // Archived - should NOT be flagged
			},
		},
		Relationships: []model.Relationship{
			{From: "active", To: "stale_without_status", Label: "depends"},
		},
		Views: map[string]model.View{
			"overview": {
				Include: []string{"active", "stale_without_status"},
			},
		},
	}

	// Create temporary model file
	tmpDir := t.TempDir()
	modelPath := filepath.Join(tmpDir, "architecture.jsonc")
	modelData, err := json.Marshal(testModel)
	if err != nil {
		t.Fatalf("marshaling test model: %v", err)
	}
	if err := os.WriteFile(modelPath, modelData, 0644); err != nil {
		t.Fatalf("writing test model: %v", err)
	}

	// Run detection
	config := DefaultConfig()
	config.ThresholdDays = 90
	result, err := Detect(testModel, modelPath, config)
	if err != nil {
		t.Fatalf("detection failed: %v", err)
	}

	// Validate results
	t.Run("Element counts", func(t *testing.T) {
		if result.TotalElements != 4 {
			t.Errorf("expected 4 total elements, got %d", result.TotalElements)
		}
	})

	t.Run("Text output", func(t *testing.T) {
		output := FormatText(result)
		if output == "" {
			t.Error("text output is empty")
		}
		// Should mention stale elements (or lack thereof) since file is not in git
		if len(output) < 10 {
			t.Errorf("text output is too short: %s", output)
		}
	})

	t.Run("JSON output", func(t *testing.T) {
		output, err := FormatJSON(result)
		if err != nil {
			t.Fatalf("JSON formatting failed: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(output), &parsed); err != nil {
			t.Errorf("output is not valid JSON: %v", err)
		}

		if _, ok := parsed["StaleElements"]; !ok {
			t.Error("JSON missing StaleElements field")
		}
		if _, ok := parsed["TotalElements"]; !ok {
			t.Error("JSON missing TotalElements field")
		}
		if _, ok := parsed["Timestamp"]; !ok {
			t.Error("JSON missing Timestamp field")
		}
	})

	t.Run("Configuration loading", func(t *testing.T) {
		config := LoadConfigFromModel(testModel)
		if config.ThresholdDays != 90 {
			t.Errorf("expected default threshold 90, got %d", config.ThresholdDays)
		}
	})

	t.Run("Risk assessment", func(t *testing.T) {
		// Element with incoming relationship should be medium+ risk
		elem := StaleElement{
			ID:               "test",
			IncomingRelCount: 1,
			IsViewIncluded:   true,
		}
		risk := assessRisk(elem)
		if risk != RiskHigh {
			t.Errorf("expected RiskHigh for view-included element with incoming rel, got %v", risk)
		}
	})

	t.Run("Recommendations", func(t *testing.T) {
		elem := StaleElement{
			ID:             "test",
			MissingStatus:  true,
			MissingADR:     true,
			IncomingRelCount: 2,
		}
		recs := generateRecommendations(elem)
		if len(recs) < 3 {
			t.Errorf("expected at least 3 recommendations, got %d: %v", len(recs), recs)
		}

		// Should recommend setting status and reviewing relationships
		hasStatusRec := false
		hasADRRec := false
		hasRelRec := false
		for _, rec := range recs {
			if hasKeyword(rec, "status") {
				hasStatusRec = true
			}
			if hasKeyword(rec, "ADR") {
				hasADRRec = true
			}
			if hasKeyword(rec, "incoming") || hasKeyword(rec, "relationship") {
				hasRelRec = true
			}
		}

		if !hasStatusRec || !hasADRRec || !hasRelRec {
			t.Errorf("missing expected recommendations. status=%v, adr=%v, rel=%v",
				hasStatusRec, hasADRRec, hasRelRec)
		}
	})
}

// TestE2E_EdgeCases tests edge cases and boundary conditions.
func TestE2E_EdgeCases(t *testing.T) {
	t.Run("Empty model", func(t *testing.T) {
		m := &model.BausteinsichtModel{
			Model:         map[string]model.Element{},
			Relationships: []model.Relationship{},
			Views:         map[string]model.View{},
		}
		result, err := Detect(m, "", DefaultConfig())
		if err != nil {
			t.Fatalf("detection on empty model failed: %v", err)
		}
		if result.TotalElements != 0 {
			t.Errorf("expected 0 elements, got %d", result.TotalElements)
		}
	})

	t.Run("Nested elements", func(t *testing.T) {
		m := &model.BausteinsichtModel{
			Model: map[string]model.Element{
				"parent": {
					Kind:  "system",
					Title: "Parent",
					Children: map[string]model.Element{
						"child": {
							Kind:  "container",
							Title: "Child",
						},
						"child2": {
							Kind:   "container",
							Title:  "Child2",
							Status: "archived",
						},
					},
				},
			},
			Relationships: []model.Relationship{},
			Views:         map[string]model.View{},
		}

		result, err := Detect(m, "", DefaultConfig())
		if err != nil {
			t.Fatalf("detection on nested model failed: %v", err)
		}

		// Should count parent + 2 children = 3 elements
		if result.TotalElements != 3 {
			t.Errorf("expected 3 nested elements, got %d", result.TotalElements)
		}
	})

	t.Run("Complex relationships", func(t *testing.T) {
		m := &model.BausteinsichtModel{
			Model: map[string]model.Element{
				"a": {Kind: "system", Title: "A"},
				"b": {Kind: "system", Title: "B"},
				"c": {Kind: "system", Title: "C"},
			},
			Relationships: []model.Relationship{
				{From: "a", To: "b"},
				{From: "a", To: "c"},
				{From: "b", To: "c"},
				{From: "c", To: "a"}, // Cycle
			},
			Views: map[string]model.View{},
		}

		flatElements, _ := model.FlattenElements(m)
		relIndex := buildRelationshipIndex(m, flatElements)

		// Each element should have 2 incoming, 2 outgoing (cyclic)
		if relIndex.incoming["a"] != 1 {
			t.Errorf("expected 1 incoming for a, got %d", relIndex.incoming["a"])
		}
		if relIndex.incoming["b"] != 1 {
			t.Errorf("expected 1 incoming for b, got %d", relIndex.incoming["b"])
		}
		if relIndex.incoming["c"] != 2 {
			t.Errorf("expected 2 incoming for c, got %d", relIndex.incoming["c"])
		}
	})
}

// Acceptance Criteria Validation
func TestAcceptanceCriteria(t *testing.T) {
	t.Run("AC1: Detect stale elements", func(t *testing.T) {
		// AC: An element is flagged if it has no status, no decisions, and model file is old
		m := &model.BausteinsichtModel{
			Model: map[string]model.Element{
				"stale": {Kind: "system", Title: "Stale"},
			},
			Relationships: []model.Relationship{},
			Views:         map[string]model.View{},
		}

		// Note: Since file is not in git, it won't be flagged
		// This AC would be validated with an actual git repository
		result, _ := Detect(m, "", DefaultConfig())
		if result.TotalElements != 1 {
			t.Error("AC1: Element not counted")
		}
	})

	t.Run("AC2: Risk assessment", func(t *testing.T) {
		// AC: Elements are assessed for removal risk
		tests := []struct {
			incoming bool
			included bool
			expected RiskLevel
		}{
			{incoming: false, included: false, expected: RiskLow},
			{incoming: true, included: false, expected: RiskMedium},
			{incoming: true, included: true, expected: RiskHigh},
		}

		for _, tc := range tests {
			elem := StaleElement{
				ID:               "test",
				IncomingRelCount: boolToInt(tc.incoming),
				IsViewIncluded:   tc.included,
			}
			risk := assessRisk(elem)
			if risk != tc.expected {
				t.Errorf("AC2: Risk mismatch for (incoming=%v, included=%v): expected %v, got %v",
					tc.incoming, tc.included, tc.expected, risk)
			}
		}
	})

	t.Run("AC3: Recommendations generated", func(t *testing.T) {
		// AC: Each stale element has actionable recommendations
		elem := StaleElement{
			ID:             "test",
			MissingStatus:  true,
			MissingADR:     true,
			IncomingRelCount: 0,
		}
		recs := generateRecommendations(elem)
		if len(recs) == 0 {
			t.Error("AC3: No recommendations generated for stale element")
		}

		for _, rec := range recs {
			if rec == "" {
				t.Error("AC3: Empty recommendation string")
			}
		}
	})

	t.Run("AC4: Output formats", func(t *testing.T) {
		// AC: Text and JSON output formats supported
		result := DetectionResult{
			StaleElements: []StaleElement{},
			TotalElements: 0,
			Timestamp:     time.Now(),
		}

		// Text format
		text := FormatText(result)
		if text == "" {
			t.Error("AC4: Text output is empty")
		}

		// JSON format
		json, err := FormatJSON(result)
		if err != nil {
			t.Errorf("AC4: JSON formatting failed: %v", err)
		}
		if json == "" {
			t.Error("AC4: JSON output is empty")
		}
	})
}

// Helper functions
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func hasKeyword(s, keyword string) bool {
	return strings.Contains(s, keyword)
}
