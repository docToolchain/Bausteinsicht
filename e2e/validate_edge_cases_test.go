package e2e

// TestValidateEdgeCases (#519) closes E2E-Test-Plan.adoc section 2 (25
// tests) — the lowest-priority item from #519's list ("likely already
// well unit-tested"), added for full closure. Confirms `internal/model`
// validation rules are actually reachable and correctly reported through
// the `validate` CLI command, including its two-JSON-stream quirk: on
// validation errors with --format json, the {"valid":false,...} report goes
// to stdout while a *second*, generic {"error":"validation failed","code":1}
// envelope (from the shared ExecuteRoot error path) goes to stderr.

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestValidateEdgeCases(t *testing.T) {
	t.Run("2.1_EmptyJSON", test2_1EmptyJSON)
	t.Run("2.2_SpecWithoutModel", test2_2SpecWithoutModel)
	t.Run("2.3_ModelWithoutSpec", test2_3ModelWithoutSpec)
	t.Run("2.4_UnknownElementKind", test2_4UnknownElementKind)
	t.Run("2.5_RelFromDoesNotExist", test2_5RelFromMissing)
	t.Run("2.6_RelToDoesNotExist", test2_6RelToMissing)
	t.Run("2.7_EmptyRelationships", test2_7EmptyRelationships)
	t.Run("2.8_EmptyViews", test2_8EmptyViews)
	t.Run("2.9_FormatJSONForErrors", test2_9FormatJSONErrors)
	t.Run("2.10_FormatJSONForValid", test2_10FormatJSONValid)
	t.Run("2.11_NonexistentModelFile", test2_11NonexistentModelFile)
	t.Run("2.12_ModelPathIsDirectory", test2_12ModelPathIsDirectory)
	t.Run("2.13_ViewScopeNonexistent", test2_13ViewScopeNonexistent)
	t.Run("2.14_ViewIncludeNonexistent", test2_14ViewIncludeNonexistent)
	t.Run("2.15_ViewExcludeNonexistent", test2_15ViewExcludeNonexistent)
	t.Run("2.16_DuplicateIDsDifferentLevels", test2_16DuplicateIDsDifferentLevels)
	t.Run("2.17_UnknownRelationshipKind", test2_17UnknownRelationshipKind)
	t.Run("2.18_EmptyStringElementID", test2_18EmptyStringElementID)
	t.Run("2.19_WhitespaceOnlyElementID", test2_19WhitespaceOnlyElementID)
	t.Run("2.20_NullRootJSON", test2_20NullRootJSON)
	t.Run("2.21_InvalidFormatFlag", test2_21InvalidFormatFlag)
	t.Run("2.22_UppercaseFormatFlag", test2_22UppercaseFormatFlag)
	t.Run("2.23_MalformedJSON", test2_23MalformedJSON)
	t.Run("2.24_FiveHundredElementsStressTest", test2_24FiveHundredElements)
	t.Run("2.25_DeepNesting", test2_25DeepNesting)
}

// test2_1EmptyJSON covers 2.1: "{}" has no specification/model — a warning
// (empty model), not a validation error.
func test2_1EmptyJSON(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeFile(t, dir+"/architecture.jsonc", `{}`)
	out, code := runCLIAllowFail(t, bin, dir, "validate")
	if code != 0 {
		t.Errorf("expected exit 0 (warning, not error) for empty model, got %d\n%s", code, out)
	}
}

// test2_2SpecWithoutModel covers 2.2: only "specification", no "model" —
// valid (empty model is a warning, not an error).
func test2_2SpecWithoutModel(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeFile(t, dir+"/architecture.jsonc", `{
  "specification": { "elements": { "system": { "notation": "System" } } }
}
`)
	out, code := runCLIAllowFail(t, bin, dir, "validate")
	if code != 0 {
		t.Errorf("expected exit 0, got %d\n%s", code, out)
	}
}

