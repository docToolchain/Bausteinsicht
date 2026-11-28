package e2e

// TestLayoutVariants (#493) verifies that `layout` with different rank-dir
// settings produces valid draw.io output with positioned elements.
// Since only "hierarchical" is implemented, we check TB vs LR directions.

import (
	"os"
	"strings"
	"testing"
)

func TestLayoutVariants(t *testing.T) {
	for _, rankDir := range []string{"TB", "LR"} {
		rankDir := rankDir
		t.Run("RankDir_"+rankDir, func(t *testing.T) {
			bin := buildBinary(t)
			dir := t.TempDir()

			runCLI(t, bin, dir, "init")
			runCLI(t, bin, dir, "sync")

			runCLI(t, bin, dir, "layout",
				"--model", "architecture.jsonc",
				"--rank-dir", rankDir,
			)

			drawioBytes, err := os.ReadFile(dir + "/architecture.drawio")
			if err != nil {
				t.Fatalf("read draw.io after layout: %v", err)
			}
			content := string(drawioBytes)

			// After layout, draw.io must still contain element geometry.
			if !strings.Contains(content, "mxGeometry") {
				t.Errorf("layout %s: draw.io missing mxGeometry (no positions computed?)", rankDir)
			}
			// Elements must still be present.
			if !strings.Contains(content, "customer") {
				t.Errorf("layout %s: draw.io missing 'customer' element after layout", rankDir)
			}
		})
	}
}

// #691: an unknown --algorithm is rejected with exit code 1 and the list of
// valid values, and the diagram is not touched; --rank-dir already did this.
func TestLayoutRejectsUnknownAlgorithm(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")
	runCLI(t, bin, dir, "sync")
	before, err := os.ReadFile(dir + "/architecture.drawio")
	if err != nil {
		t.Fatal(err)
	}

	out, code := runCLIAllowFail(t, bin, dir, "layout", "--algorithm", "foo")
	if code != 1 {
		t.Fatalf("want exit code 1 for an unknown algorithm, got %d:\n%s", code, out)
	}
	if !strings.Contains(out, `"foo"`) || !strings.Contains(out, "hierarchical") {
		t.Errorf("the error must name the value and the valid one:\n%s", out)
	}
	after, _ := os.ReadFile(dir + "/architecture.drawio")
	if string(after) != string(before) {
		t.Error("the diagram changed although the layout was refused")
	}

	if out, code := runCLIAllowFail(t, bin, dir, "layout", "--algorithm", "hierarchical"); code != 0 {
		t.Errorf("--algorithm hierarchical must work (exit %d):\n%s", code, out)
	}
}
