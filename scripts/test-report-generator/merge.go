package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// OSReport wraps a Report with OS metadata for merged reports
type OSReport struct {
	OS     string
	Icon   string
	Label  string
	Report *Report
}

// runMerge loads multiple OS reports and renders a merged HTML with tabs
func runMerge(linuxPath, windowsPath, macosPath string) error {
	var reports []OSReport

	// Load Linux report
	if linuxPath != "" {
		linux, err := loadReportFromJSON(linuxPath)
		if err != nil {
			return fmt.Errorf("load linux report: %w", err)
		}
		reports = append(reports, OSReport{
			OS:     "linux",
			Icon:   "🐧",
			Label:  "Linux",
			Report: linux,
		})
	}

	// Load Windows report
	if windowsPath != "" {
		windows, err := loadReportFromJSON(windowsPath)
		if err != nil {
			return fmt.Errorf("load windows report: %w", err)
		}
		reports = append(reports, OSReport{
			OS:     "windows",
			Icon:   "🪟",
			Label:  "Windows",
			Report: windows,
		})
	}

	// Load macOS report
	if macosPath != "" {
		macos, err := loadReportFromJSON(macosPath)
		if err != nil {
			return fmt.Errorf("load macos report: %w", err)
		}
		reports = append(reports, OSReport{
			OS:     "macos",
			Icon:   "🍎",
			Label:  "macOS",
			Report: macos,
		})
	}

	if len(reports) == 0 {
		return fmt.Errorf("no reports loaded")
	}

	html := renderMergedHTML(reports)
	fmt.Print(html)
	return nil
}