// test2_3ModelWithoutSpec covers 2.3: an element with no matching spec
// entry is an "unknown kind" error.
func test2_3ModelWithoutSpec(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeFile(t, dir+"/architecture.jsonc", `{
  "model": { "orphan": { "kind": "system", "title": "Orphan" } }
}
`)
	out, code := runCLIAllowFail(t, bin, dir, "validate")
	if code != 1 {
		t.Fatalf("expected exit 1, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "unknown kind") {
		t.Errorf("expected 'unknown kind' error, got: %s", out)
	}
}

// test2_4UnknownElementKind covers 2.4.
func test2_4UnknownElementKind(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeFile(t, dir+"/architecture.jsonc", `{
  "specification": { "elements": { "system": { "notation": "System" } } },
  "model": { "orphan": { "kind": "microservice", "title": "Orphan" } }
}
`)
	out, code := runCLIAllowFail(t, bin, dir, "validate")
	if code != 1 {
		t.Fatalf("expected exit 1, got %d\n%s", code, out)
	}
	if !strings.Contains(out, `unknown kind "microservice"`) {
		t.Errorf("expected unknown-kind error naming the kind, got: %s", out)
	}
}

// test2_5RelFromMissing covers 2.5.
func test2_5RelFromMissing(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeFile(t, dir+"/architecture.jsonc", `{
  "specification": { "elements": { "system": { "notation": "System" } } },
  "model": { "b": { "kind": "system", "title": "B" } },
  "relationships": [ { "from": "nonexistent", "to": "b" } ]
}
`)
	out, code := runCLIAllowFail(t, bin, dir, "validate")
	if code != 1 {
		t.Fatalf("expected exit 1, got %d\n%s", code, out)
	}
	if !strings.Contains(out, `"nonexistent" does not resolve`) {
		t.Errorf("expected a 'from' resolution error, got: %s", out)
	}
}

// test2_6RelToMissing covers 2.6.
func test2_6RelToMissing(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeFile(t, dir+"/architecture.jsonc", `{
  "specification": { "elements": { "system": { "notation": "System" } } },
  "model": { "a": { "kind": "system", "title": "A" } },
  "relationships": [ { "from": "a", "to": "ghost" } ]
}
`)
	out, code := runCLIAllowFail(t, bin, dir, "validate")
	if code != 1 {
		t.Fatalf("expected exit 1, got %d\n%s", code, out)
	}
	if !strings.Contains(out, `"ghost" does not resolve`) {
		t.Errorf("expected a 'to' resolution error, got: %s", out)
	}
}

// test2_7EmptyRelationships covers 2.7.
func test2_7EmptyRelationships(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeFile(t, dir+"/architecture.jsonc", `{
  "specification": { "elements": { "system": { "notation": "System" } } },
  "model": { "a": { "kind": "system", "title": "A" } },
  "relationships": []
}
`)
	out, code := runCLIAllowFail(t, bin, dir, "validate")
	if code != 0 {
		t.Errorf("expected exit 0, got %d\n%s", code, out)
	}
}

// test2_8EmptyViews covers 2.8.
func test2_8EmptyViews(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeFile(t, dir+"/architecture.jsonc", `{
  "specification": { "elements": { "system": { "notation": "System" } } },
  "model": { "a": { "kind": "system", "title": "A" } },
  "views": {}
}
`)
	out, code := runCLIAllowFail(t, bin, dir, "validate")
	if code != 0 {
		t.Errorf("expected exit 0, got %d\n%s", code, out)
	}
}

type validateJSONReport struct {
	Valid  bool `json:"valid"`
	Errors []struct {
		Path    string `json:"path"`
		Message string `json:"message"`
	} `json:"errors"`
}

// test2_9FormatJSONErrors covers 2.9: on validation errors, the
// {"valid":false,"errors":[...]} report is on stdout, and — because
// outputJSON's exitWithCode also flows through the shared ExecuteRoot error
// path — a *separate* generic {"error":"validation failed","code":1}
// envelope appears on stderr.
func test2_9FormatJSONErrors(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeFile(t, dir+"/architecture.jsonc", `{
  "specification": { "elements": { "system": { "notation": "System" } } },
  "model": { "orphan": { "kind": "bogus", "title": "Orphan" } }
}
`)
	stdout, stderr, code := runCLISplit(t, bin, dir, "validate", "--format", "json")
	if code != 1 {
		t.Fatalf("expected exit 1, got %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	var report validateJSONReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\nstdout: %s", err, stdout)
	}
	if report.Valid {
		t.Error("expected valid=false")
	}
	if len(report.Errors) != 1 || report.Errors[0].Path != "model.orphan" {
		t.Errorf("expected one error at model.orphan, got: %+v", report.Errors)
	}
	var errEnvelope jsonErrorEnvelope
	if err := json.Unmarshal([]byte(strings.TrimSpace(stderr)), &errEnvelope); err != nil {
		t.Fatalf("stderr is not a valid error envelope: %v\nstderr: %s", err, stderr)
	}
	if errEnvelope.Code != 1 {
		t.Errorf("stderr envelope code = %d, want 1", errEnvelope.Code)
	}
}

// test2_10FormatJSONValid covers 2.10.
func test2_10FormatJSONValid(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	stdout, stderr, code := runCLISplit(t, bin, dir, "validate", "--format", "json")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	var report validateJSONReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\nstdout: %s", err, stdout)
	}
	if !report.Valid {
		t.Error("expected valid=true for the sample model")
	}
	if len(report.Errors) != 0 {
		t.Errorf("expected zero errors, got: %+v", report.Errors)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Errorf("expected empty stderr on success, got: %s", stderr)
	}
}

