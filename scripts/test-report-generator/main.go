// Package main implements a test report generator that parses go test JSON output
// and coverage.out files to create comprehensive test and coverage reports.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
)

func main() {
	coverageFile := flag.String("coverage", "", "Path to coverage.out file (from go test -coverprofile)")
	previousReport := flag.String("previous", "", "Path to previous report.json for trend comparison")
	outputFormat := flag.String("format", "json", "Output format: json, markdown, html")
	slowThreshold := flag.Float64("slow-threshold", 2.0, "Performance regression threshold (x times average)")
	sourceRoot := flag.String("source-root", ".", "Root directory to resolve source file paths")
	flag.Parse()

	// Read test results from stdin (go test -json output)
	testResults, err := parseTestJSON(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing test results: %v\n", err)
		os.Exit(1)
	}

	// Read coverage data if provided
	var coverageData map[string]*CoverageInfo
	var coverageDetails *CoverageDetails
	if *coverageFile != "" {
		var err error
		coverageData, coverageDetails, err = parseCoverageFileDetailed(*coverageFile, *sourceRoot)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing coverage file: %v\n", err)
			os.Exit(1)
		}
	}

	// Generate report
	report := generateReport(testResults, coverageData, coverageDetails)

	// Detect performance regressions
	regressions := report.DetectPerformanceRegression(*slowThreshold)
	if len(regressions) > 0 {
		// Add to slowest tests list if there are regressions
		report.SlowestTests = append(report.SlowestTests, regressions...)
	}

	// Load previous report for comparison
	if *previousReport != "" {
		prev, err := loadReport(*previousReport)
		if err == nil {
			report.CompareTo(prev)
		}
	}

	// Compute derived fields before output
	report.computeDerivedFields()

	// Output report in requested format
	switch *outputFormat {
	case "json":
		output, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(output))
	case "markdown":
		fmt.Print(report.RenderMarkdown())
	case "html":
		fmt.Print(report.RenderHTML())
	default:
		fmt.Fprintf(os.Stderr, "Unknown format: %s\n", *outputFormat)
		os.Exit(1)
	}
}

// parseTestJSON reads go test -json output from reader
func parseTestJSON(r io.Reader) ([]TestResult, error) {
	decoder := json.NewDecoder(r)
	var results []TestResult
	var lastTest *TestResult

	for {
		var event TestEvent
		err := decoder.Decode(&event)
		if err == io.EOF {
			if lastTest != nil && lastTest.Result == "" {
				results = append(results, *lastTest)
			}
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decode error: %w", err)
		}

		switch event.Action {
		case "run":
			lastTest = &TestResult{
				Package: event.Package,
				Test:    event.Test,
				Result:  "",
			}
		case "pass":
			if lastTest != nil && lastTest.Test == event.Test {
				lastTest.Result = "PASS"
				lastTest.Elapsed = event.Elapsed
				results = append(results, *lastTest)
			}
		case "fail":
			if lastTest != nil && lastTest.Test == event.Test {
				lastTest.Result = "FAIL"
				lastTest.Elapsed = event.Elapsed
				lastTest.Output = event.Output
				results = append(results, *lastTest)
			}
		case "skip":
			if lastTest != nil && lastTest.Test == event.Test {
				lastTest.Result = "SKIP"
				lastTest.Elapsed = event.Elapsed
				results = append(results, *lastTest)
			}
		}
	}

	return results, nil
}

// TestEvent represents a single line from go test -json output
type TestEvent struct {
	Action  string  `json:"Action"`
	Package string  `json:"Package"`
	Test    string  `json:"Test"`
	Elapsed float64 `json:"Elapsed"`
	Output  string  `json:"Output"`
}

// TestResult represents a test execution result
type TestResult struct {
	Package string
	Test    string
	Result  string  // PASS, FAIL, SKIP
	Elapsed float64 // seconds
	Output  string  // error output
}

// CoverageInfo represents coverage for a package
type CoverageInfo struct {
	Package       string  `json:"package"`
	Coverage      float64 `json:"coverage"` // 0-100
	StmtCovered   int     `json:"stmt_covered"`
	StmtTotal     int     `json:"stmt_total"`
	IsLowCoverage bool    `json:"is_low_coverage"` // < 80%
}

