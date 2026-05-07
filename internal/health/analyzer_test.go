package health

import (
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

func TestAnalyzerCompleteness(t *testing.T) {
	m := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"system": {Notation: "box"},
			},
		},
		Model: map[string]model.Element{
			"backend": {
				Kind:        "system",
				Title:       "Backend",
				Description: "Backend service",
			},
			"db": {
				Kind:  "system",
				Title: "Database",
				// Missing description
			},
		},
		Relationships: []model.Relationship{},
	}

	a := NewAnalyzer(m)
	score := a.Analyze()

	if score.Overall == 0 {
		t.Fatal("score should not be zero")
	}

	completenessScore := score.Categories[0]
	if completenessScore.Category != CategoryCompleteness {
		t.Errorf("first category should be completeness, got %v", completenessScore.Category)
	}

	if completenessScore.Score == 100 {
		t.Error("completeness score should be less than 100 due to missing description")
	}

	if len(completenessScore.Findings) == 0 {
		t.Error("should have findings for missing documentation")
	}
}

func TestAnalyzerGrade(t *testing.T) {
	tests := []struct {
		score    float64
		expected string
	}{
		{97, "A+"},
		{93, "A"},
		{90, "B+"},
		{87, "B"},
		{80, "C+"},
		{70, "C"},
		{60, "D"},
		{50, "F"},
	}

	for _, tt := range tests {
		grade := calculateGrade(tt.score)
		if grade != tt.expected {
			t.Errorf("calculateGrade(%.0f) = %q, want %q", tt.score, grade, tt.expected)
		}
	}
}

func TestAnalyzerConformance(t *testing.T) {
	m := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"system": {Notation: "box"},
			},
			Relationships: map[string]model.RelationshipKind{
				"uses": {Notation: "arrow"},
			},
		},
		Model: map[string]model.Element{
			"a": {Kind: "system", Title: "A"},
			"b": {Kind: "unknown", Title: "B"}, // Invalid kind
		},
		Relationships: []model.Relationship{
			{From: "a", To: "b", Kind: "calls"}, // Invalid kind
		},
	}

	a := NewAnalyzer(m)
	score := a.Analyze()

	conformanceScore := score.Categories[1]
	if conformanceScore.Score == 100 {
		t.Error("conformance score should be less than 100 for violations")
	}
	if len(conformanceScore.Findings) == 0 {
		t.Error("should have conformance findings")
	}
}

func TestAnalyzerDeprecation(t *testing.T) {
	m := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"system": {Notation: "box"},
			},
		},
		Model: map[string]model.Element{
			"active": {Kind: "system", Title: "Active", Status: model.StatusDeployed},
			"old":    {Kind: "system", Title: "Old", Status: model.StatusDeprecated},
		},
		Relationships: []model.Relationship{},
	}

	a := NewAnalyzer(m)
	score := a.Analyze()

	depScore := score.Categories[3]
	if depScore.Category != CategoryDeprecation {
		t.Errorf("expected deprecation category at index 3, got %v", depScore.Category)
	}
	if depScore.Score == 100 {
		t.Error("deprecation score should be less than 100 for deprecated elements")
	}
}

func TestAnalyzerEmptyModel(t *testing.T) {
	m := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements:      make(map[string]model.ElementKind),
			Relationships: make(map[string]model.RelationshipKind),
		},
		Model:         make(map[string]model.Element),
		Relationships: []model.Relationship{},
	}

	a := NewAnalyzer(m)
	score := a.Analyze()

	if score.ElementCnt != 0 {
		t.Errorf("empty model should have 0 elements, got %d", score.ElementCnt)
	}
}

func TestSummarizeHealth(t *testing.T) {
	tests := []struct {
		score    float64
		contains string
	}{
		{95, "Excellent"},
		{85, "Good"},
		{75, "Acceptable"},
		{65, "Fair"},
		{50, "Poor"},
	}

	for _, tt := range tests {
		summary := summarizeHealth(tt.score)
		if !contains(summary, tt.contains) {
			t.Errorf("summarizeHealth(%.0f) = %q, should contain %q", tt.score, summary, tt.contains)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || len(s) > len(substr) && s[len(s)-len(substr):] == substr || len(s) > len(substr) && (func() bool {
		for i := 0; i <= len(s)-len(substr); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	})()
}
