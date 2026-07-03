// Command e2e-test-report-gen generates a line-accurate PASS/FAIL/SKIP
// AsciiDoc report for E2E-Test-Plan.adoc directly from `go test
// ./e2e/... -json` output, cross-referenced against internal/e2eplan's
// plan-ID-to-test mapping (see that package's doc comment for the naming
// convention new tests should follow).
//
// Usage:
//
//	go test ./e2e/... -json | go run ./scripts/e2e-test-report-gen [-plan path/to/E2E-Test-Plan.adoc]
//
// It is part of the tooling described in #519's "per CI statt manuell
// erzeugen" follow-up: this is the piece that lets CI generate
// src/docs/e2e-test-report-YYYY-MM-DD.adoc automatically instead of by
// hand. Wiring it into .github/workflows/go.yml (as an artifact, or as an
// auto-commit) is left to a separate, deliberate decision — see that
// issue's note that CI commits to main deserve their own sign-off.
//
// # Limitation: run the full suite
//
// Auto-detection (see internal/e2eplan) only sees test names that actually
// appear in the piped `go test -json` stream. Always pipe a full
// `go test ./e2e/...` run: a filtered/partial run (e.g. `-run TestFoo`)
// will misreport plan lines whose test exists but wasn't executed this
// time as "SKIP: not automated" rather than "NO DATA (mapped test did not
// run)" — the auto-detector has no static view of the test source, only of
// what ran. internal/e2eplan.Registry entries are exempt from this because
// they're a fixed, always-present map, not derived from the JSON stream.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/docToolchain/Bausteinsicht/internal/e2eplan"
)

func main() {
	planPath := flag.String("plan", "src/docs/spec/E2E-Test-Plan.adoc", "Path to E2E-Test-Plan.adoc")
	flag.Parse()

	lines, err := parsePlan(*planPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parsing plan: %v\n", err)
		os.Exit(1)
	}

	results, err := parseTestJSON(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parsing go test -json: %v\n", err)
		os.Exit(1)
	}

	autoDetected := detectPlanIDs(results)

	report := buildReport(lines, results, autoDetected)
	fmt.Print(report)
}

// planLine is one row from an E2E-Test-Plan.adoc section table.
type planLine struct {
	id          string
	description string
	section     string // e.g. "5. Views / Wildcards / Scope / Lifting"
}

var sectionHeadingRe = regexp.MustCompile(`^== (\d+[a-z]?)\.\s+(.+?)\s*\(\d+ tests?\)\s*$`)
var planRowRe = regexp.MustCompile(`^\|\s*(\d+[a-z]?\.\d+)\s*\|\s*(.+?)\s*\|`)

