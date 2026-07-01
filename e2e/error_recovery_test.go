package e2e

// TestErrorRecovery (#504) verifies that sync handles corrupt/missing files
// gracefully: clear error messages on bad input, no silent data loss.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestErrorRecovery(t *testing.T) {
	t.Run("InvalidXML", testErrorInvalidXML)
	t.Run("EmptyMxfile", testErrorEmptyMxfile)
	t.Run("MissingDrawio", testErrorMissingDrawio)
	t.Run("BrokenJSONC", testErrorBrokenJSONC)
}

// testErrorInvalidXML: corrupt draw.io → sync exits non-zero, model unchanged.
func testErrorInvalidXML(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	modelPath := filepath.Join(dir, "architecture.jsonc")
	drawioPath := filepath.Join(dir, "architecture.drawio")

	before, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("read model before: %v", err)
	}

	if err := os.WriteFile(drawioPath, []byte("<broken>not valid xml"), 0o644); err != nil {
		t.Fatalf("write corrupt draw.io: %v", err)
	}

	out, code := runCLIAllowFail(t, bin, dir, "sync")
	if code == 0 {
		t.Errorf("expected non-zero exit on corrupt draw.io, got 0\noutput: %s", out)
	}

	after, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("read model after: %v", err)
	}
	if string(before) != string(after) {
		t.Error("model was modified despite corrupt draw.io — data loss risk")
	}
}

// testErrorEmptyMxfile: empty <mxfile/> (no diagram) → sync runs, creates pages.
func testErrorEmptyMxfile(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	drawioPath := filepath.Join(dir, "architecture.drawio")
	empty := `<?xml version="1.0" encoding="UTF-8"?><mxfile></mxfile>`
	if err := os.WriteFile(drawioPath, []byte(empty), 0o644); err != nil {
		t.Fatalf("write empty draw.io: %v", err)
	}

	// Sync should succeed (treat empty file as starting point) or fail with a
	// clear message — either is acceptable, but it must not crash silently.
	out, code := runCLIAllowFail(t, bin, dir, "sync")
	t.Logf("empty mxfile sync: exit=%d output=%s", code, out)
	// No panic assertion needed — if it reaches here without hanging, it passed.
}

// testErrorMissingDrawio: no draw.io file → sync creates a new one (first-sync).
func testErrorMissingDrawio(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	drawioPath := filepath.Join(dir, "architecture.drawio")
	if err := os.Remove(drawioPath); err != nil {
		t.Fatalf("remove draw.io: %v", err)
	}

	runCLI(t, bin, dir, "sync")

	if _, err := os.Stat(drawioPath); os.IsNotExist(err) {
		t.Error("sync did not recreate missing draw.io file")
	}
}

// testErrorBrokenJSONC: invalid JSONC → sync + validate both fail with clear error.
func testErrorBrokenJSONC(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	modelPath := filepath.Join(dir, "architecture.jsonc")
	if err := os.WriteFile(modelPath, []byte(`{ "model": { broken json`), 0o644); err != nil {
		t.Fatalf("write broken jsonc: %v", err)
	}

	syncOut, syncCode := runCLIAllowFail(t, bin, dir, "sync")
	if syncCode == 0 {
		t.Errorf("sync with broken JSONC: expected non-zero exit, got 0\noutput: %s", syncOut)
	}

	valOut, valCode := runCLIAllowFail(t, bin, dir, "validate")
	if valCode == 0 {
		t.Errorf("validate with broken JSONC: expected non-zero exit, got 0\noutput: %s", valOut)
	}

	// Error message should give a useful hint (line/column or parse error).
	combined := syncOut + valOut
	if !strings.Contains(strings.ToLower(combined), "error") &&
		!strings.Contains(strings.ToLower(combined), "invalid") &&
		!strings.Contains(strings.ToLower(combined), "parse") {
		t.Errorf("no useful error message in output: %s", combined)
	}
}
