package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/docToolchain/Bausteinsicht/internal/health"
	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/spf13/cobra"
)

func newHealthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "health",
		Short: "Assess architecture health score",
		Long:  "Computes a comprehensive architecture health score across multiple dimensions including completeness, conformance, and complexity.",
		RunE:  runHealth,
	}

	cmd.Flags().String("output", "", "Output file for health report (default: stdout)")
	cmd.Flags().Bool("summary", false, "Show only the overall score and grade")

	return cmd
}

func runHealth(cmd *cobra.Command, _ []string) error {
	modelPath, _ := cmd.Flags().GetString("model")
	format, _ := cmd.Flags().GetString("format")
	outputPath, _ := cmd.Flags().GetString("output")
	summaryOnly, _ := cmd.Flags().GetBool("summary")

	if modelPath == "" {
		return exitWithCode(fmt.Errorf("--model flag is required"), 2)
	}

	m, err := model.Load(modelPath)
	if err != nil {
		return exitWithCode(fmt.Errorf("loading model: %w", err), 2)
	}

	// Compute health score
	analyzer := health.NewAnalyzer(m)
	score := analyzer.Analyze()

	// Format output
	var output string

	if format == "json" {
		if summaryOnly {
			summary := map[string]interface{}{
				"overall":   score.Overall,
				"grade":     score.Grade,
				"summary":   score.Summary,
				"timestamp": score.Timestamp,
			}
			data, _ := json.MarshalIndent(summary, "", "  ")
			output = string(data)
		} else {
			data, _ := json.MarshalIndent(score, "", "  ")
			output = string(data)
		}
	} else {
		output = formatHealthReport(score, summaryOnly)
	}

	// Write output
	if outputPath != "" {
		if err := os.WriteFile(outputPath, []byte(output), 0644); err != nil {
			return exitWithCode(fmt.Errorf("writing output: %w", err), 2)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Health report written to %s\n", outputPath)
	} else {
		fmt.Fprint(cmd.OutOrStdout(), output)
	}

	return nil
}

func formatHealthReport(score *health.HealthScore, summaryOnly bool) string {
	var sb strings.Builder

	// Header
	sb.WriteString(fmt.Sprintf("Architecture Health Report\n"))
	sb.WriteString(fmt.Sprintf("==========================\n\n"))

	// Overall score
	sb.WriteString(fmt.Sprintf("Overall Score: %.1f/100 [%s]\n", score.Overall, score.Grade))
	sb.WriteString(fmt.Sprintf("Summary: %s\n", score.Summary))
	sb.WriteString(fmt.Sprintf("Timestamp: %s\n\n", score.Timestamp))

	if summaryOnly {
		return sb.String()
	}

	// Model stats
	sb.WriteString(fmt.Sprintf("Model Statistics\n"))
	sb.WriteString(fmt.Sprintf("----------------\n"))
	sb.WriteString(fmt.Sprintf("Elements: %d\n", score.ElementCnt))
	sb.WriteString(fmt.Sprintf("Relationships: %d\n", score.RelCnt))
	sb.WriteString(fmt.Sprintf("Views: %d\n\n", score.ViewCnt))

	// Category scores
	sb.WriteString(fmt.Sprintf("Category Scores\n"))
	sb.WriteString(fmt.Sprintf("---------------\n"))
	for _, cat := range score.Categories {
		sb.WriteString(fmt.Sprintf("%s: %.1f/100 (weight: %.0f%%)\n", cat.Category, cat.Score, cat.Weight*100))
		if cat.Details != "" {
			sb.WriteString(fmt.Sprintf("  Details: %s\n", cat.Details))
		}
	}

	// Findings
	if len(score.Categories) > 0 {
		sb.WriteString(fmt.Sprintf("\nFindings\n"))
		sb.WriteString(fmt.Sprintf("--------\n"))

		for _, cat := range score.Categories {
			if len(cat.Findings) > 0 {
				sb.WriteString(fmt.Sprintf("\n%s (%d findings):\n", cat.Category, len(cat.Findings)))
				for _, f := range cat.Findings {
					sb.WriteString(fmt.Sprintf("  [%s] %s\n", f.Severity, f.Title))
					sb.WriteString(fmt.Sprintf("         %s\n", f.Message))
				}
			}
		}
	}

	return sb.String()
}
