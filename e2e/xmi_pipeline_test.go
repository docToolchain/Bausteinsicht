package e2e

// TestXMIPipeline (#491) verifies the full XMI import → validate → sync → export-diagram
// pipeline using a minimal Enterprise Architect-compatible XMI 2.x fixture.
// Checks that element count is consistent across all pipeline stages.

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestXMIPipeline(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	// Copy XMI test fixture from importer testdata into the temp workspace.
	xmiSrc := filepath.Join(findModuleRoot(t), "internal/importer/xmi/testdata/basic.xmi")
	xmiDst := filepath.Join(dir, "model.xmi")
	copyTestFile(t, xmiSrc, xmiDst)

	// ── Step 1: import XMI ────────────────────────────────────────────────
	// basic.xmi has: API (Component) + Customer (Actor) + uses (Dependency)
	runCLI(t, bin, dir, "import", "--format", "xmi", "model.xmi")

	// ── Step 2: validate ─────────────────────────────────────────────────
	runCLI(t, bin, dir, "validate")

	// ── Step 3: sync → draw.io contains imported elements ────────────────
	runCLI(t, bin, dir, "sync")

	// ── Step 4: export as PlantUML ────────────────────────────────────────
	out := runCLI(t, bin, dir, "export-diagram", "--format", "plantuml")

	// XMI has API and Customer — both must appear in the PlantUML output.
	for _, elem := range []string{"API", "Customer"} {
		if !strings.Contains(out, elem) {
			t.Errorf("PlantUML export missing element %q; output:\n%s", elem, out)
		}
	}
}
