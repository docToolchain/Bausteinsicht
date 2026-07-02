package e2e

// TestMultiFormatExportConsistency (#486) verifies that all text-based export
// formats (PlantUML, Mermaid, AsciiDoc table) produce consistent output from
// the same model: every top-level element visible in a view must appear in each
// format's output.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

func TestMultiFormatExportConsistency(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	// ── Setup: init creates model + draw.io ───────────────────────────────────
	runCLI(t, bin, dir, "init")

	// Load model to know which elements to expect.
	m, err := model.Load(filepath.Join(dir, "architecture.jsonc"))
	if err != nil {
		t.Fatalf("model.Load: %v", err)
	}

	// Collect top-level element titles (they appear in the landscape view).
	titlesByID := make(map[string]string)
	for id, elem := range m.Model {
		titlesByID[id] = elem.Title
	}

	outDir := filepath.Join(dir, "exports")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("mkdir exports: %v", err)
	}

	// ── PlantUML export ───────────────────────────────────────────────────────
	runCLI(t, bin, dir, "export-diagram",
		"--diagram-format", "plantuml",
		"--output", outDir,
	)

	pumlFiles, _ := filepath.Glob(filepath.Join(outDir, "*.puml"))
	if len(pumlFiles) == 0 {
		t.Error("no .puml files produced by export-diagram")
	} else {
		t.Logf("PlantUML: %d file(s) exported", len(pumlFiles))
		assertFilesContainTitles(t, "PlantUML", pumlFiles, titlesByID)
	}

	// ── Mermaid export ────────────────────────────────────────────────────────
	runCLI(t, bin, dir, "export-diagram",
		"--diagram-format", "mermaid",
		"--output", outDir,
	)

	// export-diagram --diagram-format mermaid produces .mmd files.
	mmdFiles, _ := filepath.Glob(filepath.Join(outDir, "*.mmd"))
	if len(mmdFiles) == 0 {
		t.Error("no .mmd files produced by export-diagram --diagram-format mermaid")
	} else {
		t.Logf("Mermaid: %d file(s) exported", len(mmdFiles))
		assertFilesContainTitles(t, "Mermaid", mmdFiles, titlesByID)
	}

	// ── AsciiDoc table export ─────────────────────────────────────────────────
	tableDir := filepath.Join(dir, "tables")
	_ = os.MkdirAll(tableDir, 0o755)
	runCLI(t, bin, dir, "export-table",
		"--table-format", "adoc",
		"--combined",
		"--output", tableDir,
	)

	adocFiles, _ := filepath.Glob(filepath.Join(tableDir, "*.adoc"))
	if len(adocFiles) == 0 {
		t.Error("no .adoc files produced by export-table")
	} else {
		t.Logf("AsciiDoc table: %d file(s) exported", len(adocFiles))
		assertFilesContainTitles(t, "AsciiDoc table", adocFiles, titlesByID)
	}

	// ── Consistency: PlantUML file count == Mermaid file count ───────────────
	if len(pumlFiles) != len(mmdFiles) {
		t.Errorf("format consistency: PlantUML=%d files, Mermaid=%d files — counts differ",
			len(pumlFiles), len(mmdFiles))
	}

	t.Log("multi-format export consistency OK")
}

// assertFilesContainTitles checks that at least one file contains each element title.
func assertFilesContainTitles(t *testing.T, format string, files []string, titlesByID map[string]string) {
	t.Helper()
	var combined strings.Builder
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Errorf("read %s: %v", f, err)
			continue
		}
		combined.Write(data)
	}
	content := combined.String()

	for id, title := range titlesByID {
		if !strings.Contains(content, title) {
			t.Errorf("%s export: element %q (title %q) not found in any output file", format, id, title)
		}
	}
}
