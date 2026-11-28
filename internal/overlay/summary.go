package overlay

import (
	"fmt"
	"sort"
)

// FormatHint describes the expected shape of a metrics file; it is part of the
// errors for a file in another shape and of the help of the overlay commands.
const FormatHint = `a metrics file is {"meta": {"source": "...", "generated": "...", "metric_descriptions": {"error_rate": "..."}}, ` +
	`"metrics": [{"elementId": "<model element ID>", "error_rate": 0.5}]}: one entry per element, ` +
	`the other keys of an entry are numeric metrics ("meta" is optional)`

// MetricSummary describes one metric of a metrics file.
type MetricSummary struct {
	Name        string
	Description string
	Min, Max    float64
	Elements    int
}

// Summaries lists the metrics of mf, sorted by name, with their range, the number
// of elements that have them and the description of the meta section.
func Summaries(mf *MetricsFile) []MetricSummary {
	byName := make(map[string]*MetricSummary)
	for _, em := range mf.Metrics {
		for name, v := range em.Values {
			s, ok := byName[name]
			if !ok {
				byName[name] = &MetricSummary{Name: name, Description: mf.Meta.MetricDescriptions[name], Min: v, Max: v, Elements: 1}
				continue
			}
			s.Min, s.Max = min(s.Min, v), max(s.Max, v)
			s.Elements++
		}
	}
	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]MetricSummary, 0, len(names))
	for _, name := range names {
		out = append(out, *byName[name])
	}
	return out
}

// UnknownElementIDs returns the sorted, distinct element IDs of mf that are not known.
func UnknownElementIDs(mf *MetricsFile, known map[string]bool) []string {
	seen := make(map[string]bool)
	var ids []string
	for _, em := range mf.Metrics {
		if !known[em.ElementID] && !seen[em.ElementID] {
			seen[em.ElementID] = true
			ids = append(ids, em.ElementID)
		}
	}
	sort.Strings(ids)
	return ids
}

// DefaultMetric is the metric that apply uses without --metric: the first one by
// name, or "" if the file has none.
func DefaultMetric(mf *MetricsFile) string {
	if s := Summaries(mf); len(s) > 0 {
		return s[0].Name
	}
	return ""
}

// shapeError wraps a problem with the shape of a metrics file with the expected shape.
func shapeError(problem string) error {
	return fmt.Errorf("%s; %s", problem, FormatHint)
}
