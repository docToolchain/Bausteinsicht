package e2e

// TestSnapshotLifecycle (#485) exercises the full snapshot command family:
//
//  1. init — model created
//  2. snapshot save --message baseline — baseline snapshot created
//  3. add element --id newService — new element in model
//  4. snapshot list --output-format json — exactly 1 snapshot
//  5. snapshot diff <id> — new element appears as "added"
//  6. snapshot restore <id> architecture.jsonc --force — model reverts
//  7. verify newService is absent from the restored model

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

func TestSnapshotLifecycle(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	// ── Step 1: create initial model ──────────────────────────────────────────
	runCLI(t, bin, dir, "init")

	// ── Step 2: save baseline snapshot ────────────────────────────────────────
	saveOut := runCLI(t, bin, dir, "snapshot", "save", "--message", "baseline")
	t.Logf("snapshot save output: %s", saveOut)

	// ── Step 3: add a new element to the model ────────────────────────────────
	// (No sync needed — snapshot diff compares against the JSONC model, not draw.io)
	runCLI(t, bin, dir, "add", "element",
		"--id", "newService",
		"--kind", "system",
		"--title", "New Service",
	)

	// Verify newService was added to the model file.
	m, err := model.Load(filepath.Join(dir, "architecture.jsonc"))
	if err != nil {
		t.Fatalf("model.Load after add element: %v", err)
	}
	flat, err := model.FlattenElements(m)
	if err != nil {
		t.Fatalf("FlattenElements after add element: %v", err)
	}
	if _, ok := flat["newService"]; !ok {
		t.Fatal("newService not found in model after add element")
	}

	// ── Step 4: snapshot list → exactly 1 snapshot ────────────────────────────
	listOut := runCLI(t, bin, dir, "snapshot", "list", "--output-format", "json")
	t.Logf("snapshot list output: %s", listOut)
	var snapshots []struct {
		ID      string `json:"id"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(listOut), &snapshots); err != nil {
		t.Fatalf("parse snapshot list JSON: %v\noutput: %s", err, listOut)
	}
	if len(snapshots) != 1 {
		t.Fatalf("expected 1 snapshot, got %d: %s", len(snapshots), listOut)
	}
	snapID := snapshots[0].ID
	if snapshots[0].Message != "baseline" {
		t.Errorf("snapshot message = %q, want %q", snapshots[0].Message, "baseline")
	}
	t.Logf("baseline snapshot ID: %s", snapID)

	// ── Step 5: snapshot diff → new element shows as added ────────────────────
	diffOut := runCLI(t, bin, dir, "snapshot", "diff", snapID)
	t.Logf("snapshot diff output:\n%s", diffOut)
	if !strings.Contains(diffOut, "newService") {
		t.Errorf("snapshot diff output does not mention 'newService':\n%s", diffOut)
	}

	// ── Step 6: restore baseline snapshot ────────────────────────────────────
	runCLI(t, bin, dir, "snapshot", "restore", snapID, "architecture.jsonc", "--force")

	// ── Step 7: verify newService is gone from restored model ─────────────────
	m2, err2 := model.Load(filepath.Join(dir, "architecture.jsonc"))
	if err2 != nil {
		t.Fatalf("model.Load after restore: %v", err2)
	}
	flat2, err2 := model.FlattenElements(m2)
	if err2 != nil {
		t.Fatalf("FlattenElements after restore: %v", err2)
	}
	if _, ok := flat2["newService"]; ok {
		t.Error("newService still present in model after snapshot restore")
	}

	t.Log("snapshot lifecycle OK: save → diff → restore verified")
}
