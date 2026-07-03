package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestParseTestJSON(t *testing.T) {
	input := strings.Join([]string{
		`{"Action":"run","Package":"pkg","Test":"TestFoo"}`,
		`{"Action":"run","Package":"pkg","Test":"TestFoo/5.1_Bar"}`,
		`{"Action":"pass","Package":"pkg","Test":"TestFoo/5.1_Bar"}`,
		`{"Action":"run","Package":"pkg","Test":"TestFoo/5.2_Baz"}`,
		`{"Action":"fail","Package":"pkg","Test":"TestFoo/5.2_Baz"}`,
		`{"Action":"fail","Package":"pkg","Test":"TestFoo"}`,
		`{"Action":"pass","Package":"pkg","Test":""}`, // package-level summary, must be ignored
		"",
	}, "\n")

	results, err := parseTestJSON(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parseTestJSON: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d: %+v", len(results), results)
	}
	if results["TestFoo/5.1_Bar"].Outcome != "PASS" {
		t.Errorf("TestFoo/5.1_Bar: got %q, want PASS", results["TestFoo/5.1_Bar"].Outcome)
	}
	if results["TestFoo/5.2_Baz"].Outcome != "FAIL" {
		t.Errorf("TestFoo/5.2_Baz: got %q, want FAIL", results["TestFoo/5.2_Baz"].Outcome)
	}
	if results["TestFoo"].Outcome != "FAIL" {
		t.Errorf("TestFoo: got %q, want FAIL", results["TestFoo"].Outcome)
	}
	if _, ok := results[""]; ok {
		t.Error("package-level event (empty Test field) should not produce a result")
	}
}

func TestDetectPlanIDs(t *testing.T) {
	results := map[string]testResult{
		"TestFoo":                           {Outcome: "PASS"},
		"TestFoo/5.1_ExactMatch":            {Outcome: "PASS"},
		"TestFoo/5.1_ExactMatch/Nested":     {Outcome: "PASS"}, // deeper nesting must fold into 5.1, not a separate entry
		"TestBar/notAPlanID":                {Outcome: "PASS"},
		"TestBar/8b.4_UnknownDiagramFmt":    {Outcome: "FAIL"},
		"TestBaz/10.9_JSONInjectionInTitle": {Outcome: "PASS"},
	}

	detected := detectPlanIDs(results)

	if got := detected["5.1"]; len(got) != 1 || got[0] != "TestFoo/5.1_ExactMatch" {
		t.Errorf("5.1: got %v, want exactly [TestFoo/5.1_ExactMatch]", got)
	}
	if got := detected["8b.4"]; len(got) != 1 || got[0] != "TestBar/8b.4_UnknownDiagramFmt" {
		t.Errorf("8b.4: got %v, want exactly [TestBar/8b.4_UnknownDiagramFmt]", got)
	}
	if got := detected["10.9"]; len(got) != 1 || got[0] != "TestBaz/10.9_JSONInjectionInTitle" {
		t.Errorf("10.9: got %v, want exactly [TestBaz/10.9_JSONInjectionInTitle]", got)
	}
	if _, ok := detected["notAPlanID"]; ok {
		t.Error("non-planID-prefixed subtest should not be detected")
	}
}

func TestResolveStatus(t *testing.T) {
	results := map[string]testResult{
		"TestX/99.1_Pass": {Outcome: "PASS"},
		"TestX/99.2_Fail": {Outcome: "FAIL"},
		"TestX/99.3_Skip": {Outcome: "SKIP"},
	}
	autoDetected := map[string][]string{
		"99.1": {"TestX/99.1_Pass"},
		"99.2": {"TestX/99.2_Fail"},
		"99.3": {"TestX/99.3_Skip"},
	}

	cases := []struct {
		id   string
		want string
	}{
		{"99.1", statusPass},
		{"99.2", statusFail},
		{"99.3", statusRuntimeSkip},
		{"99.99", statusNotAutomated}, // not in registry or autoDetected
	}
	for _, tc := range cases {
		if got := resolveStatus(tc.id, autoDetected, results); got != tc.want {
			t.Errorf("resolveStatus(%q) = %q, want %q", tc.id, got, tc.want)
		}
	}

	// Mapped but the test didn't actually run in this invocation -> NO DATA.
	autoDetected["99.4"] = []string{"TestX/99.4_NeverRan"}
	if got := resolveStatus("99.4", autoDetected, results); got != statusNoData {
		t.Errorf("resolveStatus(mapped-but-not-run) = %q, want %q", got, statusNoData)
	}
}

