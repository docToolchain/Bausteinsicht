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
