package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/docToolchain/Bausteinsicht/internal/graph"
	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/spf13/cobra"
)

func newGraphCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "graph",
		Short: "Analyze relationship graph for cycles and dependencies",
		Long:  "Analyzes the relationship graph to detect cycles, calculate centrality metrics, and identify dependency patterns.",
		RunE:  runGraph,
	}

	cmd.Flags().String("output", "", "Output file for analysis report (default: stdout)")
	cmd.Flags().Bool("cycles-only", false, "Show only detected cycles")
	cmd.Flags().Bool("centrality", false, "Show centrality metrics for each element")

	return cmd
}

func runGraph(cmd *cobra.Command, _ []string) error {
	modelPath, _ := cmd.Flags().GetString("model")
	format, _ := cmd.Flags().GetString("format")
	outputPath, _ := cmd.Flags().GetString("output")
	cyclesOnly, _ := cmd.Flags().GetBool("cycles-only")
	showCentrality, _ := cmd.Flags().GetBool("centrality")

	if modelPath == "" {
		return exitWithCode(fmt.Errorf("--model flag is required"), 2)
	}

	m, err := model.Load(modelPath)
	if err != nil {
		return exitWithCode(fmt.Errorf("loading model: %w", err), 2)
	}

	// Analyze graph
	analyzer := graph.NewAnalyzer(m)
	result := analyzer.Analyze()

	// Format output
	var output string

	if format == "json" {
		data, _ := json.MarshalIndent(result, "", "  ")
		output = string(data)
	} else {
		output = formatGraphReport(result, cyclesOnly, showCentrality)
	}

	// Write output
	if outputPath != "" {
		if err := os.WriteFile(outputPath, []byte(output), 0644); err != nil {
			return exitWithCode(fmt.Errorf("writing output: %w", err), 2)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Graph analysis written to %s\n", outputPath)
	} else {
		_, _ = fmt.Fprint(cmd.OutOrStdout(), output)
	}

	return nil
}

func formatGraphReport(result *graph.GraphAnalysis, cyclesOnly, showCentrality bool) string {
	var sb strings.Builder

	sb.WriteString("Relationship Graph Analysis\n")
	sb.WriteString("===========================\n\n")

	// Summary
	sb.WriteString("Summary\n")
	sb.WriteString("-------\n")
	fmt.Fprintf(&sb, "Elements: %d\n", result.ElementCount)
	fmt.Fprintf(&sb, "Relationships: %d\n", result.RelationshipCount)
	fmt.Fprintf(&sb, "Max Dependency Depth: %d\n", result.MaxDepth)
	sb.WriteString("Graph Type: ")
	if result.IDAGValid {
		sb.WriteString("DAG (acyclic)")
	} else {
		sb.WriteString("Cyclic (contains cycles)")
	}
	sb.WriteString("\n\n")

	// Cycles
	if len(result.Cycles) > 0 {
		fmt.Fprintf(&sb, "Cycles Found: %d\n", len(result.Cycles))
		sb.WriteString("--------\n")
		for idx, cycle := range result.Cycles {
			sb.WriteString(fmt.Sprintf("Cycle %d (length %d): %s\n", idx+1, cycle.Length, strings.Join(cycle.Elements, " → ")))
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString("No cycles detected (valid DAG)\n\n")
	}

	if cyclesOnly {
		return sb.String()
	}

	// Strongly connected components
	if len(result.Components) > 0 {
		cycleCount := 0
		for _, comp := range result.Components {
			if comp.IsCycle {
				cycleCount++
			}
		}
		sb.WriteString(fmt.Sprintf("Strongly Connected Components: %d\n", len(result.Components)))
		if cycleCount > 0 {
			sb.WriteString(fmt.Sprintf("  (includes %d cycle(s))\n", cycleCount))
		}
		sb.WriteString("--------\n")
		for _, comp := range result.Components {
			if comp.IsCycle {
				sb.WriteString(fmt.Sprintf("Component %d (CYCLE): %v\n", comp.ID+1, comp.Elements))
			}
		}
		sb.WriteString("\n")
	}

	// Centrality metrics
	if showCentrality && len(result.Centrality) > 0 {
		sb.WriteString("Centrality Metrics\n")
		sb.WriteString("------------------\n")
		sb.WriteString("Element                | In-Degree | Out-Degree | Betweenness | Closeness\n")
		sb.WriteString("---------------------- | --------- | ---------- | ----------- | ---------\n")

		// Sort by out-degree descending
		sorted := make([]graph.Centrality, len(result.Centrality))
		copy(sorted, result.Centrality)
		sort.Slice(sorted, func(i, j int) bool {
			if sorted[i].OutDegree != sorted[j].OutDegree {
				return sorted[i].OutDegree > sorted[j].OutDegree
			}
			return sorted[i].ID < sorted[j].ID
		})

		for _, c := range sorted {
			elemName := c.ID
			if len(elemName) > 22 {
				elemName = elemName[:19] + "..."
			}
			sb.WriteString(fmt.Sprintf("%-22s | %9d | %10d | %11.2f | %9.2f\n",
				elemName, c.InDegree, c.OutDegree, c.Betweenness, c.Closeness))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
