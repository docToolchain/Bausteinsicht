package e2e

// TestADRWorkflow verifies the `adr list`/`adr show` commands: decisions
// defined in specification.decisions and linked to an element via its
// `decisions` field are listed (optionally filtered by element) and can be
// shown individually. Covers a CLI command with no prior E2E coverage.

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

func TestADRWorkflow(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	runCLI(t, bin, dir, "init")

	modelPath := filepath.Join(dir, "architecture.jsonc")
	m, err := model.Load(modelPath)
	if err != nil {
		t.Fatalf("model.Load: %v", err)
	}
	m.Specification.Decisions = []model.DecisionRecord{
		{ID: "ADR-001", Title: "Use Go for the CLI", Status: model.ADRActive, Date: "2026-01-01", FilePath: "docs/ADR-001.adoc"},
		{ID: "ADR-002", Title: "JSONC as the model format", Status: model.ADRProposed},
	}
	cust := m.Model["customer"]
	cust.Decisions = []string{"ADR-001"}
	m.Model["customer"] = cust
	if err := model.Save(modelPath, m); err != nil {
		t.Fatalf("model.Save: %v", err)
	}

	// ── adr list: all decisions ────────────────────────────────────────────────
	listOut := runCLI(t, bin, dir, "adr", "list", "--model", "architecture.jsonc")
	if !strings.Contains(listOut, "ADR-001") || !strings.Contains(listOut, "ADR-002") {
		t.Errorf("adr list: expected both ADR-001 and ADR-002, got: %s", listOut)
	}

	// ── adr list --element: only decisions linked to that element ─────────────
	filteredOut := runCLI(t, bin, dir, "adr", "list", "--model", "architecture.jsonc", "--element", "customer")
	if !strings.Contains(filteredOut, "ADR-001") {
		t.Errorf("adr list --element customer: expected ADR-001, got: %s", filteredOut)
	}
	if strings.Contains(filteredOut, "ADR-002") {
		t.Errorf("adr list --element customer: unexpected ADR-002 (not linked to customer), got: %s", filteredOut)
	}

	// ── adr list --format json ──────────────────────────────────────────────────
	jsonOut := runCLI(t, bin, dir, "adr", "list", "--model", "architecture.jsonc", "--format", "json")
	var decisions []model.DecisionRecord
	if err := json.Unmarshal([]byte(jsonOut), &decisions); err != nil {
		t.Fatalf("adr list --format json: invalid JSON: %v\noutput: %s", err, jsonOut)
	}
	if len(decisions) != 2 {
		t.Errorf("adr list --format json: expected 2 decisions, got %d: %+v", len(decisions), decisions)
	}

	// ── adr show <id> ────────────────────────────────────────────────────────
	showOut := runCLI(t, bin, dir, "adr", "show", "ADR-001", "--model", "architecture.jsonc")
	if !strings.Contains(showOut, "ADR-001") || !strings.Contains(showOut, "Use Go for the CLI") {
		t.Errorf("adr show ADR-001: expected ID and title in output, got: %s", showOut)
	}

	// ── adr show <unknown-id>: non-zero exit ───────────────────────────────────
	_, code := runCLIAllowFail(t, bin, dir, "adr", "show", "ADR-999", "--model", "architecture.jsonc")
	if code == 0 {
		t.Error("adr show ADR-999: expected non-zero exit for unknown decision ID")
	}
}
