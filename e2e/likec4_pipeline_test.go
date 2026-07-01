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
	runCLI(t, bin, dir, "import", "--format", "likec4", "model.c4")

	// ── Step 2: validate ─────────────────────────────────────────────────
	runCLI(t, bin, dir, "validate")

	// ── Step 3: sync ─────────────────────────────────────────────────────
	runCLI(t, bin, dir, "sync")

	// ── Step 4: export as Mermaid ─────────────────────────────────────────
	out := runCLI(t, bin, dir, "export-diagram", "--format", "mermaid")

	// All top-level elements from the fixture must appear in the diagram output.
	for _, name := range []string{"Customer", "My Platform"} {
		if !strings.Contains(out, name) {
			t.Errorf("Mermaid export missing %q; output:\n%s", name, out)
		}
	}
}
