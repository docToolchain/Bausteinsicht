// Package stale detects unused or forgotten architecture elements.
package stale

import (
	"time"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// RiskLevel describes the risk of removing a stale element.
type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

// StaleElement represents an element flagged as stale.
type StaleElement struct {
	ID                   string
	Title                string
	Kind                 string
	LastModified         time.Time
	DaysSinceModified    int
	MissingStatus        bool
	MissingADR           bool
	IncomingRelCount     int
	OutgoingRelCount     int
	IsViewIncluded       bool
	Risk                 RiskLevel
	Recommendations      []string
}

// StaleConfig controls stale element detection.
type StaleConfig struct {
	ThresholdDays int      // elements not touched in this many days
	ExcludeKinds  []string // never flag these element kinds
	ExcludeTags   []string // never flag elements with these tags
}

// DetectionResult holds the result of stale element detection.
type DetectionResult struct {
	StaleElements []StaleElement
	TotalElements int
	Timestamp     time.Time
}

// DefaultConfig returns the default stale detection configuration.
func DefaultConfig() StaleConfig {
	return StaleConfig{
		ThresholdDays: 90,
		ExcludeKinds:  []string{},
		ExcludeTags:   []string{},
	}
}

// LoadConfigFromModel extracts stale detection config from model metadata.
// Looks for a "staleDetection" key in model.Meta with optional fields:
//   thresholdDays: int
//   excludeKinds: []string
//   excludeTags: []string
func LoadConfigFromModel(m *model.BausteinsichtModel) StaleConfig {
	config := DefaultConfig()
	if m == nil || m.Meta == nil {
		return config
	}

	// Look for staleDetection config in metadata
	if staleDetVal, ok := m.Meta["staleDetection"]; ok {
		if staleDetMap, ok := staleDetVal.(map[string]interface{}); ok {
			// Extract thresholdDays
			if thresholdVal, ok := staleDetMap["thresholdDays"]; ok {
				if threshold, ok := thresholdVal.(float64); ok {
					config.ThresholdDays = int(threshold)
				}
			}

			// Extract excludeKinds
			if excludeKindsVal, ok := staleDetMap["excludeKinds"]; ok {
				if kindsSlice, ok := excludeKindsVal.([]interface{}); ok {
					config.ExcludeKinds = make([]string, 0, len(kindsSlice))
					for _, k := range kindsSlice {
						if kindStr, ok := k.(string); ok {
							config.ExcludeKinds = append(config.ExcludeKinds, kindStr)
						}
					}
				}
			}

			// Extract excludeTags
			if excludeTagsVal, ok := staleDetMap["excludeTags"]; ok {
				if tagsSlice, ok := excludeTagsVal.([]interface{}); ok {
					config.ExcludeTags = make([]string, 0, len(tagsSlice))
					for _, t := range tagsSlice {
						if tagStr, ok := t.(string); ok {
							config.ExcludeTags = append(config.ExcludeTags, tagStr)
						}
					}
				}
			}
		}
	}

	return config
}
