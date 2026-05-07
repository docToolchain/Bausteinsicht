package overlay

import (
	"cmp"
	"slices"
)

func IsMetricBetter(metricName string) bool {
	goodMetrics := map[string]bool{
		"coverage":       true,
		"uptime":         true,
		"deploy_freq":    true,
		"success_rate":   true,
		"availability":   true,
	}
	badMetrics := map[string]bool{
		"error_rate":     true,
		"latency":        true,
		"p99":            true,
		"p99_ms":         true,
		"response_time":  true,
		"cpu_usage":      true,
		"memory_usage":   true,
		"error_count":    true,
		"failures":       true,
	}

	if goodMetrics[metricName] {
		return true
	}
	if badMetrics[metricName] {
		return false
	}
	return false
}

func Normalize(metrics []NormalizedMetric, higherIsBetter bool) map[string]float64 {
	if len(metrics) == 0 {
		return make(map[string]float64)
	}

	values := make([]float64, len(metrics))
	for i, m := range metrics {
		values[i] = m.Value
	}

	min := slices.Min(values)
	max := slices.Max(values)
	span := max - min

	result := make(map[string]float64)
	for _, m := range metrics {
		normalized := 0.0
		if span > 0 {
			normalized = (m.Value - min) / span
		}
		if !higherIsBetter {
			normalized = 1 - normalized
		}
		result[m.ElementID] = normalized
	}
	return result
}

func ColorForValue(normalized float64, scheme ColorScheme) string {
	switch {
	case normalized < 0.25:
		return scheme.Green
	case normalized < 0.50:
		return scheme.Yellow
	case normalized < 0.75:
		return scheme.Orange
	default:
		return scheme.Red
	}
}

func ExtractMetric(metrics []ElementMetric, metricKey string) ([]NormalizedMetric, error) {
	result := make([]NormalizedMetric, 0, len(metrics))
	for _, m := range metrics {
		if val, ok := m.Values[metricKey]; ok {
			result = append(result, NormalizedMetric{
				ElementID: m.ElementID,
				Value:     val,
			})
		}
	}
	slices.SortFunc(result, func(a, b NormalizedMetric) int {
		return cmp.Compare(a.ElementID, b.ElementID)
	})
	return result, nil
}
