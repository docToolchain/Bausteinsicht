package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/docToolchain/Bausteinsicht/internal/diagram"
	"github.com/docToolchain/Bausteinsicht/internal/export"
	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/spf13/cobra"
)

func newExportSequenceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export-sequence",
		Short: "Export dynamic views as PlantUML or Mermaid sequence diagrams",
		Long:  "Exports dynamic views (sequence diagrams) as PlantUML (.puml) or Mermaid (.md) text files.",
		RunE:  runExportSequence,
	}
	cmd.Flags().String("view", "", "Export only this dynamic view (by key)")
	cmd.Flags().String("diagram-format", "plantuml", "Diagram format: plantuml or mermaid")
	cmd.Flags().String("output", "", "Output directory (default: stdout)")
	return cmd
}

func runExportSequence(cmd *cobra.Command, _ []string) error {
	modelPath, _ := cmd.Flags().GetString("model")
	viewKey, _ := cmd.Flags().GetString("view")
	diagramFormat, _ := cmd.Flags().GetString("diagram-format")
	outputDir, _ := cmd.Flags().GetString("output")
	format, _ := cmd.Flags().GetString("format")

	if outputDir != "" {
		if err := validatePathContainment(outputDir); err != nil {
			return exitWithCode(fmt.Errorf("--output: %w", err), 2)
		}
	}

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

	var ext string
	switch diagramFormat {
	case "plantuml":
		ext = "puml"
	case "mermaid":
		ext = "md"
	default:
		return exitWithCode(fmt.Errorf("unknown diagram format %q: valid values are \"plantuml\" and \"mermaid\"", diagramFormat), 2)
	}

	// Select views to export.
	views := m.DynamicViews
	if viewKey != "" {
		var found *model.DynamicView
		for i := range m.DynamicViews {
			if m.DynamicViews[i].Key == viewKey {
				found = &m.DynamicViews[i]
				break
			}
		}
		if found == nil {
			return exitWithCode(fmt.Errorf("dynamic view %q not found", viewKey), 1)
		}
		views = []model.DynamicView{*found}
	}

	if len(views) == 0 {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "No dynamic views defined in model.")
		return nil
	}

	flat, err := model.FlattenElements(m)
	if err != nil {
		return exitWithCode(fmt.Errorf("flattening elements: %w", err), 2)
	}

	render := func(v model.DynamicView) string {
		if diagramFormat == "mermaid" {
			return diagram.RenderSequenceMermaid(v, flat)
		}
		return diagram.RenderSequencePlantUML(v, flat)
	}

	// JSON output.
	if format == "json" {
		type entry struct {
			View   string `json:"view"`
			Format string `json:"format"`
			Source string `json:"source"`
		}
		var entries []entry
		for _, v := range views {
			entries = append(entries, entry{View: v.Key, Format: diagramFormat, Source: render(v)})
		}
		data, _ := json.MarshalIndent(entries, "", "  ")
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	// Text / file output.
	for _, v := range views {
		source := render(v)

		if outputDir == "" {
			_, _ = fmt.Fprint(cmd.OutOrStdout(), source)
			continue
		}

		if err := os.MkdirAll(outputDir, 0750); err != nil {
			return exitWithCode(fmt.Errorf("creating output directory: %w", err), 2)
		}
		filename := "sequence-" + export.SafeViewKey(v.Key) + "." + ext
		outPath := filepath.Join(outputDir, filename)
		if err := os.WriteFile(outPath, []byte(source), 0600); err != nil { //nolint:gosec // output files are non-sensitive documentation
			return exitWithCode(fmt.Errorf("writing output: %w", err), 2)
		}
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Exported: %s\n", outPath)
	}

	return nil
}
