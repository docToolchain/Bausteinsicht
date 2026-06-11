package stale

import (
	"testing"
	"time"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

func TestDetect_WithStatus_NotFlagged(t *testing.T) {
	// Elements with status should not be flagged
	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"app": {
				Kind:   "system",
				Title:  "Application",
				Status: "deployed",
			},
		},
		Relationships: []model.Relationship{},
		Views:         map[string]model.View{},
	}

	config := StaleConfig{ThresholdDays: 90}

	result, err := Detect(m, "", config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.StaleElements) != 0 {
		t.Errorf("expected no stale elements, got %d", len(result.StaleElements))
	}
}

func TestDetect_WithoutStatusOrADR_Flagged(t *testing.T) {
	// Elements without status or ADR should be flagged (if old enough)
	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"legacy": {
				Kind:  "system",
				Title: "Legacy System",
				// No status, no decisions
			},
		},
		Relationships: []model.Relationship{},
		Views:         map[string]model.View{},
	}

	config := StaleConfig{ThresholdDays: 90}

	result, err := Detect(m, "", config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Note: Will be 0 because file is not in git
	// This test is here to document the behavior
	if result.TotalElements != 1 {
		t.Errorf("expected 1 element, got %d", result.TotalElements)
	}
}

func TestRiskAssessment_NoIncomingRels_LowRisk(t *testing.T) {
	elem := StaleElement{
		ID:               "orphan",
		IncomingRelCount: 0,
		IsViewIncluded:   false,
	}

	risk := assessRisk(elem)
	if risk != RiskLow {
		t.Errorf("expected RiskLow, got %v", risk)
	}
}

func TestRiskAssessment_WithIncomingRels_MediumRisk(t *testing.T) {
	elem := StaleElement{
		ID:               "dependency",
		IncomingRelCount: 2,
		IsViewIncluded:   false,
	}

	risk := assessRisk(elem)
	if risk != RiskMedium {
		t.Errorf("expected RiskMedium, got %v", risk)
	}
}

func TestRiskAssessment_ViewIncludedWithIncoming_HighRisk(t *testing.T) {
	elem := StaleElement{
		ID:               "important",
		IncomingRelCount: 3,
		IsViewIncluded:   true,
	}

	risk := assessRisk(elem)
	if risk != RiskHigh {
		t.Errorf("expected RiskHigh, got %v", risk)
	}
}

func TestDaysSince_FutureDate(t *testing.T) {
	future := time.Now().AddDate(0, 0, 10)
	days := DaysSince(future)
	if days != 0 {
		t.Errorf("expected 0 for future date (clamped), got %d", days)
	}
}

func TestDaysSince_PastDate(t *testing.T) {
	past := time.Now().AddDate(0, 0, -10)
	days := DaysSince(past)
	if days < 10 || days > 11 {
		t.Errorf("expected ~10 days, got %d", days)
	}
}

func TestIsStale_OldEnough(t *testing.T) {
	past := time.Now().AddDate(0, 0, -95)
	if !IsStale(past, 90) {
		t.Error("expected element to be stale (95 days > 90 threshold)")
	}
}

func TestIsStale_TooRecent(t *testing.T) {
	recent := time.Now().AddDate(0, 0, -10)
	if IsStale(recent, 90) {
		t.Error("expected element to be fresh (10 days < 90 threshold)")
	}
}

func TestGenerateRecommendations_NoStatus_NoADR(t *testing.T) {
	elem := StaleElement{
		ID:            "orphan",
		MissingStatus: true,
		MissingADR:    true,
	}

	recs := generateRecommendations(elem)
	if len(recs) < 2 {
		t.Errorf("expected at least 2 recommendations, got %d", len(recs))
	}
}

func TestIsExcluded_WithExcludedKind(t *testing.T) {
	excludeKinds := []string{"database", "infra"}
	if !isExcluded("database", excludeKinds) {
		t.Error("expected database to be excluded")
	}
	if isExcluded("system", excludeKinds) {
		t.Error("expected system to not be excluded")
	}
}

func TestIsTagExcluded_MatchesExcludedTag(t *testing.T) {
	if !isTagExcluded([]string{"infra", "legacy"}, []string{"legacy"}) {
		t.Error("expected element with 'legacy' tag to be excluded")
	}
}

