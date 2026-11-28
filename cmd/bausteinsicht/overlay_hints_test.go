package main

import (
	"os"
	"strings"
	"testing"
)

const overlayTestDrawio = `<?xml version="1.0" encoding="UTF-8"?>
<mxfile><diagram name="t"><mxGraphModel><root>
<mxCell id="0"/><mxCell id="1" parent="0"/>
<mxCell id="customer" value="Customer" style="fillColor=#dae8fc;" vertex="1" parent="1"><mxGeometry x="1" y="1" width="10" height="10" as="geometry"/></mxCell>
</root></mxGraphModel></diagram></mxfile>`

const overlayTestMetrics = `{"meta":{},"metrics":[
 {"elementId":"customer","error_rate":0.5,"coverage":70},
 {"elementId":"nope","error_rate":1,"coverage":10}]}`

func overlayProject(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	if out, err := executeImportCmd("init"); err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}
	if err := os.WriteFile("architecture.drawio", []byte(overlayTestDrawio), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("met.json", []byte(overlayTestMetrics), 0o600); err != nil {
		t.Fatal(err)
	}
}

// overlay list names every metric with its range and does not print an empty
// "from:  ()" header (#696).
func TestOverlayList_NamesTheMetrics(t *testing.T) {
	overlayProject(t)
	out, err := executeImportCmd("overlay", "list", "met.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"coverage", "error_rate", "min 10", "max 70", "min 0.5", "max 1", "2 elements"} {
		if !strings.Contains(out, want) {
			t.Errorf("want %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "()") || strings.Contains(out, "from:  ") {
		t.Errorf("no empty source/date in the header:\n%s", out)
	}

	out, err = executeImportCmd("overlay", "list", "met.json", "--format", "json")
	if err != nil || !strings.Contains(out, `"metrics"`) || !strings.Contains(out, `"error_rate"`) {
		t.Errorf("the JSON output lists the metrics too, got %q (%v)", out, err)
	}
}

// A metric for an element that is not in the model is reported, not ignored (#696).
func TestOverlayApply_WarnsAboutUnknownElements(t *testing.T) {
	overlayProject(t)
	out, err := executeImportCmd("overlay", "apply", "met.json", "--metric", "error_rate")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(out, `"nope"`) || !strings.Contains(strings.ToLower(out), "warning") {
		t.Errorf("want a warning that names nope:\n%s", out)
	}
}

// Without --metric the choice is the first by name, whatever the map order (#696).
func TestOverlayApply_DefaultMetricIsTheFirstByName(t *testing.T) {
	overlayProject(t)
	for i := 0; i < 20; i++ {
		out, err := executeImportCmd("overlay", "apply", "met.json")
		if err != nil || !strings.Contains(out, "metric: coverage") {
			t.Fatalf("run %d: want metric coverage, got %q (%v)", i, out, err)
		}
	}
}

// The help describes the metrics file (#696).
func TestOverlayHelp_DescribesTheFormat(t *testing.T) {
	out, err := executeImportCmd("overlay", "apply", "--help")
	if err != nil || !strings.Contains(out, `"elementId"`) {
		t.Errorf("overlay apply --help must show the expected file shape:\n%s (%v)", out, err)
	}
}