func loadReport(path string) (*Report, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var report Report
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, err
	}
	return &report, nil
}

func generateReport(tests []TestResult, coverage map[string]*CoverageInfo, details *CoverageDetails) *Report {
	stats := aggregateTests(tests)

	// Find slowest tests
	slowestTests := findSlowestTests(tests, 10)

	report := &Report{
		Timestamp:    formatTimestamp(),
		Tests:        stats,
		Coverage:     coverage,
		SlowestTests: slowestTests,
		Details:      details,
	}
	return report
}

// findSlowestTests returns the N slowest tests
func findSlowestTests(tests []TestResult, n int) []SlowTest {
	var slowTests []SlowTest
	for _, t := range tests {
		if t.Result == "PASS" || t.Result == "FAIL" { // Count passed and failed, skip skipped
			slowTests = append(slowTests, SlowTest{
				Package: t.Package,
				Test:    t.Test,
				Elapsed: t.Elapsed,
			})
		}
	}

	// Sort by elapsed time, descending
	sort.Slice(slowTests, func(i, j int) bool {
		return slowTests[i].Elapsed > slowTests[j].Elapsed
	})

	// Return top N
	if len(slowTests) > n {
		return slowTests[:n]
	}
	return slowTests
}

func aggregateTests(tests []TestResult) TestStats {
	stats := TestStats{
		ByPackage: make(map[string]*PackageStats),
		ByType:    make(map[string]*TypeStats),
	}

	for _, t := range tests {
		// Count totals
		stats.Total++
		if t.Result == "PASS" {
			stats.Passed++
		} else if t.Result == "FAIL" {
			stats.Failed++
		} else if t.Result == "SKIP" {
			stats.Skipped++
		}
		stats.TotalTime += t.Elapsed

		// By package
		if stats.ByPackage[t.Package] == nil {
			stats.ByPackage[t.Package] = &PackageStats{}
		}
		pkg := stats.ByPackage[t.Package]
		pkg.Total++
		pkg.Package = t.Package
		if t.Result == "PASS" {
			pkg.Passed++
		} else if t.Result == "FAIL" {
			pkg.Failed++
		} else if t.Result == "SKIP" {
			pkg.Skipped++
		}
		pkg.TotalTime += t.Elapsed

		// By type
		testType := classifyTestType(t.Test)
		if stats.ByType[testType] == nil {
			stats.ByType[testType] = &TypeStats{Type: testType}
		}
		typ := stats.ByType[testType]
		typ.Total++
		if t.Result == "PASS" {
			typ.Passed++
		} else if t.Result == "FAIL" {
			typ.Failed++
		} else if t.Result == "SKIP" {
			typ.Skipped++
		}
	}

	// Calculate pass rates
	if stats.Total > 0 {
		stats.PassRate = float64(stats.Passed) * 100 / float64(stats.Total)
	}
	for _, pkg := range stats.ByPackage {
		if pkg.Total > 0 {
			pkg.PassRate = float64(pkg.Passed) * 100 / float64(pkg.Total)
		}
	}
	for _, typ := range stats.ByType {
		if typ.Total > 0 {
			typ.PassRate = float64(typ.Passed) * 100 / float64(typ.Total)
		}
	}

	return stats
}

func classifyTestType(testName string) string {
	if isRegressionTest(testName) {
		return "regression"
	}
	if isIntegrationTest(testName) {
		return "integration"
	}
	return "unit"
}

func isRegressionTest(name string) bool {
	// Tests with "Regression", "Regression" suffix, or from specific issues
	return contains(name, "Regression") ||
		contains(name, "Stale") ||
		contains(name, "Boundary") ||
		contains(name, "SearchOrder")
}

func isIntegrationTest(name string) bool {
	return contains(name, "Integration") || contains(name, "E2E")
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func formatTimestamp() string {
	// Simple timestamp format
	return "2006-01-02T15:04:05Z"
}
