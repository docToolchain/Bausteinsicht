package e2e

// TestXMIPipeline (#491) verifies the full XMI import → validate → sync pipeline
// using a minimal Enterprise Architect-compatible XMI 2.x fixture.
// The XMI importer creates elements but no views; the test verifies the import
// succeeded by checking the model with `find` and that sync creates the draw.io.

import (
	"os"
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
	runCLI(t, bin, dir, "import", "--from", "xmi", "model.xmi")

	// ── Step 2: validate ─────────────────────────────────────────────────
	runCLI(t, bin, dir, "validate")

	// ── Step 3: find — elements must be discoverable in the model ────────
	// The XMI importer creates no views, so export-diagram produces no output.
	// Verify the elements exist by querying the model directly.
	findOut := runCLI(t, bin, dir, "find", "api")
	if !strings.Contains(strings.ToLower(findOut), "api") {
		t.Errorf("find 'api' did not return a match after XMI import; output:\n%s", findOut)
	}

	// ── Step 4: sync → draw.io is created (no pages since no views) ──────
	runCLI(t, bin, dir, "sync")

	if _, err := os.Stat(filepath.Join(dir, "architecture.drawio")); os.IsNotExist(err) {
		t.Error("sync did not create architecture.drawio after XMI import")
	}
}
