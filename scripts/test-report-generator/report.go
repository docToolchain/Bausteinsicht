package main

import (
	"fmt"
	"sort"
)

// Report represents a comprehensive test and coverage report
type Report struct {
	Timestamp       string                     `json:"timestamp"`
	Tests           TestStats                  `json:"tests"`
	Coverage        map[string]*CoverageInfo   `json:"coverage"`
	Delta           *ReportDelta               `json:"delta,omitempty"`
	LowCoverageList []string                   `json:"low_coverage_packages,omitempty"`
	SlowestTests    []SlowTest                 `json:"slowest_tests,omitempty"`
	RegressionTests *RegressionTestStats       `json:"regression_tests,omitempty"`
}

// TestStats aggregates test results
type TestStats struct {
	Total     int                    `json:"total"`
	Passed    int                    `json:"passed"`
	Failed    int                    `json:"failed"`
	Skipped   int                    `json:"skipped"`
	PassRate  float64                `json:"pass_rate"`
	TotalTime float64                `json:"total_time_seconds"`
	ByPackage map[string]*PackageStats `json:"by_package"`
	ByType    map[string]*TypeStats    `json:"by_type"`
}

// PackageStats aggregates test stats per package
type PackageStats struct {
	Package   string  `json:"package"`
	Total     int     `json:"total"`
	Passed    int     `json:"passed"`
	Failed    int     `json:"failed"`
	Skipped   int     `json:"skipped"`
	PassRate  float64 `json:"pass_rate"`
	TotalTime float64 `json:"total_time_seconds"`
}

// TypeStats aggregates test stats by test type
type TypeStats struct {
	Type     string  `json:"type"` // unit, integration, regression
	Total    int     `json:"total"`
	Passed   int     `json:"passed"`
	Failed   int     `json:"failed"`
	Skipped  int     `json:"skipped"`
	PassRate float64 `json:"pass_rate"`
}

// SlowTest represents a slow test
type SlowTest struct {
	Package string  `json:"package"`
	Test    string  `json:"test"`
	Elapsed float64 `json:"elapsed_seconds"`
}

// RegressionTestStats tracks regression tests
type RegressionTestStats struct {
	Total  int     `json:"total"`
	Passed int     `json:"passed"`
	Failed int     `json:"failed"`
	Skipped int    `json:"skipped"`
	PassRate float64 `json:"pass_rate"`
}

// ReportDelta represents changes from previous report
type ReportDelta struct {
	PassRateChange   float64 `json:"pass_rate_change"` // e.g., +2.5 or -1.0
	CoverageChange   float64 `json:"coverage_change"`
	PerformanceChange float64 `json:"performance_change"` // % faster/slower
	NewFailures      int     `json:"new_failures"`
	ResolvedFailures int     `json:"resolved_failures"`
}

// CompareTo compares this report with a previous one and sets Delta
func (r *Report) CompareTo(prev *Report) {
	if prev == nil {
		return
	}

	r.Delta = &ReportDelta{}

	// Pass rate change
	if prev.Tests.Total > 0 && r.Tests.Total > 0 {
		r.Delta.PassRateChange = r.Tests.PassRate - prev.Tests.PassRate
	}

	// Coverage change
	prevCoverage := calculateOverallCoverage(prev.Coverage)
	currCoverage := calculateOverallCoverage(r.Coverage)
	r.Delta.CoverageChange = currCoverage - prevCoverage

	// Performance change
	if prev.Tests.TotalTime > 0 && r.Tests.TotalTime > 0 {
		change := (r.Tests.TotalTime - prev.Tests.TotalTime) / prev.Tests.TotalTime * 100
		r.Delta.PerformanceChange = change
	}

	// Failure tracking
	r.Delta.NewFailures = r.Tests.Failed - prev.Tests.Failed
	if r.Delta.NewFailures < 0 {
		r.Delta.NewFailures = 0
		r.Delta.ResolvedFailures = prev.Tests.Failed - r.Tests.Failed
	}
}

