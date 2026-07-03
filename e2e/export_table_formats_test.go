package e2e

// TestExportTableFormats verifies export-table's Markdown output
// (--table-format md) and per-view mode (without --combined). The existing
// TestMultiFormatExportConsistency (#486) only exercises the combined
// AsciiDoc path, leaving writeMarkdownTable/FormatView/CollectRows/
// collectAllRows with no E2E coverage.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportTableFormats(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	runCLI(t, bin, dir, "init")

	t.Run("Markdown_Combined", func(t *testing.T) {
		outDir := t.TempDir()
		runCLI(t, bin, dir, "export-table",
			"--table-format", "md",
			"--combined",
			"--output", outDir,
		)
		files, _ := filepath.Glob(filepath.Join(outDir, "*.md"))
		if len(files) == 0 {
			t.Fatal("no .md files produced by export-table --table-format md --combined")
		}
		data, err := os.ReadFile(files[0])
		if err != nil {
			t.Fatalf("read %s: %v", files[0], err)
		}
		content := string(data)
		if !strings.Contains(content, "customer") {
			t.Errorf("Markdown combined export missing \"customer\":\n%s", content)
		}
		// Markdown tables use pipe-delimited rows; AsciiDoc uses "|===" fences —
		// confirm this is genuinely Markdown, not an AsciiDoc file with a .md
		// extension (which would still pass a plain "contains customer" check).
		if !strings.Contains(content, "|") || strings.Contains(content, "|===") {
			t.Errorf("Markdown export does not look like a Markdown table:\n%s", content)
		}
	})

	t.Run("AllViews_NotCombined", func(t *testing.T) {
		// Without --combined and without --view, export-table writes a single
		// file with one titled section per view (FormatAllViews) — a
		// different code path than --combined's deduplicated single table
		// (FormatCombined) or --view's single-view table (FormatView).
		outDir := t.TempDir()
		runCLI(t, bin, dir, "export-table",
			"--table-format", "adoc",
			"--output", outDir,
		)
		files, _ := filepath.Glob(filepath.Join(outDir, "*.adoc"))
		if len(files) != 1 {
			t.Fatalf("expected exactly 1 file (all-views-elements.adoc), got %d: %v", len(files), files)
		}
		data, err := os.ReadFile(files[0])
		if err != nil {
			t.Fatalf("read %s: %v", files[0], err)
		}
		content := string(data)
		for _, wantTitle := range []string{"System Context", "Container View", "API Components"} {
			if !strings.Contains(content, wantTitle) {
				t.Errorf("all-views export missing view section %q:\n%s", wantTitle, content)
			}
		}
	})

	t.Run("SingleView", func(t *testing.T) {
		outDir := t.TempDir()
		runCLI(t, bin, dir, "export-table",
			"--table-format", "adoc",
			"--view", "context",
			"--output", outDir,
		)
		files, _ := filepath.Glob(filepath.Join(outDir, "*.adoc"))
		if len(files) != 1 {
			t.Fatalf("expected exactly 1 file for --view context, got %d: %v", len(files), files)
		}
		data, err := os.ReadFile(files[0])
		if err != nil {
			t.Fatalf("read %s: %v", files[0], err)
		}
		if !strings.Contains(string(data), "customer") {
			t.Errorf("single-view export-table missing \"customer\":\n%s", data)
		}
	})
}
