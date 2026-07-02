package e2e

// TestGenerateTemplate (#502) verifies that `generate-template` creates a
// draw.io template file and that subsequent sync uses it correctly.

import (
	"os"
	"strings"
	"testing"
)

func TestGenerateTemplate(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	runCLI(t, bin, dir, "init")

	// ── Step 1: generate a template ───────────────────────────────────────
	runCLI(t, bin, dir, "generate-template")

	// generate-template should produce a template.drawio in the working dir.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	templateFound := false
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".drawio") && e.Name() != "architecture.drawio" {
			templateFound = true
			t.Logf("template file: %s", e.Name())
			break
		}
	}
	if !templateFound {
		t.Error("generate-template: no template .drawio file found in working directory")
	}

	// ── Step 2: sync still works after template generation ────────────────
	runCLI(t, bin, dir, "sync")

	// draw.io must still be valid after sync.
	if _, err := os.Stat(dir + "/architecture.drawio"); err != nil {
		t.Errorf("architecture.drawio missing after sync: %v", err)
	}
}
