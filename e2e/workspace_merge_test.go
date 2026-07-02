package e2e

// TestWorkspaceMerge (#497) verifies that `workspace merge` combines two
// separate models into a single unified model, and `workspace list` shows both.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkspaceMerge(t *testing.T) {
	bin := buildBinary(t)

	// ── Set up two model directories ──────────────────────────────────────
	dirA := t.TempDir()
	dirB := t.TempDir()
	mergeDir := t.TempDir()

	runCLI(t, bin, dirA, "init")
	runCLI(t, bin, dirB, "init")

	// ── Write a workspace config referencing both models ──────────────────
	type workspaceConfig struct {
		Workspace struct {
			Name string `json:"name"`
		} `json:"workspace"`
		Models []struct {
			ID     string `json:"id"`
			Path   string `json:"path"`
			Prefix string `json:"prefix"`
		} `json:"models"`
	}

	cfg := workspaceConfig{}
	cfg.Workspace.Name = "test-workspace"
	cfg.Models = []struct {
		ID     string `json:"id"`
		Path   string `json:"path"`
		Prefix string `json:"prefix"`
	}{
		{ID: "team-a", Path: filepath.Join(dirA, "architecture.jsonc"), Prefix: "a"},
		{ID: "team-b", Path: filepath.Join(dirB, "architecture.jsonc"), Prefix: "b"},
	}

	cfgData, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("marshal workspace config: %v", err)
	}
	cfgPath := filepath.Join(mergeDir, "workspace.json")
	if err := os.WriteFile(cfgPath, cfgData, 0o644); err != nil {
		t.Fatalf("write workspace config: %v", err)
	}

	// ── workspace list ────────────────────────────────────────────────────
	listOut := runCLI(t, bin, mergeDir, "workspace", "list", cfgPath)
	if !strings.Contains(listOut, "team-a") || !strings.Contains(listOut, "team-b") {
		t.Errorf("workspace list: expected both 'team-a' and 'team-b'\noutput: %s", listOut)
	}

	// ── workspace merge ───────────────────────────────────────────────────
	outputPath := filepath.Join(mergeDir, "merged.jsonc")
	runCLI(t, bin, mergeDir, "workspace", "merge", cfgPath, outputPath)

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("workspace merge: output file not created")
	}

	// Merged model must contain elements from both teams (prefixed).
	mergedBytes, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read merged model: %v", err)
	}
	merged := string(mergedBytes)
	// Default model has "customer" — workspace merge prefixes IDs with "<prefix>_<id>",
	// so team-a (prefix "a") contributes "a_customer".
	if !strings.Contains(merged, "a_customer") {
		t.Errorf("merged model missing 'a_customer' (team-a prefix + customer element)\ncontent: %.500s", merged)
	}
}
