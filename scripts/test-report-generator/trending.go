package main

import (
	"fmt"
	"sort"
)

// PerformanceRegression detects if tests have become significantly slower
func (r *Report) DetectPerformanceRegression(slowThreshold float64) []SlowTest {
	var regressions []SlowTest

	// Sort all slowest tests and flag those that exceed threshold vs average
	if r.Tests.Total == 0 {
		return regressions
	}

	avgTime := r.Tests.TotalTime / float64(r.Tests.Total)
	threshold := avgTime * slowThreshold

	for _, test := range r.SlowestTests {
		if test.Elapsed > threshold {
			regressions = append(regressions, test)
		}
	}

	return regressions
}

// TrendData holds historical trend information across multiple reports
type TrendData struct {
	Timestamp     string    `json:"timestamp"`
	PassRate      float64   `json:"pass_rate"`
	Coverage      float64   `json:"coverage"`
	AvgTestTime   float64   `json:"avg_test_time"`
	FailureCount  int       `json:"failure_count"`
	TotalTests    int       `json:"total_tests"`
	RegressionPct float64   `json:"regression_pct"`
}

// ExtractTrend extracts trend data from a report
func (r *Report) ExtractTrend() TrendData {
	avgTime := 0.0
	if r.Tests.Total > 0 {
		avgTime = r.Tests.TotalTime / float64(r.Tests.Total)
	}

	overallCoverage := calculateOverallCoverage(r.Coverage)
	regressionPct := 0.0
	if r.RegressionTests != nil && r.RegressionTests.Total > 0 {
		regressionPct = r.RegressionTests.PassRate
	}

	return TrendData{
		Timestamp:     r.Timestamp,
		PassRate:      r.Tests.PassRate,
		Coverage:      overallCoverage,
		AvgTestTime:   avgTime,
		FailureCount:  r.Tests.Failed,
		TotalTests:    r.Tests.Total,
		RegressionPct: regressionPct,
	}
}

// GenerateTrendMarkdown creates a Markdown section with trend analysis
func GenerateTrendMarkdown(trends []TrendData) string {
	if len(trends) == 0 {
		return ""
	}

	// Sort by timestamp (oldest first)
	sort.Slice(trends, func(i, j int) bool {
		return trends[i].Timestamp < trends[j].Timestamp
	})

	markdown := "### 📈 Trends (Last 10 Runs)\n\n"
	markdown += "| Timestamp | Pass Rate | Coverage | Avg Time | Failures |\n"
	markdown += "|-----------|-----------|----------|----------|----------|\n"

	for _, t := range trends {
		trend := "→"
		if len(trends) > 1 {
			idx := -1
			for i, tr := range trends {
				if tr.Timestamp == t.Timestamp {
					idx = i
					break
				}
			}
			if idx > 0 {
				prev := trends[idx-1]
				if t.PassRate > prev.PassRate {
					trend = "📈"
				} else if t.PassRate < prev.PassRate {
					trend = "📉"
				}
			}
		}
		markdown += fmt.Sprintf("| %s | %.1f%% %s | %.1f%% | %.3fs | %d |\n",
			t.Timestamp, t.PassRate, trend, t.Coverage, t.AvgTestTime, t.FailureCount)
	}

	return markdown + "\n"
}

// AnalyzeTrendBreakpoints identifies significant changes in trend data
type TrendBreakpoint struct {
	Type      string  // "regression" or "improvement"
	Metric    string  // "pass_rate", "coverage", "performance"
	Change    float64
	Timestamp string
}

