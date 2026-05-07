// Package main implements a test report generator that parses go test JSON output
// and coverage.out files to create comprehensive test and coverage reports.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	coverageFile := flag.String("coverage", "", "Path to coverage.out file (from go test -coverprofile)")
	previousReport := flag.String("previous", "", "Path to previous report.json for trend comparison")
	outputFormat := flag.String("format", "json", "Output format: json, markdown, html")
	flag.Parse()

	// Read test results from stdin (go test -json output)
	testResults, err := parseTestJSON(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing test results: %v\n", err)
		os.Exit(1)
	}

	// Read coverage data if provided
	var coverageData map[string]*CoverageInfo
	if *coverageFile != "" {
		var err error
		coverageData, err = parseCoverageFile(*coverageFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing coverage file: %v\n", err)
			os.Exit(1)
		}
	}

	// Generate report
	report := generateReport(testResults, coverageData)

	// Load previous report for comparison
	if *previousReport != "" {
		prev, err := loadReport(*previousReport)
		if err == nil {
			report.CompareTo(prev)
		}
	}

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
		fmt.Print(report.ToMarkdown())
	case "html":
		fmt.Print(report.ToHTML())
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
	Package      string
	Coverage     float64 // 0-100
	StmtCovered  int
	StmtTotal    int
	IsLowCoverage bool // < 80%
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

func generateReport(tests []TestResult, coverage map[string]*CoverageInfo) *Report {
	report := &Report{
		Timestamp: formatTimestamp(),
		Tests:     aggregateTests(tests),
		Coverage:  coverage,
	}
	return report
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
