package overlay

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func sampleMetrics() *MetricsFile {
	return &MetricsFile{
		Meta: MetaInfo{MetricDescriptions: map[string]string{"coverage": "Test coverage in percent"}},
		Metrics: []ElementMetric{
			{ElementID: "b", Values: map[string]float64{"error_rate": 0.9, "coverage": 80}},
			{ElementID: "a", Values: map[string]float64{"error_rate": 0.1}},
			{ElementID: "nope", Values: map[string]float64{"error_rate": 0.5}},
			{ElementID: "nope", Values: map[string]float64{"coverage": 10}},
		},
	}
}

// The metrics of a file are listed by name with range, element count and the
// description of the file's meta section (#696).
func TestSummaries(t *testing.T) {
	got := Summaries(sampleMetrics())
	want := []MetricSummary{
		{Name: "coverage", Description: "Test coverage in percent", Min: 10, Max: 80, Elements: 2},
		{Name: "error_rate", Min: 0.1, Max: 0.9, Elements: 3},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("want %+v, got %+v", want, got)
	}
}

func TestUnknownElementIDs(t *testing.T) {
	got := UnknownElementIDs(sampleMetrics(), map[string]bool{"a": true, "b": true})
	if !reflect.DeepEqual(got, []string{"nope"}) {
		t.Errorf("want [nope] once, got %v", got)
	}
}

// Without --metric the first metric by name is used, not a random map key.
func TestDefaultMetric_IsDeterministic(t *testing.T) {
	for i := 0; i < 50; i++ {
		if got := DefaultMetric(sampleMetrics()); got != "coverage" {
			t.Fatalf("want coverage (first by name), got %q", got)
		}
	}
	if got := DefaultMetric(&MetricsFile{}); got != "" {
		t.Errorf("a file without metrics has no default, got %q", got)
	}
}

// A file in another shape is refused with the shape that is expected (#696).
func TestLoadMetricsFile_WrongShapeNamesTheFormat(t *testing.T) {
	for name, content := range map[string]string{
		"per element":    `{"customer":{"error_rate":0.5}}`,
		"metrics object": `{"meta":{},"metrics":{"error_rate":0.5}}`,
		"empty list":     `{"meta":{},"metrics":[]}`,
	} {
		path := filepath.Join(t.TempDir(), "m.json")
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := LoadMetricsFile(path)
		if err == nil || !strings.Contains(err.Error(), `"elementId"`) || !strings.Contains(err.Error(), `"metrics"`) {
			t.Errorf("%s: want an error that shows the expected shape, got %v", name, err)
		}
	}
}
