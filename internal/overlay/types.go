package overlay

import (
	"encoding/json"
)

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
	Values    map[string]float64
}

func (em *ElementMetric) UnmarshalJSON(data []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	em.Values = make(map[string]float64)
	for key, val := range raw {
		if key == "elementId" {
			if str, ok := val.(string); ok {
				em.ElementID = str
			}
		} else if num, ok := val.(float64); ok {
			em.Values[key] = num
		}
	}
	return nil
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
