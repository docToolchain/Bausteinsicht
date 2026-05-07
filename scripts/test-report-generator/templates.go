package main

import (
	"fmt"
	"sort"
	"strings"
)

// RenderMarkdown generates a comprehensive Markdown report
func (r *Report) RenderMarkdown() string {
	var buf strings.Builder

	buf.WriteString("## 📊 Test Report\n\n")

	// Summary metrics
	buf.WriteString("### Summary Metrics\n\n")
	buf.WriteString(fmt.Sprintf("|  |  |\n|---|---|\n"))
	buf.WriteString(fmt.Sprintf("| **Total Tests** | %d |\n", r.Tests.Total))
	buf.WriteString(fmt.Sprintf("| **Pass Rate** | %.1f%% (%d ✅ / %d ❌) |\n", r.Tests.PassRate, r.Tests.Passed, r.Tests.Failed))
	buf.WriteString(fmt.Sprintf("| **Skipped** | %d ⏭️ |\n", r.Tests.Skipped))
	buf.WriteString(fmt.Sprintf("| **Duration** | %.2fs ⏱️ |\n", r.Tests.TotalTime))
	buf.WriteString(fmt.Sprintf("| **Overall Coverage** | %.1f%% 📦 |\n\n", calculateOverallCoverage(r.Coverage)))

	// Delta section
	if r.Delta != nil {
		buf.WriteString("### Changes from Previous Run\n\n")
		if r.Delta.PassRateChange > 0 {
			buf.WriteString(fmt.Sprintf("- ✅ Pass Rate: **+%.1f%%** (improved)\n", r.Delta.PassRateChange))
		} else if r.Delta.PassRateChange < 0 {
			buf.WriteString(fmt.Sprintf("- ⚠️ Pass Rate: **%.1f%%** (declined)\n", r.Delta.PassRateChange))
		}

		if r.Delta.CoverageChange > 0 {
			buf.WriteString(fmt.Sprintf("- 📈 Coverage: **+%.1f%%** (improved)\n", r.Delta.CoverageChange))
		} else if r.Delta.CoverageChange < 0 {
			buf.WriteString(fmt.Sprintf("- 📉 Coverage: **%.1f%%** (declined)\n", r.Delta.CoverageChange))
		}

		if r.Delta.PerformanceChange > 0 {
			buf.WriteString(fmt.Sprintf("- ⚠️ Performance: **+%.1f%% slower**\n", r.Delta.PerformanceChange))
		} else if r.Delta.PerformanceChange < 0 {
			buf.WriteString(fmt.Sprintf("- ✨ Performance: **%.1f%% faster**\n", -r.Delta.PerformanceChange))
		}

		if r.Delta.NewFailures > 0 {
			buf.WriteString(fmt.Sprintf("- 🆕 New Failures: **%d**\n", r.Delta.NewFailures))
		}
		if r.Delta.ResolvedFailures > 0 {
			buf.WriteString(fmt.Sprintf("- ✅ Resolved Failures: **%d**\n\n", r.Delta.ResolvedFailures))
		}
	}

	// By Package
	buf.WriteString("### Test Results by Package\n\n")
	buf.WriteString("| Package | Tests | ✅ Passed | ❌ Failed | ⏭️ Skipped | Pass Rate |\n")
	buf.WriteString("|---------|-------|---------|---------|----------|----------|\n")

	packages := sortedPackages(r.Tests.ByPackage)
	for _, pkg := range packages {
		stats := r.Tests.ByPackage[pkg]
		passRate := fmt.Sprintf("%.1f%%", stats.PassRate)
		if stats.PassRate < 80 {
			passRate = "⚠️ " + passRate
		}
		buf.WriteString(fmt.Sprintf("| `%s` | %d | %d | %d | %d | %s |\n",
			pkg, stats.Total, stats.Passed, stats.Failed, stats.Skipped, passRate))
	}
	buf.WriteString("\n")

	// By Test Type
	if len(r.Tests.ByType) > 0 {
		buf.WriteString("### Test Results by Type\n\n")
		buf.WriteString("| Type | Tests | ✅ Passed | ❌ Failed | ⏭️ Skipped | Pass Rate |\n")
		buf.WriteString("|------|-------|---------|---------|----------|----------|\n")

		for _, testType := range []string{"unit", "integration", "regression"} {
			if stats, ok := r.Tests.ByType[testType]; ok {
				buf.WriteString(fmt.Sprintf("| **%s** | %d | %d | %d | %d | %.1f%% |\n",
					testType, stats.Total, stats.Passed, stats.Failed, stats.Skipped, stats.PassRate))
			}
		}
		buf.WriteString("\n")
	}

	// Coverage by Package
	if len(r.Coverage) > 0 {
		buf.WriteString("### Coverage by Package\n\n")
		buf.WriteString("| Package | Coverage | Statements |\n")
		buf.WriteString("|---------|----------|-------------|\n")

		covPackages := make([]string, 0, len(r.Coverage))
		for pkg := range r.Coverage {
			covPackages = append(covPackages, pkg)
		}
		sort.Strings(covPackages)

		for _, pkg := range covPackages {
			info := r.Coverage[pkg]
			coverage := fmt.Sprintf("%.1f%%", info.Coverage)
			if info.IsLowCoverage {
				coverage = "⚠️ " + coverage
			}
			buf.WriteString(fmt.Sprintf("| `%s` | %s | %d/%d |\n",
				pkg, coverage, info.StmtCovered, info.StmtTotal))
		}
		buf.WriteString("\n")
	}

	// Low Coverage Alert
	if len(r.LowCoverageList) > 0 {
		buf.WriteString("### ⚠️ Low Coverage Packages (<80%)\n\n")
		for _, pkg := range r.LowCoverageList {
			if info, ok := r.Coverage[pkg]; ok {
				buf.WriteString(fmt.Sprintf("- `%s`: **%.1f%%** coverage (%d/%d statements)\n",
					pkg, info.Coverage, info.StmtCovered, info.StmtTotal))
			}
		}
		buf.WriteString("\n")
	}

	// Slowest Tests
	if len(r.SlowestTests) > 0 {
		buf.WriteString("### 🐢 Slowest Tests (Top 10)\n\n")
		for i, test := range r.SlowestTests {
			if i >= 10 {
				break
			}
			buf.WriteString(fmt.Sprintf("%d. `%s::%s` — **%.3fs**\n",
				i+1, test.Package, test.Test, test.Elapsed))
		}
		buf.WriteString("\n")
	}

	// Regression Tests Status
	if r.RegressionTests != nil && r.RegressionTests.Total > 0 {
		buf.WriteString("### 🔄 Regression Test Status\n\n")
		buf.WriteString(fmt.Sprintf("| Metric | Count |\n|--------|-------|\n"))
		buf.WriteString(fmt.Sprintf("| Total | %d |\n", r.RegressionTests.Total))
		buf.WriteString(fmt.Sprintf("| ✅ Passed | %d (%.1f%%) |\n", r.RegressionTests.Passed, r.RegressionTests.PassRate))
		buf.WriteString(fmt.Sprintf("| ❌ Failed | %d |\n", r.RegressionTests.Failed))
		buf.WriteString(fmt.Sprintf("| ⏭️ Skipped | %d |\n\n", r.RegressionTests.Skipped))
	}

	return buf.String()
}

