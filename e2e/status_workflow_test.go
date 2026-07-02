package e2e

// TestStatusWorkflow verifies `status` lists elements by lifecycle status
// and supports --filter and --format json. Covers a CLI command with no
// prior E2E coverage.

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

func TestStatusWorkflow(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	runCLI(t, bin, dir, "init")

	modelPath := filepath.Join(dir, "architecture.jsonc")
	m, err := model.Load(modelPath)
	if err != nil {
		t.Fatalf("model.Load: %v", err)
	}
	cust := m.Model["customer"]
	cust.Status = model.StatusDeprecated
	m.Model["customer"] = cust
	if err := model.Save(modelPath, m); err != nil {
		t.Fatalf("model.Save: %v", err)
	}

	// ── status: all elements, unset ones included ──────────────────────────────
	allOut := runCLI(t, bin, dir, "status", "--model", "architecture.jsonc")
	if !strings.Contains(allOut, "customer") {
		t.Errorf("status: expected element \"customer\" in output, got: %s", allOut)
	}

	// ── status --filter deprecated: only the one element we marked ────────────
	filteredOut := runCLI(t, bin, dir, "status", "--model", "architecture.jsonc", "--filter", "deprecated")
	if !strings.Contains(filteredOut, "customer") {
		t.Errorf("status --filter deprecated: expected \"customer\", got: %s", filteredOut)
	}

	// ── status --filter <invalid>: rejected ─────────────────────────────────────
	_, code := runCLIAllowFail(t, bin, dir, "status", "--model", "architecture.jsonc", "--filter", "not-a-real-status")
	if code == 0 {
		t.Error("status --filter not-a-real-status: expected non-zero exit for invalid filter")
	}

	// ── status --format json: verify the actual summary counts, not just "is valid JSON" ──
	jsonOut := runCLI(t, bin, dir, "status", "--model", "architecture.jsonc", "--format", "json")
	var result struct {
		Summary  map[string]int `json:"summary"`
		Elements []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"elements"`
	}
	if err := json.Unmarshal([]byte(jsonOut), &result); err != nil {
		t.Fatalf("status --format json: invalid JSON: %v\noutput: %s", err, jsonOut)
	}
	if result.Summary["deprecated"] != 1 {
		t.Errorf("status json: expected summary[\"deprecated\"] == 1, got %d (summary: %v)", result.Summary["deprecated"], result.Summary)
	}
	foundCustomer := false
	for _, e := range result.Elements {
		if e.ID == "customer" && e.Status == "deprecated" {
			foundCustomer = true
			break
		}
	}
	if !foundCustomer {
		t.Errorf("status json: expected element \"customer\" with status \"deprecated\", got: %+v", result.Elements)
	}
}