// test2_11NonexistentModelFile covers 2.11.
func test2_11NonexistentModelFile(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	out, code := runCLIAllowFail(t, bin, dir, "validate", "--model", "does-not-exist.jsonc")
	if code != 2 {
		t.Fatalf("expected exit 2, got %d\n%s", code, out)
	}
}

// test2_12ModelPathIsDirectory covers 2.12. The OS-level error text differs
// by platform (Linux/macOS: "is a directory"; Windows: "Incorrect
// function.") since it's the underlying syscall error surfacing through
// os.ReadFile, not something the product controls — so this only asserts
// the exit code (a clear, non-crashing system/IO error), not exact wording.
func test2_12ModelPathIsDirectory(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	out, code := runCLIAllowFail(t, bin, dir, "validate", "--model", ".")
	if code != 2 {
		t.Fatalf("expected exit 2, got %d\n%s", code, out)
	}
	if strings.TrimSpace(out) == "" {
		t.Error("expected a non-empty error message")
	}
}

// test2_13ViewScopeNonexistent covers 2.13 (see also section 5's dedicated
// coverage of this same rule via `sync` — this confirms `validate` reports
// it identically).
func test2_13ViewScopeNonexistent(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeFile(t, dir+"/architecture.jsonc", `{
  "specification": { "elements": { "actor": { "notation": "Actor" } } },
  "model": { "customer": { "kind": "actor", "title": "Customer" } },
  "views": { "broken": { "title": "Broken", "scope": "nonexistent", "include": ["customer"] } }
}
`)
	out, code := runCLIAllowFail(t, bin, dir, "validate")
	if code != 1 {
		t.Fatalf("expected exit 1, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "does not resolve to an existing element") {
		t.Errorf("expected a scope resolution error, got: %s", out)
	}
}

// test2_14ViewIncludeNonexistent covers 2.14: a non-wildcard include entry
// referencing a nonexistent element is a validation error (not silently
// ignored).
func test2_14ViewIncludeNonexistent(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeFile(t, dir+"/architecture.jsonc", `{
  "specification": { "elements": { "actor": { "notation": "Actor" } } },
  "model": { "customer": { "kind": "actor", "title": "Customer" } },
  "views": { "broken": { "title": "Broken", "include": ["bogus"] } }
}
`)
	out, code := runCLIAllowFail(t, bin, dir, "validate")
	if code != 1 {
		t.Fatalf("expected exit 1, got %d\n%s", code, out)
	}
	if !strings.Contains(out, `"bogus" does not exist`) {
		t.Errorf("expected an include resolution error, got: %s", out)
	}
}

// test2_15ViewExcludeNonexistent covers 2.15.
func test2_15ViewExcludeNonexistent(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeFile(t, dir+"/architecture.jsonc", `{
  "specification": { "elements": { "actor": { "notation": "Actor" } } },
  "model": { "customer": { "kind": "actor", "title": "Customer" } },
  "views": { "broken": { "title": "Broken", "include": ["customer"], "exclude": ["bogus"] } }
}
`)
	out, code := runCLIAllowFail(t, bin, dir, "validate")
	if code != 1 {
		t.Fatalf("expected exit 1, got %d\n%s", code, out)
	}
	if !strings.Contains(out, `"bogus" does not exist`) {
		t.Errorf("expected an exclude resolution error, got: %s", out)
	}
}

// test2_16DuplicateIDsDifferentLevels covers 2.16: a top-level "api" and a
// nested "parent.api" are different dot-paths — valid, no collision.
func test2_16DuplicateIDsDifferentLevels(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeFile(t, dir+"/architecture.jsonc", `{
  "specification": { "elements": { "system": { "notation": "System", "container": true } } },
  "model": {
    "api": { "kind": "system", "title": "Top-level API" },
    "parent": {
      "kind": "system", "title": "Parent",
      "children": { "api": { "kind": "system", "title": "Nested API" } }
    }
  }
}
`)
	out, code := runCLIAllowFail(t, bin, dir, "validate")
	if code != 0 {
		t.Errorf("expected exit 0 (different dot-paths, no collision), got %d\n%s", code, out)
	}
}

// test2_17UnknownRelationshipKind covers 2.17.
func test2_17UnknownRelationshipKind(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeFile(t, dir+"/architecture.jsonc", `{
  "specification": { "elements": { "system": { "notation": "System" } } },
  "model": {
    "a": { "kind": "system", "title": "A" },
    "b": { "kind": "system", "title": "B" }
  },
  "relationships": [ { "from": "a", "to": "b", "kind": "foobar" } ]
}
`)
	out, code := runCLIAllowFail(t, bin, dir, "validate")
	if code != 1 {
		t.Fatalf("expected exit 1, got %d\n%s", code, out)
	}
	if !strings.Contains(out, `unknown relationship kind "foobar"`) {
		t.Errorf("expected an unknown-relationship-kind error, got: %s", out)
	}
}

// test2_18EmptyStringElementID covers 2.18.
func test2_18EmptyStringElementID(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeFile(t, dir+"/architecture.jsonc", `{
  "specification": { "elements": { "system": { "notation": "System" } } },
  "model": { "": { "kind": "system", "title": "Nameless" } }
}
`)
	out, code := runCLIAllowFail(t, bin, dir, "validate")
	if code != 1 {
		t.Fatalf("expected exit 1, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "must not be empty or whitespace") {
		t.Errorf("expected an empty-ID error, got: %s", out)
	}
}

// test2_19WhitespaceOnlyElementID covers 2.19.
func test2_19WhitespaceOnlyElementID(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeFile(t, dir+"/architecture.jsonc", `{
  "specification": { "elements": { "system": { "notation": "System" } } },
  "model": { " ": { "kind": "system", "title": "Blank" } }
}
`)
	out, code := runCLIAllowFail(t, bin, dir, "validate")
	if code != 1 {
		t.Fatalf("expected exit 1, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "must not be empty or whitespace") {
		t.Errorf("expected a whitespace-ID error, got: %s", out)
	}
}

// test2_20NullRootJSON covers 2.20.
func test2_20NullRootJSON(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeFile(t, dir+"/architecture.jsonc", `null`)
	out, code := runCLIAllowFail(t, bin, dir, "validate")
	if code != 2 {
		t.Fatalf("expected exit 2, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "null") {
		t.Errorf("expected an error mentioning the null root, got: %s", out)
	}
}

// test2_21InvalidFormatFlag covers 2.21: --format is validated globally in
// PersistentPreRunE, before any subcommand-specific logic runs.
func test2_21InvalidFormatFlag(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	out, code := runCLIAllowFail(t, bin, dir, "validate", "--format", "xml")
	if code != 1 {
		t.Fatalf("expected exit 1, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "unknown format") {
		t.Errorf("expected an 'unknown format' error, got: %s", out)
	}
}

// test2_22UppercaseFormatFlag covers 2.22: --format is case-insensitive.
func test2_22UppercaseFormatFlag(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	stdout, _, code := runCLISplit(t, bin, dir, "validate", "--format", "JSON")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d\nstdout: %s", code, stdout)
	}
	var report validateJSONReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Errorf("expected --format JSON (uppercase) to produce valid JSON output: %v\nstdout: %s", err, stdout)
	}
}

// test2_23MalformedJSON covers 2.23.
func test2_23MalformedJSON(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeFile(t, dir+"/architecture.jsonc", `{ "model": { "a": `)
	out, code := runCLIAllowFail(t, bin, dir, "validate")
	if code != 2 {
		t.Fatalf("expected exit 2, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "parsing") {
		t.Errorf("expected a parse error, got: %s", out)
	}
}

// test2_24FiveHundredElements covers 2.24: validate on a 500-element model
// completes well within the doc's <1s budget (generous 5s ceiling here to
// absorb CI/container scheduling noise, same rationale as 13.7).
func test2_24FiveHundredElements(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	var sb strings.Builder
	sb.WriteString(`{"specification":{"elements":{"system":{"notation":"System"}}},"model":{`)
	for i := 0; i < 500; i++ {
		if i > 0 {
			sb.WriteString(",")
		}
		fmt.Fprintf(&sb, `"elem%d":{"kind":"system","title":"Element %d"}`, i, i)
	}
	sb.WriteString(`}}`)
	writeFile(t, dir+"/architecture.jsonc", sb.String())

	out, code := runCLIAllowFail(t, bin, dir, "validate")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d\n%s", code, out)
	}
}

// test2_25DeepNesting covers 2.25: 6 levels of nesting is well within
// MaxElementDepth (50) and validates cleanly.
func test2_25DeepNesting(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeFile(t, dir+"/architecture.jsonc", `{
  "specification": { "elements": { "system": { "notation": "System", "container": true } } },
  "model": {
    "l1": { "kind": "system", "title": "L1", "children": {
      "l2": { "kind": "system", "title": "L2", "children": {
        "l3": { "kind": "system", "title": "L3", "children": {
          "l4": { "kind": "system", "title": "L4", "children": {
            "l5": { "kind": "system", "title": "L5", "children": {
              "l6": { "kind": "system", "title": "L6" }
            }}
          }}
        }}
      }}
    }}
  }
}
`)
	out, code := runCLIAllowFail(t, bin, dir, "validate")
	if code != 0 {
		t.Errorf("expected exit 0, got %d\n%s", code, out)
	}
}
