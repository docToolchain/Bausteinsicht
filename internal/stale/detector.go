package stale

import (
	"fmt"
	"sort"
	"time"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// Detect identifies stale elements in a model based on git history and metadata.
// Returns a list of stale elements sorted by risk level (high to low).
func Detect(m *model.BausteinsichtModel, modelPath string, config StaleConfig) (DetectionResult, error) {
	result := DetectionResult{
		Timestamp:     time.Now(),
		StaleElements: []StaleElement{},
	}

	if m == nil {
		return result, nil
	}

	// Get last modification date of the model file (fallback for elements without explicit lastModified)
	fileLastModified, err := GetLastModifiedDate(modelPath)
	if err != nil {
		// Don't fail if git integration has issues; just flag everything as potentially stale
		fileLastModified = time.Time{}
	}

	// Flatten the model to get all elements
	flatElements, err := model.FlattenElements(m)
	if err != nil {
		return result, fmt.Errorf("flattening model elements: %w", err)
	}
	result.TotalElements = len(flatElements)

	// Build relationship index for risk assessment
	relIndex := buildRelationshipIndex(m, flatElements)

	// Check each element for staleness
	for dotPath, elem := range flatElements {
		// Skip archived elements
		if elem.Status == model.StatusArchived {
			continue
		}

		// Skip excluded kinds
		if isExcluded(elem.Kind, config.ExcludeKinds) {
			continue
		}

		// Skip elements that carry any excluded tag
		if isTagExcluded(elem.Tags, config.ExcludeTags) {
			continue
		}

		// Determine element's last modified time (priority: explicit → git → file fallback).
		elemLastModified := resolveLastModified(elem, modelPath, dotPath, fileLastModified)

		// Check staleness criteria
		if !shouldFlag(elem, elemLastModified, config) {
			continue
		}

		// Build stale element record
		staleElem := StaleElement{
			ID:                dotPath,
			Title:             elem.Title,
			Kind:              elem.Kind,
			LastModified:      elemLastModified,
			DaysSinceModified: DaysSince(elemLastModified),
			MissingStatus:     elem.Status == "",
			MissingADR:        len(elem.Decisions) == 0,
			IncomingRelCount:  relIndex.incoming[dotPath],
			OutgoingRelCount:  relIndex.outgoing[dotPath],
			IsViewIncluded:    isViewIncluded(dotPath, m, flatElements),
		}

		// Assess risk
		staleElem.Risk = assessRisk(staleElem)
		staleElem.Recommendations = generateRecommendations(staleElem)

		result.StaleElements = append(result.StaleElements, staleElem)
	}

	sort.Slice(result.StaleElements, func(i, j int) bool {
		ri, rj := riskOrder(result.StaleElements[i].Risk), riskOrder(result.StaleElements[j].Risk)
		if ri != rj {
			return ri > rj
		}
		return result.StaleElements[i].ID < result.StaleElements[j].ID
	})

	return result, nil
}

func riskOrder(r RiskLevel) int {
	switch r {
	case RiskHigh:
		return 3
	case RiskMedium:
		return 2
	case RiskLow:
		return 1
	default:
		return 0
	}
}

// shouldFlag returns true if an element should be flagged as stale.
// lastModified is the element-level modification time (from git per-element
// search or file-level fallback).
func shouldFlag(elem *model.Element, lastModified time.Time, config StaleConfig) bool {
	// Criterion 1: Element not modified within threshold days
	if lastModified.IsZero() || !IsStale(lastModified, config.ThresholdDays) {
		return false
	}

	// Criterion 2: No status set
	if elem.Status != "" {
		return false
	}

	// Criterion 3: No ADR linked
	if len(elem.Decisions) > 0 {
		return false
	}

	return true
}

// resolveLastModified returns the best available modification time for an element.
// Priority: explicit lastModified field → file-level fallback.
// Per-element git search is intentionally omitted: leaf-name matching causes
// false negatives (elements sharing a leaf name alias each other) and runs
// O(n) git subprocesses for large models.
func resolveLastModified(elem *model.Element, _ string, _ string, fallback time.Time) time.Time {
	if elem.LastModified != "" {
		if t, err := time.Parse(time.RFC3339, elem.LastModified); err == nil {
			return t
		}
	}
	return fallback
}

// isExcluded checks if an element kind is in the exclusion list.
func isExcluded(kind string, excludeKinds []string) bool {
	for _, excluded := range excludeKinds {
		if kind == excluded {
			return true
		}
	}
	return false
}

// isTagExcluded returns true if any of the element's tags appear in excludeTags.
func isTagExcluded(elemTags []string, excludeTags []string) bool {
	if len(excludeTags) == 0 {
		return false
	}
	for _, tag := range elemTags {
		for _, excluded := range excludeTags {
			if tag == excluded {
				return true
			}
		}
	}
	return false
}

// isViewIncluded checks if an element is explicitly included in any view,
// using model.MatchPattern for consistent semantics with the rest of the tool.
// flatElements must contain dotPath (as returned by model.FlattenElements).
func isViewIncluded(dotPath string, m *model.BausteinsichtModel, flatElements map[string]*model.Element) bool {
	for _, view := range m.Views {
		for _, pattern := range view.Include {
			for _, id := range model.MatchPattern(flatElements, pattern) {
				if id == dotPath {
					return true
				}
			}
		}
	}
	return false
}

// assessRisk determines the removal risk of a stale element.
func assessRisk(staleElem StaleElement) RiskLevel {
	// High risk: explicitly in views AND has incoming relationships
	if staleElem.IsViewIncluded && staleElem.IncomingRelCount > 0 {
		return RiskHigh
	}

	// Medium risk: in a view (visible in a published diagram) OR other elements depend on it
	if staleElem.IsViewIncluded || staleElem.IncomingRelCount > 0 {
		return RiskMedium
	}

	// Low risk: not in any view and no incoming relationships
	return RiskLow
}

// generateRecommendations creates actionable recommendations for a stale element.
func generateRecommendations(staleElem StaleElement) []string {
	var recommendations []string

	// Recommendation 1: Set status
	if staleElem.MissingStatus {
		if staleElem.IncomingRelCount == 0 {
			recommendations = append(recommendations, "Set status to \"archived\" if no longer needed")
		} else {
			recommendations = append(recommendations, "Set status to \"deprecated\" if still in use")
		}
	}

	// Recommendation 2: Link ADR if missing
	if staleElem.MissingADR {
		recommendations = append(recommendations, "Link an ADR documenting the decision/design")
	}

	// Recommendation 3: Review relationships
	if staleElem.IncomingRelCount > 0 {
		recommendations = append(recommendations, "Review incoming relationships before archiving")
	}

	return recommendations
}

// relationshipIndex tracks incoming and outgoing relationships for each element.
type relationshipIndex struct {
	incoming map[string]int
	outgoing map[string]int
}

// buildRelationshipIndex creates an index of incoming/outgoing relationships.
func buildRelationshipIndex(m *model.BausteinsichtModel, flatElements map[string]*model.Element) relationshipIndex {
	index := relationshipIndex{
		incoming: make(map[string]int),
		outgoing: make(map[string]int),
	}

	for _, rel := range m.Relationships {
		// Check if both ends exist in the model
		if _, exists := flatElements[rel.From]; !exists {
			continue
		}
		if _, exists := flatElements[rel.To]; !exists {
			continue
		}

		index.outgoing[rel.From]++
		index.incoming[rel.To]++
	}

	return index
}
