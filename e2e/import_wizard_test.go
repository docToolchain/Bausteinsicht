package e2e

// TestImportWizard_* (#482 v1) exercises the post-import summary and
// next-steps hint added to `bausteinsicht import`: a per-kind element count,
// output path, generated view count, warnings, a --json machine-readable
// mode, and a next-steps hint that adapts to --dry-run / --force.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const wizardStructurizrDSL = `workspace "Wizard" "minimal repro for #482" {
    model {
        user = person "User" "A user."
        widgets = softwareSystem "Widgets" "Does widget things." {
            api = container "API" "REST API." "Go"
        }
        user -> api "Uses" "HTTPS"
    }
    views {
        systemContext widgets "SystemContext" {
            include *
            autoLayout lr
        }
    }
}
`

func writeWizardDSL(t *testing.T, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "workspace.dsl"), []byte(wizardStructurizrDSL), 0o600); err != nil {
		t.Fatalf("write workspace.dsl: %v", err)
	}
}

// TestImportWizard_TextSummary verifies the human-readable post-import
// summary: per-kind counts, output path, view count, and a next-steps hint
// pointing at sync/export-diagram for a fresh import.
func TestImportWizard_TextSummary(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeWizardDSL(t, dir)

	out := runCLI(t, bin, dir, "import", "workspace.dsl", "--from", "structurizr", "--output", "architecture.jsonc")

	for _, want := range []string{
		"person: 1",
		"container: 1",
		"system: 1",
		"architecture.jsonc",
		"views: 1",
		"bausteinsicht sync",
		"bausteinsicht export diagram",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected summary to contain %q, got:\n%s", want, out)
		}
	}
}

// TestImportWizard_JSONSummary verifies --json emits a machine-readable
// summary with the same information, including a present-but-zero "views"
// key semantics (tested indirectly via the non-zero case here; the
// XMI/omitted case is covered by TestImportWizard_XMINoViewsLine).
func TestImportWizard_JSONSummary(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeWizardDSL(t, dir)

	out := runCLI(t, bin, dir, "import", "workspace.dsl", "--from", "structurizr", "--output", "architecture.jsonc", "--json")

	var summary struct {
		Elements   map[string]int `json:"elements"`
		OutputPath string         `json:"outputPath"`
		Views      int            `json:"views"`
		Warnings   []string       `json:"warnings"`
	}
	if err := json.Unmarshal([]byte(out), &summary); err != nil {
		t.Fatalf("--json output is not valid JSON: %v\noutput:\n%s", err, out)
	}
	if summary.Elements["container"] != 1 {
		t.Errorf("expected 1 container, got %d (elements=%v)", summary.Elements["container"], summary.Elements)
	}
	if summary.OutputPath == "" {
		t.Error("expected non-empty outputPath")
	}
	if summary.Views != 1 {
		t.Errorf("expected views=1, got %d", summary.Views)
	}
}

// TestImportWizard_DryRunSuppressesHint verifies that --dry-run prints the
// model only, with no next-steps hint (nothing was persisted).
func TestImportWizard_DryRunSuppressesHint(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeWizardDSL(t, dir)

	out := runCLI(t, bin, dir, "import", "workspace.dsl", "--from", "structurizr", "--dry-run")

	if strings.Contains(out, "bausteinsicht sync") || strings.Contains(out, "next steps") {
		t.Errorf("--dry-run must not print a next-steps hint, got:\n%s", out)
	}
}

// TestImportWizard_ForceOverwriteHintsDiff verifies that re-importing over
// an existing output file with --force hints at `bausteinsicht diff` instead
// of `bausteinsicht sync`, since the existing file likely already has synced
// draw.io state.
func TestImportWizard_ForceOverwriteHintsDiff(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeWizardDSL(t, dir)

	runCLI(t, bin, dir, "import", "workspace.dsl", "--from", "structurizr", "--output", "architecture.jsonc")
	out := runCLI(t, bin, dir, "import", "workspace.dsl", "--from", "structurizr", "--output", "architecture.jsonc", "--force")

	if !strings.Contains(out, "bausteinsicht diff") {
		t.Errorf("expected --force overwrite to hint at \"bausteinsicht diff\", got:\n%s", out)
	}
	if strings.Contains(out, "bausteinsicht sync") {
		t.Errorf("--force overwrite must not also hint at \"bausteinsicht sync\", got:\n%s", out)
	}
}

// TestImportWizard_XMINoViewsLine verifies the views line is omitted in text
// mode when the importer produced zero views (XMI never populates Views),
// rather than printing a permanently-zero line.
func TestImportWizard_XMINoViewsLine(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	xmiSrc := filepath.Join(findModuleRoot(t), "internal/importer/xmi/testdata/basic.xmi")
	copyTestFile(t, xmiSrc, filepath.Join(dir, "model.xmi"))

	out := runCLI(t, bin, dir, "import", "model.xmi", "--from", "xmi", "--output", "architecture.jsonc")

	if strings.Contains(out, "views:") {
		t.Errorf("expected no \"views:\" line for an XMI import with zero views, got:\n%s", out)
	}
}
