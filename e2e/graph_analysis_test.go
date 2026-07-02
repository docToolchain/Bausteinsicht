package e2e

// TestGraphAnalysis (#498) verifies `graph` command output after init+sync:
// cycle detection and centrality metrics are reported without errors.

import (
	"strings"
	"testing"
)

func TestGraphAnalysis(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	runCLI(t, bin, dir, "init")
	runCLI(t, bin, dir, "sync")

	t.Run("CyclesOnly", func(t *testing.T) {
		out := runCLI(t, bin, dir, "graph",
			"--model", "architecture.jsonc",
			"--cycles-only",
		)
		// Default model has no cycles — output should say so explicitly or be empty.
		lower := strings.ToLower(out)
		if strings.TrimSpace(out) != "" && !strings.Contains(lower, "no cycle") && !strings.Contains(lower, "cycle") {
			t.Errorf("graph --cycles-only: unexpected output (expected cycle summary or empty):\n%s", out)
		}
		t.Logf("graph --cycles-only: %s", out)
	})

	t.Run("WithCentrality", func(t *testing.T) {
		out := runCLI(t, bin, dir, "graph",
			"--model", "architecture.jsonc",
			"--centrality",
		)
		// Should list elements and some numeric centrality values.
		if strings.TrimSpace(out) == "" {
			t.Error("graph --centrality produced empty output")
		}
		t.Logf("graph --centrality: %s", out)
	})
}
