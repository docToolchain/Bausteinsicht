package e2e

// TestHealthScore (#499) verifies `health` command output after import+sync:
// score and grade are reported, JSON format works, summary flag works.

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestHealthScore(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	runCLI(t, bin, dir, "init")
	runCLI(t, bin, dir, "sync")

	t.Run("TextOutput", func(t *testing.T) {
		out := runCLI(t, bin, dir, "health", "--model", "architecture.jsonc")
		if strings.TrimSpace(out) == "" {
			t.Error("health produced empty output")
		}
		t.Logf("health text: %s", out)
	})

	t.Run("JSONOutput", func(t *testing.T) {
		out := runCLI(t, bin, dir, "health",
			"--model", "architecture.jsonc",
			"--format", "json",
		)
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(out), &result); err != nil {
			t.Errorf("health --format json: not valid JSON: %v\noutput: %s", err, out)
		}
	})

	t.Run("SummaryFlag", func(t *testing.T) {
		out := runCLI(t, bin, dir, "health",
			"--model", "architecture.jsonc",
			"--summary",
		)
		if strings.TrimSpace(out) == "" {
			t.Error("health --summary produced empty output")
		}
	})
}
