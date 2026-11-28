package overlay

// MetricSummary describes one metric of a metrics file.
type MetricSummary struct {
	Name        string
	Description string
	Min, Max    float64
	Elements    int
}

// Summaries lists the metrics of mf, sorted by name (stub, filled in by the fix).
func Summaries(mf *MetricsFile) []MetricSummary { return nil }

// UnknownElementIDs returns the sorted element IDs of mf that are not known (stub).
func UnknownElementIDs(mf *MetricsFile, known map[string]bool) []string { return nil }

// DefaultMetric is the metric that apply uses without --metric (stub).
func DefaultMetric(mf *MetricsFile) string { return "" }

// FormatHint describes the expected shape of a metrics file (stub).
const FormatHint = ""
