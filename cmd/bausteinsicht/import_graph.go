package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/docToolchain/Bausteinsicht/internal/codegraph"
	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/spf13/cobra"
)

func newImportGraphCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import-graph <codegraph-file>",
		Short: "Import a code-derived package graph as the model's as-is snapshot",
		Long: `Projects a language-neutral codegraph JSON document (emitted by a code-side
extractor: jdeps, an ArchUnit dumper, go list, Roslyn, ...) onto model
elements via their codeMapping, and writes the result as the model's asIs
snapshot. Run 'bausteinsicht diff' afterwards to compare it against toBe.

Packages with no covering codeMapping are reported, not silently dropped.
See ADR-013 for the design and https://github.com/docToolchain/bausteinsicht-archunit
for a reference extractor/adapter.

Exit codes:
  0   import successful
  1   model has no element with a codeMapping, or asIs write failed
  2   codegraph file not found, unreadable, or malformed`,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runImportGraph,
	}

	cmd.Flags().String("format", "text", "Output format: text or json")

	return cmd
}

// importGraphSummary is the --format json report shape.
type importGraphSummary struct {
	Language            string   `json:"language"`
	NodesTotal          int      `json:"nodesTotal"`
	EdgesTotal          int      `json:"edgesTotal"`
	ElementsMapped      int      `json:"elementsMapped"`
	RelationshipsMapped int      `json:"relationshipsMapped"`
	UnmappedPackages    []string `json:"unmappedPackages"`
	ModelPath           string   `json:"modelPath"`
}

func runImportGraph(cmd *cobra.Command, args []string) error {
	codeGraphPath := args[0]
	format, _ := cmd.Flags().GetString("format")
	modelPath, _ := cmd.Flags().GetString("model")

	if format != "text" && format != "json" {
		return exitWithCode(fmt.Errorf("invalid format: %s (expected text or json)", format), 2)
	}

	if err := validatePathContainment(codeGraphPath); err != nil {
		return exitWithCode(fmt.Errorf("codegraph file: %w", err), 2)
	}

	if modelPath == "" {
		detected, err := model.AutoDetect(".")
		if err != nil {
			return exitWithCode(fmt.Errorf("auto-detecting model: %w", err), 2)
		}
		modelPath = detected
	}
	if err := validatePathContainment(modelPath); err != nil {
		return exitWithCode(fmt.Errorf("--model: %w", err), 2)
	}

	cg, err := codegraph.Load(codeGraphPath)
	if err != nil {
		return exitWithCode(err, 2)
	}

	m, err := model.Load(modelPath)
	if err != nil {
		return exitWithCode(fmt.Errorf("loading model: %w", err), 2)
	}

	hasMapping, err := codegraph.HasAnyCodeMapping(m)
	if err != nil {
		return exitWithCode(err, 1)
	}
	if !hasMapping {
		return exitWithCode(fmt.Errorf("model contains no element with a codeMapping; nothing to import"), 1)
	}

	result, err := codegraph.Project(cg, m)
	if err != nil {
		return exitWithCode(fmt.Errorf("projecting codegraph onto model: %w", err), 1)
	}

	m.AsIs = result.Snapshot
	if err := model.Save(modelPath, m); err != nil {
		return exitWithCode(fmt.Errorf("saving model: %w", err), 1)
	}

	summary := importGraphSummary{
		Language:            cg.Language,
		NodesTotal:          len(cg.Nodes),
		EdgesTotal:          len(cg.Edges),
		ElementsMapped:      len(result.Snapshot.Elements),
		RelationshipsMapped: len(result.Snapshot.Relationships),
		UnmappedPackages:    result.UnmappedPackages,
		ModelPath:           modelPath,
	}

	return printImportGraphSummary(cmd, format, summary)
}

func printImportGraphSummary(cmd *cobra.Command, format string, summary importGraphSummary) error {
	if format == "json" {
		data, err := json.MarshalIndent(summary, "", "  ")
		if err != nil {
			return exitWithCode(fmt.Errorf("marshaling JSON: %w", err), 1)
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return err
	}

	out := cmd.OutOrStdout()
	if _, err := fmt.Fprintf(out, "Imported code graph (%s): %d nodes, %d edges\n",
		summary.Language, summary.NodesTotal, summary.EdgesTotal); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "Mapped: %d elements, %d relationships\n",
		summary.ElementsMapped, summary.RelationshipsMapped); err != nil {
		return err
	}
	if len(summary.UnmappedPackages) > 0 {
		if _, err := fmt.Fprintf(out, "Unmapped packages (%d): %s\n",
			len(summary.UnmappedPackages), strings.Join(summary.UnmappedPackages, ", ")); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(out, "asIs snapshot written to %s\n", summary.ModelPath)
	return err
}