func TestSplitPlanIDAndOrdering(t *testing.T) {
	ids := []string{"9.1", "8b.4", "8.12", "8.2", "20.1", "1.1"}
	sort.SliceStable(ids, func(i, j int) bool { return planIDLess(ids[i], ids[j]) })

	want := []string{"1.1", "8.2", "8.12", "8b.4", "9.1", "20.1"}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("sorted order = %v, want %v", ids, want)
		}
	}
}

func TestParsePlan(t *testing.T) {
	fixture := `= Fixture Test Plan

== Purpose

Not a numbered section — must be ignored.

| 99.1 | Should be ignored | this row is outside any "== N. Section (M tests)" block |

== 1. Basic Workflow (2 tests)

[cols="1,4,5,5"]
|===
| # | Test | Steps | Expected

| 1.1 | Init in fresh dir | Run init | Success
| 1.2 | Double init | Run init twice | Error
|===

== 5. Views (1 tests)

[cols="1,4,5,5"]
|===
| # | Test | Include/Exclude | Expected

| 5.1 | Exact match | ["webshop"] | Only webshop
|===
`
	dir := t.TempDir()
	path := filepath.Join(dir, "plan.adoc")
	if err := os.WriteFile(path, []byte(fixture), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	lines, err := parsePlan(path)
	if err != nil {
		t.Fatalf("parsePlan: %v", err)
	}
	if len(lines) != 3 {
		t.Fatalf("expected 3 plan lines, got %d: %+v", len(lines), lines)
	}
	if lines[0].id != "1.1" || lines[0].section != "1. Basic Workflow" {
		t.Errorf("lines[0] = %+v, want id=1.1 section=\"1. Basic Workflow\"", lines[0])
	}
	if lines[2].id != "5.1" || lines[2].description != "Exact match" {
		t.Errorf("lines[2] = %+v, want id=5.1 description=\"Exact match\"", lines[2])
	}
	for _, l := range lines {
		if l.id == "99.1" {
			t.Error("row outside a numbered section heading must not be parsed")
		}
	}
}

func TestBuildReport_SectionCounts(t *testing.T) {
	lines := []planLine{
		{id: "1.1", description: "A", section: "1. Sec"},
		{id: "1.2", description: "B", section: "1. Sec"},
		{id: "2.1", description: "C", section: "2. Other"},
	}
	results := map[string]testResult{
		"TestX/1.1_A": {Outcome: "PASS"},
	}
	autoDetected := map[string][]string{
		"1.1": {"TestX/1.1_A"},
	}

	report := buildReport(lines, results, autoDetected)

	if !strings.Contains(report, "| 1. Sec | 2 | 1 | 0 | 0 | 1 | 0") {
		t.Errorf("expected section summary row for '1. Sec' with 1 pass / 1 not-automated, got:\n%s", report)
	}
	if !strings.Contains(report, "| 2. Other | 1 | 0 | 0 | 0 | 1 | 0") {
		t.Errorf("expected section summary row for '2. Other' fully not-automated, got:\n%s", report)
	}
	if !strings.Contains(report, "| 1.1 | A | PASS") {
		t.Errorf("expected detail row for 1.1, got:\n%s", report)
	}
	if !strings.Contains(report, "| 2.1 | C | SKIP: not automated") {
		t.Errorf("expected detail row for 2.1, got:\n%s", report)
	}
}