// parsePlan extracts every numbered test row from every "== N. Section
// (M tests)" table in the plan document. Anything outside such a section
// (Purpose, Automation Status, Before You Start, Test History, ...) is
// ignored because those sections' headings don't match sectionHeadingRe.
func parsePlan(path string) ([]planLine, error) {
	f, err := os.Open(path) // #nosec G304 -- path from CLI flag, local dev/CI tool
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var lines []planLine
	var currentSection string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		text := scanner.Text()
		if m := sectionHeadingRe.FindStringSubmatch(text); m != nil {
			currentSection = m[1] + ". " + m[2]
			continue
		}
		if currentSection == "" {
			continue
		}
		if m := planRowRe.FindStringSubmatch(text); m != nil {
			lines = append(lines, planLine{id: m[1], description: m[2], section: currentSection})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

// testResult is the outcome of one qualified go test name (e.g.
// "TestFoo/5.1_ExactMatch"), as reported by `go test -json`.
type testResult struct {
	Name    string
	Outcome string // PASS, FAIL, SKIP
}

type testEvent struct {
	Action string `json:"Action"`
	Test   string `json:"Test"`
}

// parseTestJSON reads `go test -json` output and returns the final
// outcome of every test/subtest that ran (package-level events, which have
// an empty Test field, are ignored).
func parseTestJSON(r io.Reader) (map[string]testResult, error) {
	decoder := json.NewDecoder(r)
	results := make(map[string]testResult)
	for {
		var ev testEvent
		err := decoder.Decode(&ev)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if ev.Test == "" {
			continue
		}
		switch ev.Action {
		case "pass":
			results[ev.Test] = testResult{Name: ev.Test, Outcome: "PASS"}
		case "fail":
			results[ev.Test] = testResult{Name: ev.Test, Outcome: "FAIL"}
		case "skip":
			results[ev.Test] = testResult{Name: ev.Test, Outcome: "SKIP"}
		}
	}
	return results, nil
}

// subtestPlanIDRe matches the first path segment after a top-level test
// name, e.g. the "5.1_ExactMatch" in "TestFoo/5.1_ExactMatch" or
// "TestFoo/5.1_ExactMatch/NestedCase" — nested subtests fold into the same
// planID as their direct parent, since Go already propagates a child
// failure up to the parent's own result.
var subtestPlanIDRe = regexp.MustCompile(`^(\d+[a-z]?\.\d+)_`)

// detectPlanIDs finds every test name following the "<planID>_..." subtest
// naming convention and maps planID -> qualified test name.
func detectPlanIDs(results map[string]testResult) map[string][]string {
	detected := make(map[string][]string)
	for name := range results {
		parts := strings.Split(name, "/")
		if len(parts) < 2 {
			continue
		}
		m := subtestPlanIDRe.FindStringSubmatch(parts[1])
		if m == nil {
			continue
		}
		id := m[1]
		// Only record the direct-child level (parts[0]/parts[1]), not
		// deeper nested subtests, to avoid double-counting.
		qualified := parts[0] + "/" + parts[1]
		found := false
		for _, existing := range detected[id] {
			if existing == qualified {
				found = true
				break
			}
		}
		if !found {
			detected[id] = append(detected[id], qualified)
		}
	}
	return detected
}

const (
	statusPass         = "PASS"
	statusFail         = "FAIL"
	statusRuntimeSkip  = "SKIP (runtime)"
	statusNotAutomated = "SKIP: not automated"
	statusNoData       = "NO DATA (mapped test did not run)"
)

// resolveStatus determines a single plan line's status from its mapped
// test name(s) (registry entries take priority over auto-detection when
// both exist for the same ID) and the actual go test -json results.
func resolveStatus(id string, autoDetected map[string][]string, results map[string]testResult) string {
	names := e2eplan.Registry[id]
	if len(names) == 0 {
		names = autoDetected[id]
	}
	if len(names) == 0 {
		return statusNotAutomated
	}

	sawAny := false
	worst := statusPass
	for _, name := range names {
		res, ok := results[name]
		if !ok {
			continue
		}
		sawAny = true
		switch res.Outcome {
		case "FAIL":
			return statusFail
		case "SKIP":
			worst = statusRuntimeSkip
		}
	}
	if !sawAny {
		return statusNoData
	}
	return worst
}

// buildReport renders the full AsciiDoc report: a per-section summary
// table followed by a per-line detail table (the "3.15 PASS", "10.8 FAIL"
// granularity #519 asked for).
func buildReport(lines []planLine, results map[string]testResult, autoDetected map[string][]string) string {
	type sectionStats struct {
		name                                                 string
		total, pass, fail, runtimeSkip, notAutomated, noData int
	}
	var sectionOrder []string
	stats := make(map[string]*sectionStats)

	statusByID := make(map[string]string, len(lines))
	for _, l := range lines {
		status := resolveStatus(l.id, autoDetected, results)
		statusByID[l.id] = status

		s, ok := stats[l.section]
		if !ok {
			s = &sectionStats{name: l.section}
			stats[l.section] = s
			sectionOrder = append(sectionOrder, l.section)
		}
		s.total++
		switch status {
		case statusPass:
			s.pass++
		case statusFail:
			s.fail++
		case statusRuntimeSkip:
			s.runtimeSkip++
		case statusNotAutomated:
			s.notAutomated++
		case statusNoData:
			s.noData++
		}
	}

	var b strings.Builder
	b.WriteString("== Summary (generated)\n\n")
	b.WriteString(`[cols="1,3,1,1,1,1,1,1"]` + "\n|===\n")
	b.WriteString("| Section | Tests | Pass | Fail | Runtime Skip | Not Automated | No Data\n\n")
	for _, sec := range sectionOrder {
		s := stats[sec]
		fmt.Fprintf(&b, "| %s | %d | %d | %d | %d | %d | %d\n",
			s.name, s.total, s.pass, s.fail, s.runtimeSkip, s.notAutomated, s.noData)
	}
	b.WriteString("|===\n\n")

	b.WriteString("== Detail (generated)\n\n")
	b.WriteString(`[cols="1,4,2"]` + "\n|===\n")
	b.WriteString("| # | Test | Result\n\n")
	sorted := make([]planLine, len(lines))
	copy(sorted, lines)
	sort.SliceStable(sorted, func(i, j int) bool { return planIDLess(sorted[i].id, sorted[j].id) })
	for _, l := range sorted {
		fmt.Fprintf(&b, "| %s | %s | %s\n", l.id, l.description, statusByID[l.id])
	}
	b.WriteString("|===\n")

	return b.String()
}

// planIDLess orders IDs like "8b.4" and "8.12" sensibly: numeric section
// number first (so "20.1" sorts after "8.1", not before it as a plain
// string compare would — '2' < '8' lexicographically), then the optional
// letter suffix ("8" before "8b"), then numeric line number (so "8.2"
// sorts before "8.12").
func planIDLess(a, b string) bool {
	na, la, ta := splitPlanID(a)
	nb, lb, tb := splitPlanID(b)
	if na != nb {
		return na < nb
	}
	if la != lb {
		return la < lb
	}
	return ta < tb
}

var planIDRe = regexp.MustCompile(`^(\d+)([a-z]?)\.(\d+)$`)

// splitPlanID splits a plan ID into its numeric section, optional letter
// suffix, and numeric line number, e.g. "8b.4" -> (8, "b", 4).
func splitPlanID(id string) (sectionNum int, sectionLetter string, line int) {
	m := planIDRe.FindStringSubmatch(id)
	if m == nil {
		return 0, id, 0
	}
	// planIDRe guarantees m[1] and m[3] are all digits, so these errors are unreachable.
	sectionNum, _ = strconv.Atoi(m[1])
	line, _ = strconv.Atoi(m[3])
	return sectionNum, m[2], line
}
