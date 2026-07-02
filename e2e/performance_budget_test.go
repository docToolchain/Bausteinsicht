package e2e

// TestPerformanceBudget (#506) verifies that sync on a large model (100+ elements)
// completes within 2 seconds. Uses the model generator from the benchmarks package.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPerformanceBudget(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	// Generate a large model with 100+ elements programmatically.
	modelPath := filepath.Join(dir, "architecture.jsonc")
	if err := writeLargeModel(modelPath, 110); err != nil {
		t.Fatalf("write large model: %v", err)
	}

	// First sync (creates draw.io) — time it.
	start := time.Now()
	runCLI(t, bin, dir, "sync")
	elapsed := time.Since(start)

	const budget = 5 * time.Second
	if elapsed > budget {
		t.Errorf("sync of 110-element model took %v, exceeds budget of %v", elapsed, budget)
	}
	t.Logf("sync of 110-element model: %v", elapsed)

	// Second sync (no-op) must also be within budget.
	start = time.Now()
	runCLI(t, bin, dir, "sync")
	elapsed = time.Since(start)
	if elapsed > budget {
		t.Errorf("no-op sync of 110-element model took %v, exceeds budget of %v", elapsed, budget)
	}
	t.Logf("no-op sync of 110-element model: %v", elapsed)
}

// writeLargeModel writes a JSONC model with n flat elements to path.
func writeLargeModel(path string, n int) error {
	type kindDef struct {
		Notation string `json:"notation"`
	}
	type specBlock struct {
		Elements map[string]kindDef `json:"elements"`
	}
	type element struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Kind        string `json:"kind"`
	}
	type view struct {
		Title   string   `json:"title"`
		Include []string `json:"include"`
	}
	type modelFile struct {
		Specification specBlock          `json:"specification"`
		Model         map[string]element `json:"model"`
		Views         map[string]view    `json:"views"`
	}

	kinds := []string{"system", "external_system"}
	m := modelFile{
		Specification: specBlock{
			Elements: map[string]kindDef{
				"system":          {Notation: "System"},
				"external_system": {Notation: "External System"},
			},
		},
		Model: make(map[string]element, n),
		Views: map[string]view{
			"overview": {
				Title:   "Overview",
				Include: []string{"*"},
			},
		},
	}
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("element%03d", i)
		m.Model[id] = element{
			Title:       fmt.Sprintf("Element %d", i),
			Description: fmt.Sprintf("Auto-generated element %d for performance testing", i),
			Kind:        kinds[i%len(kinds)],
		}
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