// RenderHTML generates an interactive HTML dashboard with Plotly charts
func (r *Report) RenderHTML() string {
	overallCoverage := calculateOverallCoverage(r.Coverage)

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Test Report Dashboard</title>
	<script src="https://cdn.plot.ly/plotly-latest.min.js"></script>
	<style>
		* { margin: 0; padding: 0; box-sizing: border-box; }
		body {
			font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "Roboto", "Oxygen", "Ubuntu", sans-serif;
			background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
			min-height: 100vh;
			padding: 40px 20px;
		}
		.container {
			max-width: 1400px;
			margin: 0 auto;
			background: white;
			border-radius: 12px;
			box-shadow: 0 20px 60px rgba(0,0,0,0.3);
			overflow: hidden;
		}
		.header {
			background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
			color: white;
			padding: 40px;
			text-align: center;
		}
		.header h1 { font-size: 2.5em; margin-bottom: 10px; }
		.header p { font-size: 1.1em; opacity: 0.95; }
		.metrics {
			display: grid;
			grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
			gap: 20px;
			padding: 40px;
			background: #f8f9fa;
			border-bottom: 1px solid #e0e0e0;
		}
		.metric-card {
			background: white;
			padding: 25px;
			border-radius: 8px;
			border-left: 4px solid #667eea;
			box-shadow: 0 2px 8px rgba(0,0,0,0.08);
			text-align: center;
		}
		.metric-value {
			font-size: 2.5em;
			font-weight: bold;
			color: #667eea;
			margin-bottom: 8px;
		}
		.metric-label {
			color: #666;
			font-size: 0.95em;
			text-transform: uppercase;
			letter-spacing: 0.5px;
		}
		.content {
			padding: 40px;
		}
		.chart-section {
			margin-bottom: 50px;
		}
		.chart-section h2 {
			color: #333;
			margin-bottom: 20px;
			padding-bottom: 10px;
			border-bottom: 2px solid #667eea;
		}
		.chart-container {
			background: white;
			border-radius: 8px;
			box-shadow: 0 2px 8px rgba(0,0,0,0.08);
			padding: 20px;
			margin-bottom: 20px;
		}
		table {
			width: 100%%;
			border-collapse: collapse;
			margin: 20px 0;
		}
		th {
			background: #f8f9fa;
			padding: 15px;
			text-align: left;
			font-weight: 600;
			color: #333;
			border-bottom: 2px solid #ddd;
		}
		td {
			padding: 12px 15px;
			border-bottom: 1px solid #eee;
		}
		tr:hover { background: #f8f9fa; }
		.low-coverage { color: #d9534f; font-weight: 600; }
		.footer {
			background: #f8f9fa;
			padding: 20px 40px;
			border-top: 1px solid #e0e0e0;
			font-size: 0.9em;
			color: #666;
		}
		@media (max-width: 768px) {
			.header { padding: 20px; }
			.header h1 { font-size: 1.8em; }
			.metrics { grid-template-columns: 1fr; }
			.content { padding: 20px; }
		}
	</style>
</head>
<body>
	<div class="container">
		<div class="header">
			<h1>📊 Test Report Dashboard</h1>
			<p>Comprehensive Test Coverage & Performance Analysis</p>
		</div>

		<div class="metrics">
			<div class="metric-card">
				<div class="metric-value">%d</div>
				<div class="metric-label">Total Tests</div>
			</div>
			<div class="metric-card">
				<div class="metric-value">%.1f%%</div>
				<div class="metric-label">Pass Rate</div>
			</div>
			<div class="metric-card">
				<div class="metric-value">%.1f%%</div>
				<div class="metric-label">Coverage</div>
			</div>
			<div class="metric-card">
				<div class="metric-value">%.2fs</div>
				<div class="metric-label">Duration</div>
			</div>
		</div>

		<div class="content">
`, r.Tests.Total, r.Tests.PassRate, overallCoverage, r.Tests.TotalTime)

	// Add delta section if available
	if r.Delta != nil {
		html += `			<div class="chart-section">
				<h2>📈 Changes from Previous Run</h2>
				<table>
					<tr>
						<th>Metric</th>
						<th>Change</th>
					</tr>
`
		if r.Delta.PassRateChange > 0 {
			html += fmt.Sprintf(`					<tr>
						<td>Pass Rate</td>
						<td><span style="color: #5cb85c;">+%.1f%% ✅</span></td>
					</tr>
`, r.Delta.PassRateChange)
		} else if r.Delta.PassRateChange < 0 {
			html += fmt.Sprintf(`					<tr>
						<td>Pass Rate</td>
						<td><span style="color: #d9534f;">%.1f%% ❌</span></td>
					</tr>
`, r.Delta.PassRateChange)
		}

		if r.Delta.CoverageChange != 0 {
			sign := "+"
			color := "#5cb85c"
			if r.Delta.CoverageChange < 0 {
				sign = ""
				color = "#d9534f"
			}
			html += fmt.Sprintf(`					<tr>
						<td>Coverage</td>
						<td><span style="color: %s;">%s%.1f%%</span></td>
					</tr>
`, color, sign, r.Delta.CoverageChange)
		}

		html += `				</table>
			</div>
`
	}

	// Charts for test results by package
	html += `			<div class="chart-section">
				<h2>📦 Test Results by Package</h2>
				<div id="packageChart" class="chart-container"></div>
			</div>

			<div class="chart-section">
				<h2>🔍 Package Details</h2>
				<table>
					<tr>
						<th>Package</th>
						<th>Tests</th>
						<th>✅ Passed</th>
						<th>❌ Failed</th>
						<th>⏭️ Skipped</th>
						<th>Pass Rate</th>
					</tr>
`

	for _, pkg := range sortedPackages(r.Tests.ByPackage) {
		stats := r.Tests.ByPackage[pkg]
		passRateStyle := ""
		if stats.PassRate < 80 {
			passRateStyle = ` class="low-coverage"`
		}
		html += fmt.Sprintf(`					<tr>
						<td><code>%s</code></td>
						<td>%d</td>
						<td>%d</td>
						<td>%d</td>
						<td>%d</td>
						<td%s>%.1f%%</td>
					</tr>
`, pkg, stats.Total, stats.Passed, stats.Failed, stats.Skipped, passRateStyle, stats.PassRate)
	}

	html += `				</table>
			</div>
		</div>

		<div class="footer">
			<p>Generated at ` + r.Timestamp + `</p>
		</div>
	</div>

	<script>
		// Package test results chart
		var packageNames = [` + formatPackageList(r.Tests.ByPackage) + `];
		var passed = [` + formatPackageMetric(r.Tests.ByPackage, "Passed") + `];
		var failed = [` + formatPackageMetric(r.Tests.ByPackage, "Failed") + `];
		var skipped = [` + formatPackageMetric(r.Tests.ByPackage, "Skipped") + `];

		var trace1 = { name: '✅ Passed', x: packageNames, y: passed, type: 'bar', marker: {color: '#5cb85c'} };
		var trace2 = { name: '❌ Failed', x: packageNames, y: failed, type: 'bar', marker: {color: '#d9534f'} };
		var trace3 = { name: '⏭️ Skipped', x: packageNames, y: skipped, type: 'bar', marker: {color: '#f0ad4e'} };

		var data = [trace1, trace2, trace3];
		var layout = { barmode: 'stack', height: 400, hovermode: 'x unified' };

		Plotly.newPlot('packageChart', data, layout);
	</script>
</body>
</html>`

	return html
}

// Helper functions for HTML generation
func formatPackageList(packages map[string]*PackageStats) string {
	var names []string
	for name := range packages {
		names = append(names, name)
	}
	sort.Strings(names)

	var result []string
	for _, name := range names {
		result = append(result, fmt.Sprintf(`'%s'`, name))
	}
	return strings.Join(result, ", ")
}

func formatPackageMetric(packages map[string]*PackageStats, metric string) string {
	var names []string
	for name := range packages {
		names = append(names, name)
	}
	sort.Strings(names)

	var result []string
	for _, name := range names {
		pkg := packages[name]
		var value int
		switch metric {
		case "Passed":
			value = pkg.Passed
		case "Failed":
			value = pkg.Failed
		case "Skipped":
			value = pkg.Skipped
		}
		result = append(result, fmt.Sprintf(`%d`, value))
	}
	return strings.Join(result, ", ")
}
