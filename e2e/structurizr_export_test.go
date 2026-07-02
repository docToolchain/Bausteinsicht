package e2e

// TestStructurizrDSLExport verifies `export-diagram --diagram-format
// structurizr` writes the whole workspace as a single Structurizr DSL file.
// Covers a code path with no prior E2E coverage (distinct from `import
// --from structurizr`, which reads DSL rather than writing it).

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStructurizrDSLExport(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	runCLI(t, bin, dir, "init")

	outDir := t.TempDir()
	runCLI(t, bin, dir, "export-diagram",
		"--model", "architecture.jsonc",
		"--diagram-format", "structurizr",
		"--output", outDir,
	)

	dslPath := filepath.Join(outDir, "workspace.dsl")
	data, err := os.ReadFile(dslPath)
	if err != nil {
		t.Fatalf("read exported workspace.dsl: %v", err)
	}
	dsl := string(data)

	for _, want := range []string{"workspace", "model", "views", "customer"} {
		if !strings.Contains(dsl, want) {
			t.Errorf("exported workspace.dsl missing %q:\n%s", want, dsl)
		}
	}

	// --view is not supported with structurizr format (exports the whole
	// workspace only) — must be rejected, not silently ignored.
	_, code := runCLIAllowFail(t, bin, dir, "export-diagram",
		"--model", "architecture.jsonc",
		"--diagram-format", "structurizr",
		"--view", "context",
	)
	if code == 0 {
		t.Error("export-diagram --diagram-format structurizr --view context: expected rejection (exit != 0)")
	}

	// Round-trip: the exported DSL should itself import back cleanly (the
	// export/import pair should be at least structurally consistent).
	reimportDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(reimportDir, "workspace.dsl"), data, 0o600); err != nil {
		t.Fatalf("write workspace.dsl for re-import: %v", err)
	}
	runCLI(t, bin, reimportDir, "import", "workspace.dsl", "--from", "structurizr", "--output", "reimported.jsonc")
	reimported, err := os.ReadFile(filepath.Join(reimportDir, "reimported.jsonc"))
	if err != nil {
		t.Fatalf("read reimported.jsonc: %v", err)
	}
	if !strings.Contains(string(reimported), "customer") {
		t.Errorf("reimported model missing \"customer\" after export->import round-trip:\n%s", reimported)
	}
}
