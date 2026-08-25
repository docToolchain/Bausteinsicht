package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/spf13/cobra"
)

func executeImportCmd(args ...string) (string, error) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return buf.String(), err
}

// absDSL returns the absolute path to a testdata DSL file (avoids .. in paths
// which the security validator rejects).
//
// IMPORTANT: This helper assumes tests run with working directory in cmd/bausteinsicht/.
// Tests must be run with: go test ./cmd/bausteinsicht or make test
func absDSL(t *testing.T, parts ...string) string {
	t.Helper()
	rel := filepath.Join(parts...)
	abs, err := filepath.Abs(filepath.Join("..", "..", rel))
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	return abs
}

func TestImportCmd_Structurizr_DryRun(t *testing.T) {
	dsl := absDSL(t, "internal", "importer", "structurizr", "testdata", "simple.dsl")
	out, err := executeImportCmd("import", "--from", "structurizr", "--dry-run", dsl)
	if err != nil {
		t.Fatalf("expected no error, got %v\nOutput: %s", err, out)
	}
	if !strings.Contains(out, `"model"`) {
		t.Errorf("expected JSON model in output, got: %s", out)
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &m); err != nil {
		t.Errorf("output is not valid JSON: %v\nOutput: %s", err, out)
	}
}

func TestImportCmd_LikeC4_DryRun(t *testing.T) {
	c4 := absDSL(t, "internal", "importer", "likec4", "testdata", "simple.c4")
	out, err := executeImportCmd("import", "--from", "likec4", "--dry-run", c4)
	if err != nil {
		t.Fatalf("expected no error, got %v\nOutput: %s", err, out)
	}
	if !strings.Contains(out, `"model"`) {
		t.Errorf("expected JSON model in output, got: %s", out)
	}
}

func TestImportCmd_OutputFile(t *testing.T) {
	dsl := absDSL(t, "internal", "importer", "structurizr", "testdata", "simple.dsl")
	outFile := filepath.Join(t.TempDir(), "architecture.jsonc")
	out, err := executeImportCmd("import", "--from", "structurizr", "--output", outFile, dsl)
	if err != nil {
		t.Fatalf("expected no error, got %v\nOutput: %s", err, out)
	}
	if !strings.Contains(out, "Imported model written to") {
		t.Errorf("expected success message, got: %s", out)
	}
	if _, err := os.Stat(outFile); err != nil {
		t.Errorf("output file not created: %v", err)
	}
}

