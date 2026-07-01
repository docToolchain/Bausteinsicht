package e2e

// TestOverlayHeatmap (#488) exercises the overlay apply/remove lifecycle.
//
// Sub-test "BareDrawio": uses a hand-crafted draw.io with direct <mxCell> children
// to verify the core overlay logic (apply, check, remove, check).
//
// Sub-test "BausteinsichtOutput": uses real init+sync output, where elements are
// wrapped in <object bausteinsicht_id="..."> tags. Verifies that overlay correctly
// matches on bausteinsicht_id and applies/removes colors on the inner mxCell.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const minimalDrawio = `<?xml version="1.0" encoding="UTF-8"?>
<mxfile>
  <diagram name="test">
    <mxGraphModel>
      <root>
        <mxCell id="0"/>
        <mxCell id="1" parent="0"/>
        <mxCell id="customer" value="Customer" style="rounded=1;fillColor=#dae8fc;" vertex="1" parent="1">
          <mxGeometry x="100" y="100" width="120" height="60" as="geometry"/>
        </mxCell>
        <mxCell id="api" value="API" style="rounded=1;fillColor=#d5e8d4;" vertex="1" parent="1">
          <mxGeometry x="300" y="100" width="120" height="60" as="geometry"/>
        </mxCell>
      </root>
    </mxGraphModel>
  </diagram>
</mxfile>`

func TestOverlayHeatmap(t *testing.T) {
	t.Run("BareDrawio", testOverlayBareDrawio)
	t.Run("BausteinsichtOutput", testOverlayBausteinsichtOutput)
}

func testOverlayBareDrawio(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	// init to get a valid model file (required by overlay cmd)
	runCLI(t, bin, dir, "init")

	overlayDrawio := filepath.Join(dir, "overlay-test.drawio")
	if err := os.WriteFile(overlayDrawio, []byte(minimalDrawio), 0o644); err != nil {
		t.Fatalf("write overlay-test.drawio: %v", err)
	}

	metricsPath := writeMetrics(t, dir, []metricEntry{
		{ElementID: "customer", Coverage: 0.2},
		{ElementID: "api", Coverage: 0.9},
	})

	runCLI(t, bin, dir,
		"overlay", "apply",
		"--model", "architecture.jsonc",
		"--output", overlayDrawio,
		"--metric", "coverage",
		metricsPath,
	)

	afterApply := readFile(t, overlayDrawio)
	if !strings.Contains(afterApply, "data-original-fill") {
		t.Error("overlay apply: expected 'data-original-fill' attribute in draw.io XML")
	}

	runCLI(t, bin, dir,
		"overlay", "remove",
		"--model", "architecture.jsonc",
		"--output", overlayDrawio,
	)

	afterRemove := readFile(t, overlayDrawio)
	if strings.Contains(afterRemove, "data-original-fill") {
		t.Error("overlay remove: 'data-original-fill' still present after remove")
	}
}

// testOverlayBausteinsichtOutput verifies overlay works on real bausteinsicht sync output,
// where draw.io elements are <object bausteinsicht_id="..."><mxCell .../></object>.
// The overlay matches on bausteinsicht_id, so metrics use model element IDs directly.
func testOverlayBausteinsichtOutput(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	// init creates architecture.jsonc + architecture.drawio with <object>-wrapped elements.
	runCLI(t, bin, dir, "init")

	drawioPath := filepath.Join(dir, "architecture.drawio")

	// The default model's top-level element is "customer"; use its model ID (bausteinsicht_id).
	metricsPath := writeMetrics(t, dir, []metricEntry{
		{ElementID: "customer", Coverage: 0.1},
	})

	runCLI(t, bin, dir,
		"overlay", "apply",
		"--model", "architecture.jsonc",
		"--output", drawioPath,
		"--metric", "coverage",
		metricsPath,
	)

	afterApply := readFile(t, drawioPath)
	if !strings.Contains(afterApply, "data-original-fill") {
		t.Error("overlay apply on bausteinsicht output: expected 'data-original-fill' on inner mxCell")
	}

	runCLI(t, bin, dir,
		"overlay", "remove",
		"--model", "architecture.jsonc",
		"--output", drawioPath,
	)

	afterRemove := readFile(t, drawioPath)
	if strings.Contains(afterRemove, "data-original-fill") {
		t.Error("overlay remove on bausteinsicht output: 'data-original-fill' still present after remove")
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────────

type metricEntry struct {
	ElementID string  `json:"elementId"`
	Coverage  float64 `json:"coverage"`
}

type metricsFileJSON struct {
	Meta struct {
		Source             string            `json:"source"`
		Generated          string            `json:"generated"`
		MetricDescriptions map[string]string `json:"metric_descriptions"`
	} `json:"meta"`
	Metrics []metricEntry `json:"metrics"`
}

func writeMetrics(t *testing.T, dir string, entries []metricEntry) string {
	t.Helper()
	var mf metricsFileJSON
	mf.Meta.Source = "e2e-test"
	mf.Meta.Generated = "2026-01-01T00:00:00Z"
	mf.Meta.MetricDescriptions = map[string]string{"coverage": "Test coverage (0–1)"}
	mf.Metrics = entries
	data, err := json.MarshalIndent(mf, "", "  ")
	if err != nil {
		t.Fatalf("marshal metrics: %v", err)
	}
	path := filepath.Join(dir, "metrics.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write metrics.json: %v", err)
	}
	return path
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