// ToMarkdown converts the report to GitHub Markdown
func (r *Report) ToMarkdown() string {
	out := "## 📊 Test Report\n\n"

	// Summary
	out += "### Summary\n"
	out += fmt.Sprintf("| Metric | Value |\n")
	out += fmt.Sprintf("|--------|-------|\n")
	out += fmt.Sprintf("| Total Tests | %d |\n", r.Tests.Total)
	out += fmt.Sprintf("| ✅ Passed | %d (%.1f%%) |\n", r.Tests.Passed, r.Tests.PassRate)
	out += fmt.Sprintf("| ❌ Failed | %d |\n", r.Tests.Failed)
	out += fmt.Sprintf("| ⏭️ Skipped | %d |\n", r.Tests.Skipped)
	out += fmt.Sprintf("| ⏱️ Duration | %.2fs |\n", r.Tests.TotalTime)
	out += fmt.Sprintf("| 📦 Coverage | %.1f%% |\n\n", calculateOverallCoverage(r.Coverage))

	// Deltas if available
	if r.Delta != nil {
		out += "### Changes from Previous Run\n"
		out += fmt.Sprintf("- Pass Rate: %+.1f%%\n", r.Delta.PassRateChange)
		out += fmt.Sprintf("- Coverage: %+.1f%%\n", r.Delta.CoverageChange)
		if r.Delta.PerformanceChange >= 0 {
			out += fmt.Sprintf("- Performance: %+.1f%% slower ⚠️\n", r.Delta.PerformanceChange)
		} else {
			out += fmt.Sprintf("- Performance: %.1f%% faster ✨\n", -r.Delta.PerformanceChange)
		}
		out += fmt.Sprintf("- New Failures: %d\n\n", r.Delta.NewFailures)
	}

	// By Package
	out += "### By Package\n"
	out += "| Package | Tests | Passed | Failed | Skip | Pass Rate |\n"
	out += "|---------|-------|--------|--------|------|----------|\n"
	for _, pkg := range sortedPackages(r.Tests.ByPackage) {
		stats := r.Tests.ByPackage[pkg]
		out += fmt.Sprintf("| `%s` | %d | %d | %d | %d | %.1f%% |\n",
			pkg, stats.Total, stats.Passed, stats.Failed, stats.Skipped, stats.PassRate)
	}
	out += "\n"

	// By Test Type
	if len(r.Tests.ByType) > 0 {
		out += "### By Test Type\n"
		out += "| Type | Tests | Passed | Failed | Skip | Pass Rate |\n"
		out += "|------|-------|--------|--------|------|----------|\n"
		for _, testType := range []string{"unit", "integration", "regression"} {
			if stats, ok := r.Tests.ByType[testType]; ok {
				out += fmt.Sprintf("| %s | %d | %d | %d | %d | %.1f%% |\n",
					testType, stats.Total, stats.Passed, stats.Failed, stats.Skipped, stats.PassRate)
			}
		}
		out += "\n"
	}

	// Low Coverage Packages
	if len(r.LowCoverageList) > 0 {
		out += "### ⚠️ Low Coverage Packages (<80%)\n"
		for _, pkg := range r.LowCoverageList {
			if info, ok := r.Coverage[pkg]; ok {
				out += fmt.Sprintf("- `%s`: %.1f%% (%d/%d statements)\n",
					pkg, info.Coverage, info.StmtCovered, info.StmtTotal)
			}
		}
		out += "\n"
	}

	// Slowest Tests
	if len(r.SlowestTests) > 0 {
		out += "### 🐢 Slowest Tests (Top 5)\n"
		for i, test := range r.SlowestTests {
			if i >= 5 {
				break
			}
			out += fmt.Sprintf("%d. `%s::%s` — %.2fs\n", i+1, test.Package, test.Test, test.Elapsed)
		}
		out += "\n"
	}

	return out
}

// ToHTML converts the report to an interactive HTML dashboard
func (r *Report) ToHTML() string {
	html := `<!DOCTYPE html>
<html>
<head>
	<title>Test Report</title>
	<script src="https://cdn.plot.ly/plotly-latest.min.js"></script>
	<style>
		body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; margin: 20px; }
		.summary { display: grid; grid-template-columns: repeat(3, 1fr); gap: 20px; margin-bottom: 30px; }
		.metric { background: #f5f5f5; padding: 20px; border-radius: 8px; border-left: 4px solid #0366d6; }
		.metric-value { font-size: 32px; font-weight: bold; }
		.metric-label { color: #666; margin-top: 5px; }
		.chart { margin-bottom: 40px; }
		table { width: 100%; border-collapse: collapse; margin-bottom: 30px; }
		th, td { padding: 12px; text-align: left; border-bottom: 1px solid #ddd; }
		th { background: #f6f8fa; font-weight: 600; }
		tr:hover { background: #f9f9f9; }
	</style>
</head>
<body>
	<h1>📊 Test Report</h1>
	<div class="summary">
		<div class="metric">
			<div class="metric-value">%d</div>
			<div class="metric-label">Total Tests</div>
		</div>
		<div class="metric">
			<div class="metric-value" style="color: #28a745;">%.1f%%</div>
			<div class="metric-label">Pass Rate</div>
		</div>
		<div class="metric">
			<div class="metric-value" style="color: #0366d6;">%.1f%%</div>
			<div class="metric-label">Coverage</div>
		</div>
	</div>

	<div class="chart">
		<h2>Test Results by Package</h2>
		<div id="packageChart"></div>
	</div>

	<h2>Details by Package</h2>
	<table>
		<tr><th>Package</th><th>Tests</th><th>Passed</th><th>Failed</th><th>Skip</th><th>Pass Rate</th></tr>
`

	for _, pkg := range sortedPackages(r.Tests.ByPackage) {
		stats := r.Tests.ByPackage[pkg]
		html += fmt.Sprintf("<tr><td>%s</td><td>%d</td><td>%d</td><td>%d</td><td>%d</td><td>%.1f%%</td></tr>\n",
			pkg, stats.Total, stats.Passed, stats.Failed, stats.Skipped, stats.PassRate)
	}

	html += `	</table>
</body>
</html>`

	return fmt.Sprintf(html, r.Tests.Total, r.Tests.PassRate, calculateOverallCoverage(r.Coverage))
}

func calculateOverallCoverage(coverage map[string]*CoverageInfo) float64 {
	if len(coverage) == 0 {
		return 0
	}
	var totalStmt, coveredStmt int
	for _, info := range coverage {
		totalStmt += info.StmtTotal
		coveredStmt += info.StmtCovered
	}
	if totalStmt == 0 {
		return 0
	}
	return float64(coveredStmt) * 100 / float64(totalStmt)
}

func sortedPackages(pkgs map[string]*PackageStats) []string {
	var names []string
	for name := range pkgs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