// FindTrendBreakpoints identifies points where metrics changed significantly
func FindTrendBreakpoints(trends []TrendData, threshold float64) []TrendBreakpoint {
	var breakpoints []TrendBreakpoint

	if len(trends) < 2 {
		return breakpoints
	}

	// Sort by timestamp
	sort.Slice(trends, func(i, j int) bool {
		return trends[i].Timestamp < trends[j].Timestamp
	})

	for i := 1; i < len(trends); i++ {
		curr := trends[i]
		prev := trends[i-1]

		// Check pass rate change
		passRateChange := curr.PassRate - prev.PassRate
		if absFloat(passRateChange) >= threshold {
			breakType := "improvement"
			if passRateChange < 0 {
				breakType = "regression"
			}
			breakpoints = append(breakpoints, TrendBreakpoint{
				Type:      breakType,
				Metric:    "pass_rate",
				Change:    passRateChange,
				Timestamp: curr.Timestamp,
			})
		}

		// Check coverage change
		coverageChange := curr.Coverage - prev.Coverage
		if absFloat(coverageChange) >= threshold {
			breakType := "improvement"
			if coverageChange < 0 {
				breakType = "regression"
			}
			breakpoints = append(breakpoints, TrendBreakpoint{
				Type:      breakType,
				Metric:    "coverage",
				Change:    coverageChange,
				Timestamp: curr.Timestamp,
			})
		}

		// Check performance change (% slower)
		if prev.AvgTestTime > 0 {
			performanceChange := (curr.AvgTestTime - prev.AvgTestTime) / prev.AvgTestTime * 100
			if absFloat(performanceChange) >= 10 { // 10% threshold
				breakType := "regression"
				if performanceChange < 0 {
					breakType = "improvement"
				}
				breakpoints = append(breakpoints, TrendBreakpoint{
					Type:      breakType,
					Metric:    "performance",
					Change:    performanceChange,
					Timestamp: curr.Timestamp,
				})
			}
		}
	}

	return breakpoints
}

func absFloat(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

// GenerateAdvancedHTML creates an HTML dashboard with trend charts
func (r *Report) RenderAdvancedHTML(trends []TrendData) string {
	// Base HTML with trend section
	html := r.RenderHTML()

	// Insert trend section before closing body tag
	trendSection := fmt.Sprintf(`
			<div class="chart-section">
				<h2>📊 Historical Trends</h2>
				<div id="trendChart" class="chart-container"></div>
			</div>

			<script>
				// Prepare trend data
				var timestamps = [%s];
				var passRates = [%s];
				var coverages = [%s];

				// Pass rate trend
				var tracePassRate = {
					name: 'Pass Rate',
					x: timestamps,
					y: passRates,
					type: 'scatter',
					mode: 'lines+markers',
					line: {color: '#667eea'},
					fill: 'tozeroy'
				};

				// Coverage trend
				var traceCoverage = {
					name: 'Coverage',
					x: timestamps,
					y: coverages,
					type: 'scatter',
					mode: 'lines+markers',
					line: {color: '#764ba2'},
					fill: 'tozeroy',
					yaxis: 'y2'
				};

				var data = [tracePassRate, traceCoverage];
				var layout = {
					height: 400,
					hovermode: 'x unified',
					yaxis: { title: 'Pass Rate (%%)' },
					yaxis2: { title: 'Coverage (%%)', overlaying: 'y', side: 'right' }
				};

				Plotly.newPlot('trendChart', data, layout);
			</script>
`, formatTrendTimestamps(trends), formatTrendMetric(trends, "PassRate"), formatTrendMetric(trends, "Coverage"))

	// Replace body close tag
	html = html[:len(html)-7] + trendSection + html[len(html)-7:]
	return html
}

func formatTrendTimestamps(trends []TrendData) string {
	var timestamps []string
	for _, t := range trends {
		timestamps = append(timestamps, fmt.Sprintf(`'%s'`, t.Timestamp))
	}
	return joinStrings(timestamps, ", ")
}

func formatTrendMetric(trends []TrendData, metric string) string {
	var values []string
	for _, t := range trends {
		var val float64
		switch metric {
		case "PassRate":
			val = t.PassRate
		case "Coverage":
			val = t.Coverage
		}
		values = append(values, fmt.Sprintf("%.1f", val))
	}
	return joinStrings(values, ", ")
}

func joinStrings(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
