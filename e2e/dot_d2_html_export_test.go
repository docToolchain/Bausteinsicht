package e2e

// TestDotD2HTMLExport verifies the three export-diagram formats that had no
// prior E2E coverage: dot (Graphviz), d2, and html (interactive HTML5
// viewer). All three share the handleNewFormats code path in
// cmd/bausteinsicht/export_diagram.go, distinct from the plantuml/mermaid
// path already covered by TestMultiFormatExportConsistency.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

func TestDotD2HTMLExport(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	runCLI(t, bin, dir, "init")

	m, err := model.Load(filepath.Join(dir, "architecture.jsonc"))
	if err != nil {
		t.Fatalf("model.Load: %v", err)
	}
	titlesByID := make(map[string]string)
	for id, elem := range m.Model {
		titlesByID[id] = elem.Title
	}

	t.Run("DOT", func(t *testing.T) {
		outDir := t.TempDir()
		runCLI(t, bin, dir, "export-diagram", "--diagram-format", "dot", "--output", outDir)
		files, _ := filepath.Glob(filepath.Join(outDir, "*.dot"))
		if len(files) == 0 {
			t.Fatal("no .dot files produced by export-diagram --diagram-format dot")
		}
		assertFilesContainTitles(t, "DOT", files, titlesByID)
	})

	t.Run("D2", func(t *testing.T) {
		outDir := t.TempDir()
		runCLI(t, bin, dir, "export-diagram", "--diagram-format", "d2", "--output", outDir)
		files, _ := filepath.Glob(filepath.Join(outDir, "*.d2"))
		if len(files) == 0 {
			t.Fatal("no .d2 files produced by export-diagram --diagram-format d2")
		}
		assertFilesContainTitles(t, "D2", files, titlesByID)
	})

	t.Run("HTML", func(t *testing.T) {
		outDir := t.TempDir()
		runCLI(t, bin, dir, "export-diagram", "--diagram-format", "html", "--output", outDir)
		files, _ := filepath.Glob(filepath.Join(outDir, "*.html"))
		if len(files) == 0 {
			t.Fatal("no .html files produced by export-diagram --diagram-format html")
		}
		assertFilesContainTitles(t, "HTML", files, titlesByID)

		// The HTML output is meant to be an interactive viewer — a bare title
		// dump without any markup would still pass a plain "contains title"
		// check, so also confirm it's actually HTML.
		data, err := os.ReadFile(files[0])
		if err != nil {
			t.Fatalf("read %s: %v", files[0], err)
		}
		if !strings.Contains(string(data), "<html") && !strings.Contains(string(data), "<!DOCTYPE") {
			t.Errorf("HTML export does not look like HTML (no <html>/<!DOCTYPE>):\n%.300s", data)
		}
	})

	t.Run("HTML_SingleView", func(t *testing.T) {
		// Single-view HTML export takes a different branch in
		// handleNewFormats than the multi-view loop exercised above.
		outDir := t.TempDir()
		var firstView string
		for key := range m.Views {
			firstView = key
			break
		}
		if firstView == "" {
			t.Skip("model has no views")
		}
		runCLI(t, bin, dir, "export-diagram",
			"--diagram-format", "html",
			"--view", firstView,
			"--output", outDir,
		)
		files, _ := filepath.Glob(filepath.Join(outDir, "*.html"))
		if len(files) != 1 {
			t.Fatalf("expected exactly 1 HTML file for single-view export, got %d", len(files))
		}
	})

	t.Run("JSONOutput", func(t *testing.T) {
		// --format json is supported for dot/d2/html too (handleNewFormats'
		// own JSON branch, separate from the multi-view file-writing branch).
		out := runCLI(t, bin, dir, "export-diagram",
			"--diagram-format", "dot",
			"--format", "json",
		)
		if !strings.Contains(out, `"format"`) || !strings.Contains(out, `"dot"`) {
			t.Errorf("export-diagram --diagram-format dot --format json: expected a format=dot entry, got: %s", out)
		}
	})
}
