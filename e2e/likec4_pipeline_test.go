package e2e

// TestLikeC4Pipeline (#501) verifies the full LikeC4 import → validate → sync →
// export-diagram pipeline using the existing simple.c4 fixture.
// Checks that elements from the C4 model appear in the Mermaid export.

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLikeC4Pipeline(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	// Use existing LikeC4 test fixture (user + myPlatform + containers + views).
	c4Src := filepath.Join(findModuleRoot(t), "internal/importer/likec4/testdata/simple.c4")
	c4Dst := filepath.Join(dir, "model.c4")
	copyTestFile(t, c4Src, c4Dst)

	// ── Step 1: import ────────────────────────────────────────────────────
	// simple.c4: user (person), myPlatform (system), frontend/api/db (containers)
	runCLI(t, bin, dir, "import", "--from", "likec4", "model.c4")

	// ── Step 2: validate ─────────────────────────────────────────────────
	runCLI(t, bin, dir, "validate")

	// ── Step 3: sync ─────────────────────────────────────────────────────
	runCLI(t, bin, dir, "sync")

	// ── Step 4: export as Mermaid ─────────────────────────────────────────
	out := runCLI(t, bin, dir, "export-diagram", "--diagram-format", "mermaid")

	// The views are scoped to myPlatform (user/Customer is not in any view).
	// Verify the diagram output is non-empty and contains the system boundary.
	if strings.TrimSpace(out) == "" {
		t.Errorf("Mermaid export produced empty output")
	}
	// "My Platform" appears as a System_Boundary in the scoped view output.
	if !strings.Contains(out, "My Platform") && !strings.Contains(out, "myPlatform") {
		t.Errorf("Mermaid export missing 'My Platform' boundary; output:\n%s", out)
	}
	t.Logf("Mermaid export: %s", out)
}
