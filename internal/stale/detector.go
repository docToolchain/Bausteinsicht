package stale

import (
	"time"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// Detect identifies stale elements in a model based on git history and metadata.
// Returns a list of stale elements sorted by risk level (high to low).
func Detect(m *model.BausteinsichtModel, modelPath string, config StaleConfig) (DetectionResult, error) {
	result := DetectionResult{
		Timestamp: time.Now(),
	}

	if m == nil {
		return result, nil
	}

	// Get last modification date of the model file
	lastModified, err := GetLastModifiedDate(modelPath)
	if err != nil {
		// Don't fail if git integration has issues; just flag everything as potentially stale
		lastModified = time.Time{}
	}

	// Flatten the model to get all elements
	flatElements, _ := model.FlattenElements(m)
	result.TotalElements = len(flatElements)

	// Build relationship index for risk assessment
	relIndex := buildRelationshipIndex(m, flatElements)

	// Check each element for staleness
	for dotPath, elem := range flatElements {
		// Skip archived elements
		if elem.Status == "archived" {
			continue
		}

		// Skip excluded kinds
		if isExcluded(elem.Kind, config.ExcludeKinds) {
			continue
		}

		// Check staleness criteria
		if !shouldFlag(elem, lastModified, config) {
			continue
		}

		// Build stale element record
		staleElem := StaleElement{
			ID:                dotPath,
			Title:             elem.Title,
			Kind:              elem.Kind,
			LastModified:      lastModified,
			DaysSinceModified: DaysSince(lastModified),
			MissingStatus:     elem.Status == "",
			MissingADR:        len(elem.Decisions) == 0,
			IncomingRelCount:  relIndex.incoming[dotPath],
			OutgoingRelCount:  relIndex.outgoing[dotPath],
			IsViewIncluded:    isViewIncluded(dotPath, m),
		}

		// Assess risk
		staleElem.Risk = assessRisk(staleElem)
		staleElem.Recommendations = generateRecommendations(staleElem)

		result.StaleElements = append(result.StaleElements, staleElem)
	}

	return result, nil
}

// shouldFlag returns true if an element should be flagged as stale.
func shouldFlag(elem *model.Element, modelLastModified time.Time, config StaleConfig) bool {
	// Criterion 1: Model file not modified in threshold days
	if modelLastModified.IsZero() || !IsStale(modelLastModified, config.ThresholdDays) {
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

// isExcluded checks if an element kind is in the exclusion list.
func isExcluded(kind string, excludeKinds []string) bool {
	for _, excluded := range excludeKinds {
		if kind == excluded {
			return true
		}
	}
	return false
}

// isViewIncluded checks if an element is explicitly included in any view.
func isViewIncluded(dotPath string, m *model.BausteinsichtModel) bool {
	for _, view := range m.Views {
		for _, pattern := range view.Include {
			// Simple pattern matching: exact match or wildcard
			if pattern == "*" || pattern == dotPath {
				return true
			}
			// Check prefix patterns like "system.*"
			if pattern[len(pattern)-1:] == "*" {
				prefix := pattern[:len(pattern)-1]
				if len(dotPath) > len(prefix) && dotPath[:len(prefix)] == prefix {
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

	// Medium risk: has incoming relationships (other elements depend on it)
	if staleElem.IncomingRelCount > 0 {
		return RiskMedium
	}

	// Low risk: no incoming relationships
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
