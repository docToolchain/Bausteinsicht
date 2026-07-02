package e2e

// TestSequenceDiagram (#500) verifies `export-sequence` produces PlantUML and
// Mermaid output for a model containing a dynamic view with sequence steps.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

func TestSequenceDiagram(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	runCLI(t, bin, dir, "init")

	// Add a dynamic view to the model.
	modelPath := filepath.Join(dir, "architecture.jsonc")
	m, err := model.Load(modelPath)
	if err != nil {
		t.Fatalf("model.Load: %v", err)
	}
	m.DynamicViews = []model.DynamicView{
		{
			Key:   "checkout-flow",
			Title: "Checkout Flow",
			Steps: []model.SequenceStep{
				{From: "customer", To: "onlineshop.frontend", Label: "place order"},
				{From: "onlineshop.frontend", To: "onlineshop.api", Label: "POST /orders"},
			},
		},
	}
	if err := model.Save(modelPath, m); err != nil {
		t.Fatalf("model.Save: %v", err)
	}

	t.Run("PlantUML", func(t *testing.T) {
		outDir := t.TempDir()
		runCLI(t, bin, dir, "export-sequence",
			"--model", "architecture.jsonc",
			"--diagram-format", "plantuml",
			"--output", outDir,
		)
		entries, err := os.ReadDir(outDir)
		if err != nil {
			t.Fatalf("read output dir: %v", err)
		}
		if len(entries) == 0 {
			t.Fatal("export-sequence plantuml: no files produced")
		}
		for _, e := range entries {
			content, _ := os.ReadFile(filepath.Join(outDir, e.Name()))
			if !strings.Contains(string(content), "checkout") &&
				!strings.Contains(strings.ToLower(string(content)), "sequence") &&
				!strings.Contains(string(content), "customer") {
				t.Errorf("PlantUML output %s missing expected content:\n%s", e.Name(), content)
			}
		}
	})

	t.Run("Mermaid", func(t *testing.T) {
		outDir := t.TempDir()
		runCLI(t, bin, dir, "export-sequence",
			"--model", "architecture.jsonc",
			"--diagram-format", "mermaid",
			"--output", outDir,
		)
		entries, err := os.ReadDir(outDir)
		if err != nil {
			t.Fatalf("read output dir: %v", err)
		}
		if len(entries) == 0 {
			t.Fatal("export-sequence mermaid: no files produced")
		}
		for _, e := range entries {
			content, _ := os.ReadFile(filepath.Join(outDir, e.Name()))
			if !strings.Contains(string(content), "checkout") &&
				!strings.Contains(strings.ToLower(string(content)), "sequence") &&
				!strings.Contains(string(content), "customer") {
				t.Errorf("Mermaid output %s missing expected content:\n%s", e.Name(), content)
			}
		}
	})

}
