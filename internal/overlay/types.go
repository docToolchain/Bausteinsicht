package overlay

type MetricsFile struct {
	Meta    MetaInfo       `json:"meta"`
	Metrics []ElementMetric `json:"metrics"`
}

type MetaInfo struct {
	Generated           string            `json:"generated"`
	Source              string            `json:"source"`
	MetricDescriptions  map[string]string `json:"metric_descriptions"`
}

type ElementMetric struct {
	ElementID string             `json:"elementId"`
	Values    map[string]float64 `json:"values,flatten"`
}

type NormalizedMetric struct {
	ElementID string
	Value     float64
}

type ColorScheme struct {
	Green  string
	Yellow string
	Orange string
	Red    string
}

var DefaultColorScheme = ColorScheme{
	Green:  "#d5e8d4",
	Yellow: "#fff2cc",
	Orange: "#ffe6cc",
	Red:    "#f8cecc",
}
