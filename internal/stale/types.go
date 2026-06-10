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
	ID                string    `json:"id"`
	Title             string    `json:"title"`
	Kind              string    `json:"kind"`
	LastModified      time.Time `json:"lastModified"`
	DaysSinceModified int       `json:"daysSinceModified"`
	MissingStatus     bool      `json:"missingStatus"`
	MissingADR        bool      `json:"missingAdr"`
	IncomingRelCount  int       `json:"incomingRelCount"`
	OutgoingRelCount  int       `json:"outgoingRelCount"`
	IsViewIncluded    bool      `json:"isViewIncluded"`
	Risk              RiskLevel `json:"risk"`
	Recommendations   []string  `json:"recommendations"`
}

// StaleConfig controls stale element detection.
type StaleConfig struct {
	ThresholdDays int      // elements not touched in this many days
	ExcludeKinds  []string // never flag these element kinds
	ExcludeTags   []string // never flag elements with these tags
}

// DetectionResult holds the result of stale element detection.
type DetectionResult struct {
	StaleElements []StaleElement `json:"staleElements"`
	TotalElements int            `json:"totalElements"`
	Timestamp     time.Time      `json:"timestamp"`
}

// DefaultConfig returns the default stale detection configuration.
func DefaultConfig() StaleConfig {
	return StaleConfig{
		ThresholdDays: 90,
		ExcludeKinds:  []string{},
		ExcludeTags:   []string{},
	}
}

// metaInt reads an int from a JSON-decoded map (JSON numbers decode as float64).
func metaInt(m map[string]interface{}, key string, def int) int {
	v, ok := m[key]
	if !ok {
		return def
	}
	f, ok := v.(float64)
	if !ok {
		return def
	}
	return int(f)
}

// metaStringSlice reads a []string from a JSON-decoded map.
func metaStringSlice(m map[string]interface{}, key string) []string {
	v, ok := m[key]
	if !ok {
		return nil
	}
	items, ok := v.([]interface{})
	if !ok {
		return nil
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// LoadConfigFromModel extracts stale detection config from model metadata.
// Looks for a "staleDetection" key in model.Meta with optional fields:
//
//	thresholdDays: int
//	excludeKinds: []string
//	excludeTags: []string
func LoadConfigFromModel(m *model.BausteinsichtModel) StaleConfig {
	config := DefaultConfig()
	if m == nil || m.Meta == nil {
		return config
	}
	staleDetVal, ok := m.Meta["staleDetection"]
	if !ok {
		return config
	}
	staleDetMap, ok := staleDetVal.(map[string]interface{})
	if !ok {
		return config
	}
	config.ThresholdDays = metaInt(staleDetMap, "thresholdDays", config.ThresholdDays)
	if kinds := metaStringSlice(staleDetMap, "excludeKinds"); kinds != nil {
		config.ExcludeKinds = kinds
	}
	if tags := metaStringSlice(staleDetMap, "excludeTags"); tags != nil {
		config.ExcludeTags = tags
	}
	return config
}