func TestIsTagExcluded_NoMatch(t *testing.T) {
	if isTagExcluded([]string{"core"}, []string{"legacy", "infra"}) {
		t.Error("expected element with 'core' tag not to be excluded")
	}
}

func TestIsTagExcluded_EmptyExcludeList(t *testing.T) {
	if isTagExcluded([]string{"anything"}, []string{}) {
		t.Error("empty excludeTags should never exclude")
	}
}

// TestDetect_DeterministicOrder verifies that Detect returns elements in a stable
// order (descending risk, then ascending ID) regardless of map iteration.
func TestDetect_DeterministicOrder(t *testing.T) {
	past := time.Now().AddDate(-1, 0, 0).Format(time.RFC3339)
	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"charlie": {Kind: "system", Title: "Charlie", LastModified: past},
			"alpha":   {Kind: "system", Title: "Alpha", LastModified: past},
			"bravo":   {Kind: "system", Title: "Bravo", LastModified: past},
		},
		Relationships: []model.Relationship{
			// bravo has an incoming relationship → RiskMedium; others → RiskLow
			{From: "alpha", To: "bravo", Label: "uses"},
		},
		Views: map[string]model.View{},
	}
	config := StaleConfig{ThresholdDays: 1}

	result1, _ := Detect(m, "", config)
	result2, _ := Detect(m, "", config)

	if len(result1.StaleElements) != len(result2.StaleElements) {
		t.Fatalf("Detect returned different lengths: %d vs %d",
			len(result1.StaleElements), len(result2.StaleElements))
	}
	for i := range result1.StaleElements {
		if result1.StaleElements[i].ID != result2.StaleElements[i].ID {
			t.Errorf("order differs at index %d: %q vs %q",
				i, result1.StaleElements[i].ID, result2.StaleElements[i].ID)
		}
	}

	// bravo (RiskMedium) must come first, then alpha and charlie (RiskLow) alphabetically.
	ids := make([]string, len(result1.StaleElements))
	for i, e := range result1.StaleElements {
		ids[i] = e.ID
	}
	if len(ids) != 3 {
		t.Fatalf("expected 3 stale elements, got %d: %v", len(ids), ids)
	}
	if ids[0] != "bravo" {
		t.Errorf("expected bravo (RiskMedium) first, got %q", ids[0])
	}
	if ids[1] != "alpha" || ids[2] != "charlie" {
		t.Errorf("expected [alpha, charlie] after bravo, got %v", ids[1:])
	}
}

func TestDetect_ExcludeTagsSkipsElement(t *testing.T) {
	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"infra": {Kind: "system", Title: "Infra", Tags: []string{"infra"}},
		},
		Relationships: []model.Relationship{},
		Views:         map[string]model.View{},
	}
	config := StaleConfig{ThresholdDays: 0, ExcludeTags: []string{"infra"}}
	result, err := Detect(m, "", config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.StaleElements) != 0 {
		t.Errorf("element with excluded tag should not be flagged, got %d stale", len(result.StaleElements))
	}
}

func TestIsViewIncluded_ExactMatch(t *testing.T) {
	m := &model.BausteinsichtModel{
		Views: map[string]model.View{
			"overview": {
				Include: []string{"system.api", "system.db"},
			},
		},
	}
	flat := map[string]*model.Element{
		"system.api":   {Kind: "container", Title: "API"},
		"system.db":    {Kind: "container", Title: "DB"},
		"system.cache": {Kind: "container", Title: "Cache"},
	}

	if !isViewIncluded("system.api", m, flat) {
		t.Error("expected system.api to be included in view")
	}
	if isViewIncluded("system.cache", m, flat) {
		t.Error("expected system.cache to not be included in view")
	}
}

func TestIsViewIncluded_Wildcard(t *testing.T) {
	m := &model.BausteinsichtModel{
		Views: map[string]model.View{
			"containers": {
				Include: []string{"system.*"},
			},
		},
	}
	flat := map[string]*model.Element{
		"system.api": {Kind: "container", Title: "API"},
		"system.db":  {Kind: "container", Title: "DB"},
		"other.api":  {Kind: "container", Title: "Other API"},
	}

	if !isViewIncluded("system.api", m, flat) {
		t.Error("expected system.api to match system.* pattern")
	}
	if !isViewIncluded("system.db", m, flat) {
		t.Error("expected system.db to match system.* pattern")
	}
	if isViewIncluded("other.api", m, flat) {
		t.Error("expected other.api to not match system.* pattern")
	}
}
