package stale

import (
	"encoding/json"
	"fmt"
	"strings"
)

// FormatText returns a human-readable text representation of stale elements.
func FormatText(result DetectionResult) string {
	if len(result.StaleElements) == 0 {
		return fmt.Sprintf("No stale elements found (%d total elements checked)", result.TotalElements)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Stale Elements (%d found, %d total)\n", len(result.StaleElements), result.TotalElements)
	sb.WriteString("==================================================\n\n")

	for _, elem := range result.StaleElements {
		// Element header
		fmt.Fprintf(&sb, "%-20s [%s]   Last changed: %d days ago\n",
			elem.ID, elem.Kind, elem.DaysSinceModified)

		// Status and ADR
		if elem.MissingStatus {
			sb.WriteString("                     No lifecycle status set\n")
		} else {
			sb.WriteString("                     Status set\n")
		}

		if elem.MissingADR {
			sb.WriteString("                     No ADR linked\n")
		}

		// Risk assessment
		riskIcon := riskIcon(elem.Risk)
		if elem.IncomingRelCount > 0 {
			fmt.Fprintf(&sb, "                     %s Has %d incoming relationships — may still be active\n",
				riskIcon, elem.IncomingRelCount)
		} else {
			fmt.Fprintf(&sb, "                     %s No incoming relationships — safe to archive\n", riskIcon)
		}

		sb.WriteString("\n")
	}

	// Summary
	highRiskCount := countByRisk(result.StaleElements, RiskHigh)
	mediumRiskCount := countByRisk(result.StaleElements, RiskMedium)
	lowRiskCount := countByRisk(result.StaleElements, RiskLow)

	fmt.Fprintf(&sb, "Risk summary: %d high, %d medium, %d low\n\n",
		highRiskCount, mediumRiskCount, lowRiskCount)

	// Recommendations
	sb.WriteString("Suggested actions:\n")
	sb.WriteString("  - Set status: \"deprecated\" or \"archived\" if no longer needed\n")
	sb.WriteString("  - Link ADR if the element has documented decisions\n")
	sb.WriteString("  - Review incoming relationships before archiving\n")

	return sb.String()
}

// FormatJSON returns a JSON representation of stale elements.
func FormatJSON(result DetectionResult) (string, error) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// riskIcon returns a visual representation of risk level.
func riskIcon(risk RiskLevel) string {
	switch risk {
	case RiskHigh:
		return "🔴"
	case RiskMedium:
		return "🟡"
	case RiskLow:
		return "✅"
	default:
		return "❓"
	}
}

// countByRisk counts elements with a specific risk level.
func countByRisk(elements []StaleElement, risk RiskLevel) int {
	count := 0
	for _, elem := range elements {
		if elem.Risk == risk {
			count++
		}
	}
	return count
}
