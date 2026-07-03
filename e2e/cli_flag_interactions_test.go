package e2e

// TestCLIFlagInteractions (#519) closes E2E-Test-Plan.adoc section 11 (9
// tests, previously zero e2e coverage): --verbose across commands,
// --model/--template extension and existence validation, --format json
// combined with --verbose, and unknown-flag/--help handling.

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCLIFlagInteractions(t *testing.T) {
	t.Run("11.1_VerboseWithValidate", testFlagVerboseValidate)
	t.Run("11.2_VerboseWithSync", testFlagVerboseSync)
	t.Run("11.3_VerboseWithExport", testFlagVerboseExport)
	t.Run("11.4_ModelPointingToDrawio", testFlagModelWrongType)
	t.Run("11.5_TemplatePointingToJsonc", testFlagTemplateWrongType)
	t.Run("11.6_FormatJSONWithVerbose", testFlagFormatJSONWithVerbose)
	t.Run("11.7_BothModelAndTemplateNonexistent", testFlagBothNonexistent)
	t.Run("11.8_HelpOnSubcommands", testFlagHelpOnSubcommands)
	t.Run("11.9_UnknownFlag", testFlagUnknown)
}

// testFlagVerboseValidate covers 11.1: --verbose adds element/relationship/
// view counts to validate's stderr output.
func testFlagVerboseValidate(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	plain := runCLI(t, bin, dir, "validate")
	verbose := runCLI(t, bin, dir, "validate", "--verbose")

	if len(verbose) <= len(plain) {
		t.Errorf("--verbose output (%d bytes) should be longer than plain output (%d bytes)", len(verbose), len(plain))
	}
	if !strings.Contains(verbose, "elements") || !strings.Contains(verbose, "relationships") {
		t.Errorf("expected element/relationship counts in verbose output, got: %s", verbose)
	}
}

// testFlagVerboseSync covers 11.2: --verbose adds pre-sync counts and
// post-sync stats to sync's stderr output.
func testFlagVerboseSync(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	plain := runCLI(t, bin, dir, "sync")
	verbose := runCLI(t, bin, dir, "sync", "--verbose")

	if len(verbose) <= len(plain) {
		t.Errorf("--verbose output (%d bytes) should be longer than plain output (%d bytes)", len(verbose), len(plain))
	}
	if !strings.Contains(verbose, "Syncing model") {
		t.Errorf("expected 'Syncing model' in verbose sync output, got: %s", verbose)
	}
}

// testFlagVerboseExport covers 11.3: --verbose prints which draw.io CLI
// binary was resolved, before the actual export subprocess runs — so this
// is observable even if headless draw.io export itself isn't exercised.
// Still requires draw.io to be *found* on PATH (not run) — CI runners don't
// have it installed, so this skips gracefully there, same as this suite's
// other draw.io-dependent tests (see findDrawioCmd in bigbank_arc42_test.go).
func testFlagVerboseExport(t *testing.T) {
	if findDrawioCmd() == "" {
		t.Skip("draw.io CLI not found on PATH — skipping (see e.g. bigbank_arc42_test.go for the same pattern)")
	}
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	out, _ := runCLIAllowFail(t, bin, dir, "export", "--verbose")
	if !strings.Contains(out, "Using draw.io CLI") {
		t.Errorf("expected 'Using draw.io CLI' verbose line, got: %s", out)
	}
}

// testFlagModelWrongType covers 11.4: --model pointing at a .drawio file
// fails with a clear JSON-parse error (no dedicated extension check for
// --model, unlike --template — it's caught downstream by model.Load trying
// to parse XML as JSONC).
func testFlagModelWrongType(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	out, code := runCLIAllowFail(t, bin, dir, "validate", "--model", "architecture.drawio")
	if code != 2 {
		t.Fatalf("expected exit 2, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "parsing") {
		t.Errorf("expected a parse error, got: %s", out)
	}
}

// testFlagTemplateWrongType covers 11.5: --template pointing at a .jsonc
// file is rejected outright by an extension check (not a silent fallback
// to the default template).
func testFlagTemplateWrongType(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	out, code := runCLIAllowFail(t, bin, dir, "sync", "--template", "architecture.jsonc")
	if code != 1 {
		t.Fatalf("expected exit 1, got %d\n%s", code, out)
	}
	if !strings.Contains(out, ".drawio extension") {
		t.Errorf("expected a .drawio extension error, got: %s", out)
	}
}

// testFlagFormatJSONWithVerbose covers 11.6: --format json together with
// --verbose must still produce clean, parseable JSON on stdout — verbose
// detail is routed to stderr only.
func testFlagFormatJSONWithVerbose(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	stdout, stderr, code := runCLISplit(t, bin, dir, "sync", "--format", "json", "--verbose")
	if code != 0 {
		t.Fatalf("sync --format json --verbose failed: exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &parsed); err != nil {
		t.Errorf("stdout is not valid JSON with --verbose: %v\nstdout: %s", err, stdout)
	}
	if strings.Contains(stdout, "Syncing model") {
		t.Error("verbose text leaked into stdout JSON output")
	}
}

// testFlagBothNonexistent covers 11.7: when both --model and --template
// point at nonexistent files, the error is clear (the model load failure
// surfaces first, since sync loads the model before the template).
func testFlagBothNonexistent(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	out, code := runCLIAllowFail(t, bin, dir, "sync",
		"--model", "nonexistent-model.jsonc",
		"--template", "nonexistent-template.drawio",
	)
	if code != 2 {
		t.Fatalf("expected exit 2, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "loading model") {
		t.Errorf("expected a clear 'loading model' error, got: %s", out)
	}
}

// testFlagHelpOnSubcommands covers 11.8: --help on every listed subcommand
// prints help text and exits 0.
func testFlagHelpOnSubcommands(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	cases := [][]string{
		{"init", "--help"},
		{"sync", "--help"},
		{"validate", "--help"},
		{"watch", "--help"},
		{"add", "element", "--help"},
		{"add", "relationship", "--help"},
		{"export", "--help"},
	}
	for _, args := range cases {
		out, code := runCLIAllowFail(t, bin, dir, args...)
		if code != 0 {
			t.Errorf("%v: expected exit 0, got %d\n%s", args, code, out)
		}
		if !strings.Contains(out, "Usage:") {
			t.Errorf("%v: expected 'Usage:' in help text, got: %s", args, out)
		}
	}
}

// testFlagUnknown covers 11.9: an unrecognized flag is rejected by
// cobra/pflag before RunE executes.
func testFlagUnknown(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	out, code := runCLIAllowFail(t, bin, dir, "sync", "--bogus")
	if code != 1 {
		t.Fatalf("expected exit 1, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "unknown flag") {
		t.Errorf("expected 'unknown flag' error, got: %s", out)
	}
}
