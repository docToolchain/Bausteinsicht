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
func LoadConfigFromModel(m *model.BausteinsichtModel) StaleConfig {
	config := DefaultConfig()
	if m == nil || m.Meta == nil {
		return config
	}

	// Check for staleDetection config in metadata
	// This would be parsed from model.Meta if it contains staleDetection settings
	// For now, return default config
	return config
}
