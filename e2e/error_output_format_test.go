package e2e

// TestErrorOutputFormatting (#519) closes E2E-Test-Plan.adoc section 15 (8
// tests, previously zero e2e coverage): the {"error":"...","code":N} JSON
// error contract (cmd/bausteinsicht/root.go ExecuteRoot, root.go:112-134)
// is shared by every command, so these tests confirm it's actually applied
// uniformly — including that verbose/success output never leaks onto the
// wrong stream when --format json is active.

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestErrorOutputFormatting(t *testing.T) {
	t.Run("15.1_ErrorAsJSON_Validate", testErrJSONValidate)
	t.Run("15.2_ErrorAsJSON_Sync", testErrJSONSync)
	t.Run("15.3_ErrorAsJSON_AddElement", testErrJSONAddElement)
	t.Run("15.4_ErrorAsJSON_Export", testErrJSONExport)
	t.Run("15.5_ErrorAsText_Default", testErrTextDefault)
	t.Run("15.6_ErrorExitCodes", testErrExitCodes)
	t.Run("15.7_SuccessJSONConsistency", testSuccessJSONConsistency)
	t.Run("15.8_NoMixedStdoutStderr", testErrNoMixedStreams)
}

// jsonErrorEnvelope is the shared {"error":"...","code":N} shape.
type jsonErrorEnvelope struct {
	Error string `json:"error"`
	Code  int    `json:"code"`
}

// assertJSONErrorOnStderr parses stderr as the shared error envelope and
// confirms stdout is empty (JSON errors go to stderr only).
func assertJSONErrorOnStderr(t *testing.T, stdout, stderr string, wantCode int) {
	t.Helper()
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("expected empty stdout for a JSON error, got: %q", stdout)
	}
	var env jsonErrorEnvelope
	if err := json.Unmarshal([]byte(strings.TrimSpace(stderr)), &env); err != nil {
		t.Fatalf("stderr is not a valid {\"error\":...,\"code\":...} JSON envelope: %v\nstderr: %s", err, stderr)
	}
	if env.Error == "" {
		t.Error("error field is empty")
	}
	if env.Code != wantCode {
		t.Errorf("code = %d, want %d", env.Code, wantCode)
	}
}

// testErrJSONValidate covers 15.1.
func testErrJSONValidate(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	stdout, stderr, code := runCLISplit(t, bin, dir, "validate", "--model", "/nonexistent", "--format", "json")
	if code == 0 {
		t.Fatal("expected non-zero exit")
	}
	assertJSONErrorOnStderr(t, stdout, stderr, code)
}

// testErrJSONSync covers 15.2.
func testErrJSONSync(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	stdout, stderr, code := runCLISplit(t, bin, dir, "sync", "--model", "/nonexistent", "--format", "json")
	if code == 0 {
		t.Fatal("expected non-zero exit")
	}
	assertJSONErrorOnStderr(t, stdout, stderr, code)
}

// testErrJSONAddElement covers 15.3.
func testErrJSONAddElement(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	stdout, stderr, code := runCLISplit(t, bin, dir, "add", "element",
		"--id", "test1", "--title", "Test", "--kind", "invalid", "--format", "json")
	if code == 0 {
		t.Fatal("expected non-zero exit")
	}
	assertJSONErrorOnStderr(t, stdout, stderr, code)
}

// testErrJSONExport covers 15.4.
func testErrJSONExport(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	stdout, stderr, code := runCLISplit(t, bin, dir, "export", "--format", "json", "--drawio-path", "/nonexistent")
	if code == 0 {
		t.Fatal("expected non-zero exit")
	}
	assertJSONErrorOnStderr(t, stdout, stderr, code)
}

// testErrTextDefault covers 15.5: without --format json, errors are plain
// text on stderr (not a JSON envelope).
func testErrTextDefault(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	stdout, stderr, code := runCLISplit(t, bin, dir, "validate", "--model", "/nonexistent")
	if code == 0 {
		t.Fatal("expected non-zero exit")
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("expected empty stdout, got: %q", stdout)
	}
	trimmed := strings.TrimSpace(stderr)
	if trimmed == "" {
		t.Fatal("expected an error message on stderr")
	}
	var env jsonErrorEnvelope
	if err := json.Unmarshal([]byte(trimmed), &env); err == nil {
		t.Errorf("expected plain text error, but stderr parsed as JSON: %s", stderr)
	}
}

// testErrExitCodes covers 15.6: user errors (bad input/state) exit 1,
// system/IO errors (file not found, can't load) exit 2.
func testErrExitCodes(t *testing.T) {
	bin := buildBinary(t)

	cases := []struct {
		name     string
		args     []string
		wantCode int
		needInit bool
	}{
		{"validate nonexistent model (system/IO)", []string{"validate", "--model", "/nonexistent"}, 2, false},
		{"sync nonexistent model (system/IO)", []string{"sync", "--model", "/nonexistent"}, 2, false},
		{"add element unknown kind (user error)", []string{"add", "element", "--id", "x", "--title", "T", "--kind", "bogus"}, 1, true},
		{"add element invalid ID (user error)", []string{"add", "element", "--id", "$(x)", "--title", "T", "--kind", "system"}, 1, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if tc.needInit {
				runCLI(t, bin, dir, "init")
			}
			out, code := runCLIAllowFail(t, bin, dir, tc.args...)
			if code != tc.wantCode {
				t.Errorf("exit = %d, want %d\n%s", code, tc.wantCode, out)
			}
		})
	}
}

// testSuccessJSONConsistency covers 15.7: successful --format json output
// is always valid, parseable JSON across commands.
func testSuccessJSONConsistency(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	stdout, _, code := runCLISplit(t, bin, dir, "init", "--format", "json")
	if code != 0 {
		t.Fatalf("init --format json failed: exit %d\n%s", code, stdout)
	}
	requireValidJSON(t, "init", stdout)

	stdout, _, code = runCLISplit(t, bin, dir, "validate", "--format", "json")
	if code != 0 {
		t.Fatalf("validate --format json failed: exit %d\n%s", code, stdout)
	}
	requireValidJSON(t, "validate", stdout)

	stdout, _, code = runCLISplit(t, bin, dir, "sync", "--format", "json")
	if code != 0 {
		t.Fatalf("sync --format json failed: exit %d\n%s", code, stdout)
	}
	requireValidJSON(t, "sync", stdout)

	stdout, _, code = runCLISplit(t, bin, dir, "add", "element", "--id", "jsontest", "--title", "T", "--kind", "system", "--format", "json")
	if code != 0 {
		t.Fatalf("add element --format json failed: exit %d\n%s", code, stdout)
	}
	requireValidJSON(t, "add element", stdout)
}

func requireValidJSON(t *testing.T, label, stdout string) {
	t.Helper()
	var v interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &v); err != nil {
		t.Errorf("%s: stdout is not valid JSON: %v\nstdout: %s", label, err, stdout)
	}
}

// testErrNoMixedStreams covers 15.8: a --format json error must not print
// anything to stdout — the JSON envelope is stderr-only.
func testErrNoMixedStreams(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	stdout, stderr, code := runCLISplit(t, bin, dir, "add", "element",
		"--id", "test2", "--title", "T", "--kind", "totally-bogus-kind", "--format", "json")
	if code == 0 {
		t.Fatal("expected non-zero exit")
	}
	if stdout != "" {
		t.Errorf("expected zero bytes on stdout, got %d bytes: %q", len(stdout), stdout)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Fatal("expected JSON error on stderr")
	}
}
