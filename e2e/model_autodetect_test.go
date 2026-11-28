package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #692: health, graph, status, stale, diff, snapshot, overlay and adr did not
// find the model file themselves, the others did.
func TestCommandsAutoDetectTheModel(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")
	// A model that is not called architecture.jsonc must work as well.
	if err := os.Rename(filepath.Join(dir, "architecture.jsonc"), filepath.Join(dir, "my-model.jsonc")); err != nil {
		t.Fatal(err)
	}
	drawio := filepath.Join(dir, "overlay.drawio")
	if err := os.WriteFile(drawio, []byte(minimalDrawio), 0o644); err != nil {
		t.Fatal(err)
	}
	metrics := writeMetrics(t, dir, []metricEntry{{ElementID: "customer", Coverage: 0.2}})

	for _, args := range [][]string{
		{"validate"},
		{"health"},
		{"graph"},
		{"status"},
		{"stale"},
		{"adr", "list"},
		{"snapshot", "save", "autodetect"},
		{"overlay", "apply", "--output", drawio, "--metric", "coverage", metrics},
		{"overlay", "remove", "--output", drawio},
	} {
		out, code := runCLIAllowFail(t, bin, dir, args...)
		if code != 0 {
			t.Errorf("bausteinsicht %s: exit %d, want 0\n%s", strings.Join(args, " "), code, out)
		}
	}

	// diff needs asIs/toBe; the point is that it gets as far as reading the model.
	out, code := runCLIAllowFail(t, bin, dir, "diff")
	if code != 1 || !strings.Contains(out, "asIs and toBe") {
		t.Errorf("bausteinsicht diff: exit %d\n%s", code, out)
	}
}

func TestAutoDetectNamesWhatItLookedFor(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	for _, args := range [][]string{{"health"}, {"graph"}, {"status"}, {"stale"}, {"validate"}, {"overlay", "remove"}} {
		out, code := runCLIAllowFail(t, bin, dir, args...)
		if code != 2 || !strings.Contains(out, "no model file found") || !strings.Contains(out, "*.jsonc") || !strings.Contains(out, "--model") {
			t.Errorf("bausteinsicht %s without a model: exit %d\n%s", strings.Join(args, " "), code, out)
		}
		if strings.Contains(out, "reading :") || strings.Contains(out, "flag is required") {
			t.Errorf("bausteinsicht %s: unhelpful message\n%s", strings.Join(args, " "), out)
		}
	}
}