func TestImportCmd_OutputExists_ExitCode2(t *testing.T) {
	dsl := absDSL(t, "internal", "importer", "structurizr", "testdata", "simple.dsl")
	outFile := filepath.Join(t.TempDir(), "architecture.jsonc")
	if err := os.WriteFile(outFile, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := executeImportCmd("import", "--from", "structurizr", "--output", outFile, dsl)
	if err == nil {
		t.Fatal("expected error for existing output file")
	}
	ee, ok := err.(*exitError)
	if !ok {
		t.Fatalf("expected exitError, got %T: %v", err, err)
	}
	if ee.code != 2 {
		t.Errorf("expected exit code 2, got %d", ee.code)
	}
}

func TestImportCmd_Force_OverwritesFile(t *testing.T) {
	dsl := absDSL(t, "internal", "importer", "structurizr", "testdata", "simple.dsl")
	outFile := filepath.Join(t.TempDir(), "architecture.jsonc")
	if err := os.WriteFile(outFile, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := executeImportCmd("import", "--from", "structurizr", "--output", outFile, "--force", dsl)
	if err != nil {
		t.Fatalf("expected no error with --force, got %v", err)
	}
}

func TestImportCmd_UnknownFormat_ExitCode1(t *testing.T) {
	_, err := executeImportCmd("import", "--from", "unknown", "--dry-run", "anyfile.dsl")
	if err == nil {
		t.Fatal("expected error for unknown format")
	}
	ee, ok := err.(*exitError)
	if !ok {
		t.Fatalf("expected exitError, got %T: %v", err, err)
	}
	if ee.code != 1 {
		t.Errorf("expected exit code 1, got %d", ee.code)
	}
}

func TestImportCmd_NonExistentFile_ExitCode1(t *testing.T) {
	_, err := executeImportCmd("import", "--from", "structurizr", "--dry-run", "/nonexistent/model.dsl")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
	ee, ok := err.(*exitError)
	if !ok {
		t.Fatalf("expected exitError, got %T: %v", err, err)
	}
	if ee.code != 1 {
		t.Errorf("expected exit code 1, got %d", ee.code)
	}
}

// TestImportCmd_TextSummary covers the post-import summary text (#482 v1):
// per-kind counts, views count, and the fresh-import next-steps hint.
func TestImportCmd_TextSummary(t *testing.T) {
	dsl := absDSL(t, "internal", "importer", "structurizr", "testdata", "simple.dsl")
	outFile := filepath.Join(t.TempDir(), "architecture.jsonc")
	out, err := executeImportCmd("import", "--from", "structurizr", "--output", outFile, dsl)
	if err != nil {
		t.Fatalf("expected no error, got %v\nOutput: %s", err, out)
	}
	for _, want := range []string{
		"person: 1",
		"container: 3",
		"system: 2",
		"views: 2",
		"Next steps: bausteinsicht sync, bausteinsicht export diagram",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected summary to contain %q, got:\n%s", want, out)
		}
	}
}

// TestImportCmd_JSONSummary covers --json's machine-readable summary shape.
func TestImportCmd_JSONSummary(t *testing.T) {
	dsl := absDSL(t, "internal", "importer", "structurizr", "testdata", "simple.dsl")
	outFile := filepath.Join(t.TempDir(), "architecture.jsonc")
	out, err := executeImportCmd("import", "--from", "structurizr", "--output", outFile, "--json", dsl)
	if err != nil {
		t.Fatalf("expected no error, got %v\nOutput: %s", err, out)
	}

	var summary struct {
		Elements   map[string]int `json:"elements"`
		OutputPath string         `json:"outputPath"`
		Views      int            `json:"views"`
		Warnings   []string       `json:"warnings"`
	}
	if err := json.Unmarshal([]byte(out), &summary); err != nil {
		t.Fatalf("--json output is not valid JSON: %v\noutput:\n%s", err, out)
	}
	if summary.Elements["container"] != 3 {
		t.Errorf("expected 3 containers, got %d (elements=%v)", summary.Elements["container"], summary.Elements)
	}
	if summary.OutputPath != outFile {
		t.Errorf("expected outputPath %q, got %q", outFile, summary.OutputPath)
	}
	if summary.Views != 2 {
		t.Errorf("expected views=2, got %d", summary.Views)
	}
	if summary.Warnings == nil {
		t.Error("expected warnings to be a non-nil (possibly empty) array in JSON output")
	}
}

// TestImportCmd_DryRunSuppressesSummary covers that --dry-run prints only the
// model JSON — no per-kind counts, no next-steps hint, since nothing was
// persisted.
func TestImportCmd_DryRunSuppressesSummary(t *testing.T) {
	dsl := absDSL(t, "internal", "importer", "structurizr", "testdata", "simple.dsl")
	out, err := executeImportCmd("import", "--from", "structurizr", "--dry-run", dsl)
	if err != nil {
		t.Fatalf("expected no error, got %v\nOutput: %s", err, out)
	}
	if strings.Contains(out, "Next steps:") {
		t.Errorf("--dry-run must not print a next-steps hint, got:\n%s", out)
	}
	if strings.Contains(out, "container:") {
		t.Errorf("--dry-run must not print per-kind counts, got:\n%s", out)
	}
}

// TestImportCmd_ForceOverwriteHintsDiff covers that overwriting an existing
// output file with --force hints at `bausteinsicht diff` instead of `sync`.
func TestImportCmd_ForceOverwriteHintsDiff(t *testing.T) {
	dsl := absDSL(t, "internal", "importer", "structurizr", "testdata", "simple.dsl")
	outFile := filepath.Join(t.TempDir(), "architecture.jsonc")
	if err := os.WriteFile(outFile, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := executeImportCmd("import", "--from", "structurizr", "--output", outFile, "--force", dsl)
	if err != nil {
		t.Fatalf("expected no error with --force, got %v", err)
	}
	if !strings.Contains(out, "Next steps: bausteinsicht diff") {
		t.Errorf("expected --force overwrite to hint at \"bausteinsicht diff\", got:\n%s", out)
	}
	if strings.Contains(out, "bausteinsicht sync") {
		t.Errorf("--force overwrite must not also hint at \"bausteinsicht sync\", got:\n%s", out)
	}
}

// TestImportCmd_XMINoViewsLine covers that the views line is omitted in text
// mode when the importer produced zero views (XMI never populates Views).
func TestImportCmd_XMINoViewsLine(t *testing.T) {
	xmi := absDSL(t, "internal", "importer", "xmi", "testdata", "basic.xmi")
	outFile := filepath.Join(t.TempDir(), "architecture.jsonc")
	out, err := executeImportCmd("import", "--from", "xmi", "--output", outFile, xmi)
	if err != nil {
		t.Fatalf("expected no error, got %v\nOutput: %s", err, out)
	}
	if strings.Contains(out, "views:") {
		t.Errorf("expected no \"views:\" line for an XMI import with zero views, got:\n%s", out)
	}
}

// TestImportCmd_LikeC4_NonExistentFile_ExitCode1 is the likec4 counterpart of
// TestImportCmd_NonExistentFile_ExitCode1 — covers the likec4 import-failure
// branch in runImport's format switch.
func TestImportCmd_LikeC4_NonExistentFile_ExitCode1(t *testing.T) {
	_, err := executeImportCmd("import", "--from", "likec4", "--dry-run", "/nonexistent/model.c4")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
	ee, ok := err.(*exitError)
	if !ok {
		t.Fatalf("expected exitError, got %T: %v", err, err)
	}
	if ee.code != 1 {
		t.Errorf("expected exit code 1, got %d", ee.code)
	}
}

// TestImportCmd_XMI_KindMapParseError_ExitCode1 covers the --kind-map parse
// error branch (a malformed entry, missing "Type=kind").
func TestImportCmd_XMI_KindMapParseError_ExitCode1(t *testing.T) {
	xmi := absDSL(t, "internal", "importer", "xmi", "testdata", "basic.xmi")
	_, err := executeImportCmd("import", "--from", "xmi", "--dry-run", "--kind-map", "badentry", xmi)
	if err == nil {
		t.Fatal("expected error for malformed --kind-map entry")
	}
	ee, ok := err.(*exitError)
	if !ok {
		t.Fatalf("expected exitError, got %T: %v", err, err)
	}
	if ee.code != 1 {
		t.Errorf("expected exit code 1, got %d", ee.code)
	}
}

// TestImportCmd_XMI_InvalidFile_ExitCode1 covers the xmi.Import failure
// branch (malformed XML), distinct from a merely-missing file.
func TestImportCmd_XMI_InvalidFile_ExitCode1(t *testing.T) {
	invalid := absDSL(t, "internal", "importer", "xmi", "testdata", "invalid.xmi")
	_, err := executeImportCmd("import", "--from", "xmi", "--dry-run", invalid)
	if err == nil {
		t.Fatal("expected error for invalid XMI content")
	}
	ee, ok := err.(*exitError)
	if !ok {
		t.Fatalf("expected exitError, got %T: %v", err, err)
	}
	if ee.code != 1 {
		t.Errorf("expected exit code 1, got %d", ee.code)
	}
}

// TestImportCmd_InputPathTraversal_ExitCode1 covers validatePathContainment
// rejecting a ".." component in the input path.
func TestImportCmd_InputPathTraversal_ExitCode1(t *testing.T) {
	_, err := executeImportCmd("import", "--from", "structurizr", "--dry-run", "../evil.dsl")
	if err == nil {
		t.Fatal("expected error for path traversal in input path")
	}
	ee, ok := err.(*exitError)
	if !ok {
		t.Fatalf("expected exitError, got %T: %v", err, err)
	}
	if ee.code != 1 {
		t.Errorf("expected exit code 1, got %d", ee.code)
	}
}

// TestImportCmd_OutputPathTraversal_ExitCode1 covers validatePathContainment
// rejecting a ".." component in --output.
func TestImportCmd_OutputPathTraversal_ExitCode1(t *testing.T) {
	dsl := absDSL(t, "internal", "importer", "structurizr", "testdata", "simple.dsl")
	_, err := executeImportCmd("import", "--from", "structurizr", "--output", "../evil.jsonc", dsl)
	if err == nil {
		t.Fatal("expected error for path traversal in --output")
	}
	ee, ok := err.(*exitError)
	if !ok {
		t.Fatalf("expected exitError, got %T: %v", err, err)
	}
	if ee.code != 1 {
		t.Errorf("expected exit code 1, got %d", ee.code)
	}
}

// TestImportCmd_MkdirAllError_ExitCode1 covers os.MkdirAll failing: a path
// component of --output's directory already exists as a regular file, so it
// cannot be created as a directory.
func TestImportCmd_MkdirAllError_ExitCode1(t *testing.T) {
	dsl := absDSL(t, "internal", "importer", "structurizr", "testdata", "simple.dsl")
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	outFile := filepath.Join(blocker, "sub", "architecture.jsonc")
	_, err := executeImportCmd("import", "--from", "structurizr", "--output", outFile, dsl)
	if err == nil {
		t.Fatal("expected error creating output directory under a file")
	}
	ee, ok := err.(*exitError)
	if !ok {
		t.Fatalf("expected exitError, got %T: %v", err, err)
	}
	if ee.code != 1 {
		t.Errorf("expected exit code 1, got %d", ee.code)
	}
}

// TestImportCmd_WriteFileError_ExitCode1 covers os.WriteFile failing: --force
// bypasses the pre-existence check, but the output path is itself an
// existing directory, so the write fails.
func TestImportCmd_WriteFileError_ExitCode1(t *testing.T) {
	dsl := absDSL(t, "internal", "importer", "structurizr", "testdata", "simple.dsl")
	dir := t.TempDir()
	outAsDir := filepath.Join(dir, "isadir")
	if err := os.Mkdir(outAsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := executeImportCmd("import", "--from", "structurizr", "--output", outAsDir, "--force", dsl)
	if err == nil {
		t.Fatal("expected error writing to a directory path")
	}
	ee, ok := err.(*exitError)
	if !ok {
		t.Fatalf("expected exitError, got %T: %v", err, err)
	}
	if ee.code != 1 {
		t.Errorf("expected exit code 1, got %d", ee.code)
	}
}

// warningDSL imports to exactly 3 element kinds (person, system, container),
// 1 view, and 1 warning (autoLayout direction "lr" not preserved) — used to
// pin down the exact number of stdout writes the summary/hint paths make, so
// TestImportCmd_TextSummary_WriteFailureAtEveryOffset and
// TestImportCmd_DryRun_WriteFailureAtEveryOffset can inject a failure at
// every single write and verify each one is handled.
const warningDSL = `workspace "W" "one warning fixture" {
    model {
        user = person "User" "A user."
        sys = softwareSystem "Sys" "A system." {
            api = container "API" "An API." "Go"
        }
    }
    views {
        systemContext sys "Context" {
            include *
            autoLayout lr
        }
    }
}
`

// failAfterWriter succeeds for the first n writes, then fails every write
// after that. Used to exercise the "if _, err := fmt.Fprint...; err != nil"
// branches, which are otherwise unreachable with an in-memory buffer that
// never errors — a real-world equivalent is a broken pipe (e.g.
// `bausteinsicht import ... | head -1`, where the reader closes early).
type failAfterWriter struct{ n int }

func (w *failAfterWriter) Write(p []byte) (int, error) {
	if w.n <= 0 {
		return 0, fmt.Errorf("simulated write failure")
	}
	w.n--
	return len(p), nil
}

func executeImportCmdWithWriter(w io.Writer, args ...string) error {
	cmd := NewRootCmd()
	cmd.SetOut(w)
	cmd.SetErr(w)
	cmd.SetArgs(args)
	return cmd.Execute()
}

// TestImportCmd_TextSummary_WriteFailureAtEveryOffset covers every
// fmt.Fprint*/Fprintln error-return branch in printImportSummaryText: the
// "Imported model written to" line, the warning line, each per-kind count
// line, the views line, and the next-steps line — 7 writes total for
// warningDSL (1 + 1 warning + 3 kinds + 1 views + 1 next-steps).
func TestImportCmd_TextSummary_WriteFailureAtEveryOffset(t *testing.T) {
	dir := t.TempDir()
	dslPath := filepath.Join(dir, "workspace.dsl")
	if err := os.WriteFile(dslPath, []byte(warningDSL), 0o600); err != nil {
		t.Fatal(err)
	}
	const totalWrites = 7

	for n := 0; n <= totalWrites; n++ {
		t.Run(fmt.Sprintf("failAt%d", n), func(t *testing.T) {
			outFile := filepath.Join(t.TempDir(), "architecture.jsonc")
			w := &failAfterWriter{n: n}
			err := executeImportCmdWithWriter(w, "import", "--from", "structurizr", "--output", outFile, dslPath)
			if n < totalWrites {
				if err == nil {
					t.Fatalf("expected write failure at offset %d, got nil error", n)
				}
			} else if err != nil {
				t.Fatalf("expected success once the writer stops failing, got: %v", err)
			}
		})
	}
}

// TestImportCmd_DryRun_WriteFailureAtEveryOffset is the --dry-run
// counterpart: covers the model-JSON print and the warnings loop in
// runImport's dry-run branch — 2 writes total for warningDSL (1 model print
// + 1 warning).
func TestImportCmd_DryRun_WriteFailureAtEveryOffset(t *testing.T) {
	dir := t.TempDir()
	dslPath := filepath.Join(dir, "workspace.dsl")
	if err := os.WriteFile(dslPath, []byte(warningDSL), 0o600); err != nil {
		t.Fatal(err)
	}
	const totalWrites = 2

	for n := 0; n <= totalWrites; n++ {
		t.Run(fmt.Sprintf("failAt%d", n), func(t *testing.T) {
			w := &failAfterWriter{n: n}
			err := executeImportCmdWithWriter(w, "import", "--from", "structurizr", "--dry-run", dslPath)
			if n < totalWrites {
				if err == nil {
					t.Fatalf("expected write failure at offset %d, got nil error", n)
				}
			} else if err != nil {
				t.Fatalf("expected success once the writer stops failing, got: %v", err)
			}
		})
	}
}

// TestImportCmd_JSONSummary_WriteFailure covers printImportSummaryJSON's
// single write (the whole summary is marshaled first, then written once).
func TestImportCmd_JSONSummary_WriteFailure(t *testing.T) {
	dsl := absDSL(t, "internal", "importer", "structurizr", "testdata", "simple.dsl")
	outFile := filepath.Join(t.TempDir(), "architecture.jsonc")
	w := &failAfterWriter{n: 0}
	err := executeImportCmdWithWriter(w, "import", "--from", "structurizr", "--output", outFile, "--json", dsl)
	if err == nil {
		t.Fatal("expected error when the JSON summary write fails")
	}
}

// deeplyNestedModel builds a model whose element hierarchy exceeds
// model.MaxElementDepth — no real importer output can reach this in
// practice (structurizr/likec4 only ever produce a fixed 3-tier hierarchy,
// and XMI enforces its own, separate 50-level cap before ever returning a
// model), so it's constructed by hand to exercise
// elementCountsByKind's/model.FlattenElements' depth-limit error return.
func deeplyNestedModel() *model.BausteinsichtModel {
	var current model.Element
	for i := 0; i <= 60; i++ {
		current = model.Element{Kind: "component", Title: "Level", Children: map[string]model.Element{"child": current}}
	}
	return &model.BausteinsichtModel{Model: map[string]model.Element{"root": current}}
}

// TestElementCountsByKind_DepthExceeded_Error covers elementCountsByKind's
// error-wrapping branch when model.FlattenElements rejects the hierarchy.
func TestElementCountsByKind_DepthExceeded_Error(t *testing.T) {
	_, err := elementCountsByKind(deeplyNestedModel())
	if err == nil {
		t.Fatal("expected an error for a model exceeding MaxElementDepth")
	}
	if !strings.Contains(err.Error(), "flattening imported model") {
		t.Errorf("expected wrapped error to mention \"flattening imported model\", got: %v", err)
	}
}

// TestPrintImportSummaryText_DepthExceeded_Error covers the
// elementCountsByKind error-return branch inside printImportSummaryText.
func TestPrintImportSummaryText_DepthExceeded_Error(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	err := printImportSummaryText(cmd, deeplyNestedModel(), "architecture.jsonc", nil, false)
	if err == nil {
		t.Fatal("expected an error for a model exceeding MaxElementDepth")
	}
}

// TestPrintImportSummaryJSON_DepthExceeded_Error covers the
// elementCountsByKind error-return branch inside printImportSummaryJSON.
func TestPrintImportSummaryJSON_DepthExceeded_Error(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	err := printImportSummaryJSON(cmd, deeplyNestedModel(), "architecture.jsonc", nil)
	if err == nil {
		t.Fatal("expected an error for a model exceeding MaxElementDepth")
	}
}
