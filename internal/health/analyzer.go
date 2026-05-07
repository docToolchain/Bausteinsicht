package health

import (
	"fmt"
	"time"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// Analyzer computes health scores for a model.
type Analyzer struct {
	model *model.BausteinsichtModel
}

// NewAnalyzer creates a new health analyzer.
func NewAnalyzer(m *model.BausteinsichtModel) *Analyzer {
	return &Analyzer{model: m}
}

// Analyze computes a comprehensive health score.
func (a *Analyzer) Analyze() *HealthScore {
	flatElems, _ := model.FlattenElements(a.model)

	categories := []CategoryScore{
		a.scoreCompleteness(flatElems),
		a.scoreConformance(flatElems),
		a.scoreComplexity(flatElems),
		a.scoreDeprecation(flatElems),
		a.scoreDocumentation(flatElems),
	}

	// Calculate weighted overall score
	var totalScore float64
	var totalWeight float64
	for _, cat := range categories {
		totalScore += cat.Score * cat.Weight
		totalWeight += cat.Weight
	}

	overall := totalScore / totalWeight
	if totalWeight == 0 {
		overall = 0
	}

	return &HealthScore{
		Overall:    overall,
		Categories: categories,
		Grade:      calculateGrade(overall),
		Summary:    summarizeHealth(overall),
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		ElementCnt: len(flatElems),
		RelCnt:     len(a.model.Relationships),
		ViewCnt:    len(a.model.Views),
	}
}

// scoreCompleteness measures how well-documented the model is.
func (a *Analyzer) scoreCompleteness(elems map[string]*model.Element) CategoryScore {
	var findings []Finding
	documented := 0
	missing := 0

	for id, elem := range elems {
		if elem.Title == "" || (elem.Description == "" && elem.Technology == "") {
			findings = append(findings, Finding{
				Category: CategoryCompleteness,
				Severity: "minor",
				Title:    "Missing element description or technology",
				Message:  fmt.Sprintf("element %q lacks description or technology", id),
				Elements: []string{id},
			})
			missing++
		} else {
			documented++
		}
	}

	score := float64(documented) / float64(documented+missing) * 100
	if documented+missing == 0 {
		score = 100
	}

	return CategoryScore{
		Category: CategoryCompleteness,
		Score:    score,
		Weight:   0.2,
		Findings: findings,
		Details:  fmt.Sprintf("%d/%d elements documented", documented, documented+missing),
	}
}

// scoreConformance checks for policy violations.
func (a *Analyzer) scoreConformance(elems map[string]*model.Element) CategoryScore {
	var findings []Finding

	// Check for undefined relationship kinds
	for i, rel := range a.model.Relationships {
		if rel.Kind != "" {
			if _, ok := a.model.Specification.Relationships[rel.Kind]; !ok {
				findings = append(findings, Finding{
					Category: CategoryConformance,
					Severity: "major",
					Title:    "Undefined relationship kind",
					Message:  fmt.Sprintf("relationships[%d] uses unknown kind %q", i, rel.Kind),
					Elements: []string{rel.From, rel.To},
				})
			}
		}
	}

	// Check for undefined element kinds
	for id, elem := range elems {
		if _, ok := a.model.Specification.Elements[elem.Kind]; !ok {
			findings = append(findings, Finding{
				Category: CategoryConformance,
				Severity: "major",
				Title:    "Undefined element kind",
				Message:  fmt.Sprintf("element %q uses unknown kind %q", id, elem.Kind),
				Elements: []string{id},
			})
		}
	}

	score := 100.0 - float64(len(findings)*5)
	if score < 0 {
		score = 0
	}

	return CategoryScore{
		Category: CategoryConformance,
		Score:    score,
		Weight:   0.3,
		Findings: findings,
		Details:  fmt.Sprintf("%d violations found", len(findings)),
	}
}

// scoreComplexity assesses architectural complexity.
func (a *Analyzer) scoreComplexity(elems map[string]*model.Element) CategoryScore {
	var findings []Finding

	// Measure relationship density
	maxRels := len(elems) * (len(elems) - 1)
	if maxRels == 0 {
		maxRels = 1
	}
	density := float64(len(a.model.Relationships)) / float64(maxRels)

	// Count high-degree nodes (elements with many relationships)
	inDegree := make(map[string]int)
	outDegree := make(map[string]int)
	for _, rel := range a.model.Relationships {
		inDegree[rel.To]++
		outDegree[rel.From]++
	}

	for id, out := range outDegree {
		if out > 5 {
			findings = append(findings, Finding{
				Category: CategoryComplexity,
				Severity: "major",
				Title:    "High outgoing dependency count",
				Message:  fmt.Sprintf("element %q has %d outgoing relationships (threshold: 5)", id, out),
				Elements: []string{id},
			})
		}
	}

	// Score based on density and high-degree nodes
	score := 100.0
	if density > 0.3 {
		score -= 20
		findings = append(findings, Finding{
			Category: CategoryComplexity,
			Severity: "minor",
			Title:    "High relationship density",
			Message:  fmt.Sprintf("Architecture has %d relationships across %d elements (density: %.2f)", len(a.model.Relationships), len(elems), density),
		})
	}
	score -= float64(len(findings)) * 3

	if score < 0 {
		score = 0
	}

	return CategoryScore{
		Category: CategoryComplexity,
		Score:    score,
		Weight:   0.15,
		Findings: findings,
		Details:  fmt.Sprintf("density: %.2f, high-degree nodes: %d", density, len(findings)),
	}
}

// scoreDeprecation checks for deprecated elements still in use.
func (a *Analyzer) scoreDeprecation(elems map[string]*model.Element) CategoryScore {
	var findings []Finding
	deprecated := 0
	active := 0

	for id, elem := range elems {
		if elem.Status == model.StatusDeprecated {
			deprecated++
			findings = append(findings, Finding{
				Category: CategoryDeprecation,
				Severity: "major",
				Title:    "Deprecated element still present",
				Message:  fmt.Sprintf("element %q is marked as deprecated", id),
				Elements: []string{id},
			})
		} else if elem.Status == model.StatusDeployed || elem.Status == "" {
			active++
		}
	}

	score := 100.0 - float64(deprecated*10)
	if score < 0 {
		score = 0
	}

	return CategoryScore{
		Category: CategoryDeprecation,
		Score:    score,
		Weight:   0.15,
		Findings: findings,
		Details:  fmt.Sprintf("%d active, %d deprecated", active, deprecated),
	}
}

// scoreDocumentation measures the quality of documentation.
func (a *Analyzer) scoreDocumentation(elems map[string]*model.Element) CategoryScore {
	var findings []Finding
	withDocs := 0

	for id, elem := range elems {
		if elem.Description != "" && len(elem.Description) > 20 {
			withDocs++
		} else if elem.Description != "" {
			findings = append(findings, Finding{
				Category: CategoryDocumentation,
				Severity: "minor",
				Title:    "Brief element description",
				Message:  fmt.Sprintf("element %q has short description (< 20 chars)", id),
				Elements: []string{id},
			})
		}
	}

	score := (float64(withDocs) / float64(len(elems))) * 100
	if len(elems) == 0 {
		score = 100
	}

	return CategoryScore{
		Category: CategoryDocumentation,
		Score:    score,
		Weight:   0.2,
		Findings: findings,
		Details:  fmt.Sprintf("%d/%d elements have substantial descriptions", withDocs, len(elems)),
	}
}

// summarizeHealth creates a human-readable summary.
func summarizeHealth(score float64) string {
	switch {
	case score >= 90:
		return "Excellent architecture. Well-structured, documented, and maintainable."
	case score >= 80:
		return "Good architecture. Minor improvements recommended."
	case score >= 70:
		return "Acceptable architecture. Several areas need attention."
	case score >= 60:
		return "Fair architecture. Multiple improvements needed."
	default:
		return "Poor architecture. Significant refactoring recommended."
	}
}