// loadReportFromJSON loads a Report from a JSON file
func loadReportFromJSON(path string) (*Report, error) {
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

// renderOverviewTab generates a comparison overview of all OS reports
func renderOverviewTab(reports []OSReport) string {
	var buf strings.Builder

	buf.WriteString(`		<div class="chart-section">
			<h2>🔄 Platform Comparison</h2>
			<table class="comparison-table">
				<tr>
					<th>Platform</th>
					<th>Total Tests</th>
					<th>Passed</th>
					<th>Failed</th>
					<th>Skipped</th>
					<th>Pass Rate</th>
					<th>Duration</th>
					<th>Coverage</th>
				</tr>
`)

	for _, osReport := range reports {
		overallCov := calculateOverallCoverage(osReport.Report.Coverage)
		passed := osReport.Report.Tests.Passed
		failed := osReport.Report.Tests.Failed
		passRate := osReport.Report.Tests.PassRate
		duration := osReport.Report.Tests.TotalTime

		failedStyle := ""
		if failed > 0 {
			failedStyle = ` class="highlight-fail"`
		}

		covStyle := ""
		if overallCov < 80 {
			covStyle = ` class="highlight-warn"`
		}

		buf.WriteString(fmt.Sprintf(`				<tr>
					<td>%s %s</td>
					<td>%d</td>
					<td>%d</td>
					<td%s>%d</td>
					<td>%d</td>
					<td>%.1f%%</td>
					<td>%.2fs</td>
					<td%s>%.1f%%</td>
				</tr>
`, osReport.Icon, osReport.Label, osReport.Report.Tests.Total, passed, failedStyle, failed, osReport.Report.Tests.Skipped, passRate, duration, covStyle, overallCov))
	}

	buf.WriteString(`			</table>
		</div>
`)

	// Package differences
	packageDiffs := getPackageDifferences(reports)
	if len(packageDiffs) > 0 {
		buf.WriteString(`		<div class="chart-section">
			<h2>📦 Package Differences</h2>
			<p style="font-size: 0.9em; color: #666; margin-bottom: 15px;">Showing packages with test count or failure differences across platforms</p>
			<table class="comparison-table">
				<tr>
					<th>Package</th>
`)
		for _, osReport := range reports {
			buf.WriteString(fmt.Sprintf(`					<th>%s %s</th>
`, osReport.Icon, osReport.Label))
		}
		buf.WriteString(`				</tr>
`)

		for _, pkg := range sortedPackages(packageDiffs) {
			buf.WriteString(`				<tr>
`)
			buf.WriteString(fmt.Sprintf(`					<td><code>%s</code></td>
`, pkg))

			for _, osReport := range reports {
				if stats, ok := osReport.Report.Tests.ByPackage[pkg]; ok {
					style := ""
					if stats.Failed > 0 {
						style = ` class="highlight-fail"`
					}
					buf.WriteString(fmt.Sprintf(`					<td%s>%d / %d (✅ %d)</td>
`, style, stats.Failed, stats.Total, stats.Passed))
				} else {
					buf.WriteString(`					<td style="color: #999;">N/A</td>
`)
				}
			}
			buf.WriteString(`				</tr>
`)
		}

		buf.WriteString(`			</table>
		</div>
`)
	}

	return buf.String()
}

// getPackageDifferences returns a map of packages that have differences across OS reports
func getPackageDifferences(reports []OSReport) map[string]*PackageStats {
	diffs := make(map[string]*PackageStats)

	// Collect all packages
	allPackages := make(map[string]bool)
	for _, osReport := range reports {
		for pkg := range osReport.Report.Tests.ByPackage {
			allPackages[pkg] = true
		}
	}

	// Find packages with differences
	for pkg := range allPackages {
		var totalCounts []int
		var failCounts []int

		for _, osReport := range reports {
			if stats, ok := osReport.Report.Tests.ByPackage[pkg]; ok {
				totalCounts = append(totalCounts, stats.Total)
				failCounts = append(failCounts, stats.Failed)
			} else {
				totalCounts = append(totalCounts, 0)
				failCounts = append(failCounts, 0)
			}
		}

		// Check if there are differences
		hasDiff := false
		if len(totalCounts) > 0 {
			first := totalCounts[0]
			for _, count := range totalCounts {
				if count != first {
					hasDiff = true
					break
				}
			}
		}

		// Check if any platform has failures
		hasFailures := false
		for _, count := range failCounts {
			if count > 0 {
				hasFailures = true
				break
			}
		}

		if hasDiff || hasFailures {
			diffs[pkg] = &PackageStats{}
		}
	}

	return diffs
}

// renderMergedHTML generates an HTML report with tabs for multiple OS reports
func renderMergedHTML(reports []OSReport) string {
	if len(reports) == 0 {
		return ""
	}

	// Use first report for timestamp and overall metrics
	first := reports[0].Report

	html := `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Multi-OS Test Report Dashboard</title>
	<script src="https://cdn.plot.ly/plotly-latest.min.js"></script>
	<style>
		* { margin: 0; padding: 0; box-sizing: border-box; }
		body {
			font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "Roboto", "Oxygen", "Ubuntu", sans-serif;
			background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
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
			background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
			color: white;
			padding: 40px;
			text-align: center;
		}
		.header h1 { font-size: 2.5em; margin-bottom: 10px; }
		.header p { font-size: 1.1em; opacity: 0.95; }

		/* Tab Navigation */
		.tab-nav {
			background: #f8f9fa;
			border-bottom: 2px solid #e0e0e0;
			padding: 0;
			display: flex;
			flex-wrap: wrap;
			gap: 0;
		}
		.tab-radio { display: none; }
		.tab-label {
			padding: 15px 25px;
			cursor: pointer;
			background: #f8f9fa;
			border: none;
			font-size: 1em;
			font-weight: 500;
			color: #666;
			transition: all 0.3s ease;
			border-bottom: 3px solid transparent;
			user-select: none;
		}
		.tab-label:hover { background: #e9ecef; }
		.tab-radio:checked + .tab-label {
			color: #667eea;
			background: white;
			border-bottom-color: #667eea;
		}

		.tab-content {
			display: none;
			padding: 40px;
		}
		#tab-overview:checked ~ .tab-content-overview,
		#tab-linux:checked ~ .tab-content-linux,
		#tab-windows:checked ~ .tab-content-windows,
		#tab-macos:checked ~ .tab-content-macos {
			display: block;
		}

		.comparison-table {
			width: 100%;
			border-collapse: collapse;
		}
		.comparison-table th {
			background: #f8f9fa;
			padding: 15px;
			text-align: left;
			font-weight: 600;
			color: #333;
			border-bottom: 2px solid #ddd;
		}
		.comparison-table td {
			padding: 12px 15px;
			border-bottom: 1px solid #eee;
		}
		.comparison-table tr:hover {
			background: #f9f9f9;
		}
		.highlight-fail {
			color: #d9534f;
			font-weight: 600;
		}
		.highlight-warn {
			color: #fd7e14;
			font-weight: 600;
		}

		.metrics {
			display: grid;
			grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
			gap: 20px;
			margin-bottom: 30px;
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
			width: 100%;
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

		/* Coverage Viewer */
		.coverage-section { margin-top: 50px; }
		.coverage-section h2 { color: #333; margin-bottom: 20px; padding-bottom: 10px; border-bottom: 2px solid #667eea; }
		.file-accordion { border: 1px solid #e0e0e0; border-radius: 6px; margin-bottom: 10px; overflow: hidden; }
		.file-accordion details { border: none; }
		.file-header {
			padding: 12px 20px; cursor: pointer; background: #f8f9fa; border: none; width: 100%;
			text-align: left; font-size: 0.95em; user-select: none; display: flex; justify-content: space-between; align-items: center;
		}
		.file-header:hover { background: #e9ecef; }
		.file-accordion details[open] > summary { background: #e8eaf6; }
		.file-name { font-family: "SFMono-Regular", Consolas, monospace; font-weight: 600; color: #333; }
		.file-coverage-badge {
			font-size: 0.85em; padding: 3px 10px; border-radius: 12px; font-weight: 600; color: white;
		}
		.badge-high { background: #28a745; }
		.badge-mid { background: #fd7e14; }
		.badge-low { background: #dc3545; }
		.code-viewer { font-family: "SFMono-Regular", Consolas, monospace; font-size: 0.82em; overflow-x: auto; }
		.code-table { width: 100%; border-collapse: collapse; }
		.line-num {
			width: 50px; min-width: 50px; text-align: right; padding: 1px 12px 1px 6px;
			color: #999; background: #f8f9fa; border-right: 1px solid #e0e0e0; user-select: none; vertical-align: top;
		}
		.line-code { padding: 1px 6px 1px 12px; white-space: pre; }
		.line-covered { background: #d4edda; }
		.line-uncovered { background: #f8d7da; }

		@media (max-width: 768px) {
			.header { padding: 20px; }
			.header h1 { font-size: 1.8em; }
			.tab-label { padding: 10px 15px; font-size: 0.9em; }
			.tab-content { padding: 20px; }
			.metrics { grid-template-columns: 1fr; }
		}
	</style>
</head>
<body>
	<div class="container">
		<div class="header">
			<h1>📊 Multi-OS Test Report Dashboard</h1>
			<p>Cross-Platform Test Coverage & Performance Analysis</p>
		</div>

		<div class="tab-nav">
			<label for="tab-overview" class="tab-label">📊 Overview</label>
`

	// Render OS tab labels
	for _, report := range reports {
		html += `			<label for="tab-` + report.OS + `" class="tab-label">` + report.Icon + ` ` + report.Label + `</label>
`
	}

	html += `		</div>
`

	// Hidden input elements for tab state management
	html += `		<input type="radio" id="tab-overview" name="tab-group" class="tab-radio" checked>
`

	// OS inputs
	for _, osReport := range reports {
		html += `		<input type="radio" id="tab-` + osReport.OS + `" name="tab-group" class="tab-radio">
`
	}

	// Render Overview tab content
	html += `		<div class="tab-content tab-content-overview">
` + renderOverviewTab(reports) + `		</div>
`

	// Render OS tab content
	for _, osReport := range reports {
		r := osReport.Report

		// Calculate metrics for this report
		overallCoverage := calculateOverallCoverage(r.Coverage)
		totalTests := fmt.Sprintf("%d", r.Tests.Total)
		passRate := fmt.Sprintf("%.1f", r.Tests.PassRate)
		coverage := fmt.Sprintf("%.1f", overallCoverage)
		duration := fmt.Sprintf("%.2f", r.Tests.TotalTime)

		html += `		<div class="tab-content tab-content-` + osReport.OS + `">
			<div class="metrics">
				<div class="metric-card">
					<div class="metric-value">` + totalTests + `</div>
					<div class="metric-label">Total Tests</div>
				</div>
				<div class="metric-card">
					<div class="metric-value">` + passRate + `%</div>
					<div class="metric-label">Pass Rate</div>
				</div>
				<div class="metric-card">
					<div class="metric-value">` + coverage + `%</div>
					<div class="metric-label">Coverage</div>
				</div>
				<div class="metric-card">
					<div class="metric-value">` + duration + `s</div>
					<div class="metric-label">Duration</div>
				</div>
			</div>
`

		html += r.renderTabContent(osReport.OS)

		html += `		</div>
`
	}

	html += `
		<div class="footer">
			<p>Generated at ` + first.Timestamp + `</p>
		</div>
	</div>

	<script>
		function initPlotlyChart(tabID) {
			// Will be defined in each tab's content
		}

		// Init first tab charts immediately
		document.addEventListener('DOMContentLoaded', function() {
			initPlotlyChart('` + reports[0].OS + `');
		});

		// Re-init charts on tab change
		document.querySelectorAll('.tab-radio').forEach(radio => {
			radio.addEventListener('change', function() {
				if (this.checked) {
					setTimeout(() => initPlotlyChart(this.id.replace('tab-', '')), 100);
				}
			});
		});
	</script>
</body>
</html>`

	return html
}
