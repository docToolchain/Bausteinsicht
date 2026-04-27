package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/docToolchain/Bausteinsicht/internal/diagram"
	"github.com/docToolchain/Bausteinsicht/internal/export"
	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/spf13/cobra"
)

func newExportDiagramCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export-diagram",
		Short: "Export views as PlantUML C4 or Mermaid C4 diagrams",
		Long:  "Exports architecture views as text-based C4 diagrams (PlantUML or Mermaid).",
		RunE:  runExportDiagram,
	}

	cmd.Flags().String("view", "", "Export only this view (by key)")
	cmd.Flags().String("diagram-format", "plantuml", "Diagram format: plantuml or mermaid")
	cmd.Flags().String("output", "", "Output directory (default: stdout)")

	return cmd
}

func runExportDiagram(cmd *cobra.Command, _ []string) error {
	modelPath, _ := cmd.Flags().GetString("model")
	viewKey, _ := cmd.Flags().GetString("view")
	diagramFormat, _ := cmd.Flags().GetString("diagram-format")
	outputDir, _ := cmd.Flags().GetString("output")

	if modelPath == "" {
		detected, err := model.AutoDetect(".")
		if err != nil {
			return exitWithCode(fmt.Errorf("auto-detecting model: %w", err), 2)
		}
		modelPath = detected
	}

	m, err := model.Load(modelPath)
	if err != nil {
		return exitWithCode(fmt.Errorf("loading model: %w", err), 2)
	}

	var f diagram.Format
	var ext string
	switch diagramFormat {
	case "plantuml":
		f = diagram.PlantUML
		ext = "puml"
	case "mermaid":
		f = diagram.Mermaid
		ext = "mmd"
	default:
		return exitWithCode(fmt.Errorf("unknown diagram format %q: valid values are \"plantuml\" and \"mermaid\"", diagramFormat), 2)
	}

	// Determine which views to export.
	views := make(map[string]model.View)
	if viewKey != "" {
		v, ok := m.Views[viewKey]
		if !ok {
			return exitWithCode(fmt.Errorf("view %q not found", viewKey), 1)
		}
		views[viewKey] = v
	} else {
		views = m.Views
	}

	format, _ := cmd.Flags().GetString("format")

	// When --format json, output structured JSON with diagram source. (#241)
	if format == "json" {
		type diagramEntry struct {
			View   string `json:"view"`
			Format string `json:"format"`
			Source string `json:"source"`
		}
		var entries []diagramEntry
		keys := sortedKeys(views)
		for _, key := range keys {
			result, fmtErr := diagram.FormatView(m, key, f)
			if fmtErr != nil {
				return exitWithCode(fmtErr, 1)
			}
			entries = append(entries, diagramEntry{
				View:   key,
				Format: diagramFormat,
				Source: result,
			})
		}
		data, _ := json.MarshalIndent(entries, "", "  ")
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	for key := range views {
		result, fmtErr := diagram.FormatView(m, key, f)
		if fmtErr != nil {
			return exitWithCode(fmtErr, 1)
		}

		if outputDir == "" {
			_, _ = fmt.Fprint(cmd.OutOrStdout(), result)
			continue
		}

		if err := os.MkdirAll(outputDir, 0750); err != nil {
			return exitWithCode(fmt.Errorf("creating output directory: %w", err), 2)
		}
		outPath := filepath.Join(outputDir, export.SafeViewKey(key)+"."+ext)
		if err := os.WriteFile(outPath, []byte(result), 0600); err != nil { //nolint:gosec // output files are non-sensitive documentation
			return exitWithCode(fmt.Errorf("writing output: %w", err), 2)
		}
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Exported: %s\n", outPath)
	}

	return nil
}

func sortedKeys(views map[string]model.View) []string {
	keys := make([]string, 0, len(views))
	for k := range views {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
