package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/docToolchain/Bausteinsicht/internal/diagram"
	"github.com/docToolchain/Bausteinsicht/internal/export"
	dslexport "github.com/docToolchain/Bausteinsicht/internal/exporter/structurizr"
	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/spf13/cobra"
)

func newExportDiagramCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export-diagram",
		Short: "Export views as C4 diagrams (PlantUML, Mermaid, DOT, D2, HTML5, Structurizr DSL)",
		Long:  "Exports architecture views as text-based C4 diagrams (PlantUML, Mermaid, DOT, D2), interactive HTML5 viewer, or Structurizr DSL workspace.",
		RunE:  runExportDiagram,
	}

	cmd.Flags().String("view", "", "Export only this view (by key)")
	cmd.Flags().String("diagram-format", "plantuml", "Diagram format: plantuml, mermaid, dot, d2, html, or structurizr")
	cmd.Flags().String("output", "", "Output directory (default: stdout)")

	return cmd
}

func runExportDiagram(cmd *cobra.Command, _ []string) error {
	modelPath, _ := cmd.Flags().GetString("model")
	viewKey, _ := cmd.Flags().GetString("view")
	diagramFormat, _ := cmd.Flags().GetString("diagram-format")
	outputDir, _ := cmd.Flags().GetString("output")

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

	// Structurizr DSL export: outputs the whole workspace in one file.
	if diagramFormat == "structurizr" {
		// Structurizr exports the entire workspace, not individual views
		if viewKey != "" {
			return exitWithCode(fmt.Errorf("--view is not supported with structurizr format (exports entire workspace)"), 1)
		}
		dsl := dslexport.Export(m)
		if outputDir == "" {
			_, _ = fmt.Fprint(cmd.OutOrStdout(), dsl)
			return nil
		}
		if err := os.MkdirAll(outputDir, 0750); err != nil {
			return exitWithCode(fmt.Errorf("creating output directory: %w", err), 2)
		}
		outPath := filepath.Join(outputDir, "workspace.dsl")
		if err := os.WriteFile(outPath, []byte(dsl), 0600); err != nil { //nolint:gosec
			return exitWithCode(fmt.Errorf("writing output: %w", err), 2)
		}
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Exported: %s\n", outPath)
		return nil
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

	outputFormat, _ := cmd.Flags().GetString("format")

	// Handle new export formats (DOT, D2, HTML) — with JSON envelope support
	switch diagramFormat {
	case "dot", "d2", "html":
		return handleNewFormats(cmd, m, views, diagramFormat, outputFormat, outputDir, viewKey)
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
		return exitWithCode(fmt.Errorf("unknown diagram format %q: valid values are \"plantuml\", \"mermaid\", \"dot\", \"d2\", \"html\", or \"structurizr\"", diagramFormat), 2)
	}

	// When --format json, output structured JSON with diagram source. (#241)
	if outputFormat == "json" {
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

func handleNewFormats(cmd *cobra.Command, m *model.BausteinsichtModel, views map[string]model.View, diagramFormat, outputFormat, outputDir, viewKey string) error {
	var renderFunc func(*model.BausteinsichtModel, string) (string, error)
	var ext string

	switch diagramFormat {
	case "dot":
		renderFunc = diagram.RenderDOT
		ext = "dot"
	case "d2":
		renderFunc = diagram.RenderD2
		ext = "d2"
	case "html":
		renderFunc = diagram.RenderHTML
		ext = "html"
	default:
		return exitWithCode(fmt.Errorf("unsupported format: %s", diagramFormat), 2)
	}

	// When --format json, output structured JSON with diagram source
	if outputFormat == "json" {
		type diagramEntry struct {
			View   string `json:"view"`
			Format string `json:"format"`
			Source string `json:"source"`
		}
		var entries []diagramEntry
		keys := sortedKeys(views)
		for _, key := range keys {
			result, fmtErr := renderFunc(m, key)
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

	// For HTML, create a single file containing all views
	if diagramFormat == "html" {
		// When exporting to HTML, we need to handle multiple views in a single file
		if viewKey != "" {
			// Single view HTML export
			result, err := renderFunc(m, viewKey)
			if err != nil {
				return exitWithCode(err, 1)
			}

			if outputDir == "" {
				_, _ = fmt.Fprint(cmd.OutOrStdout(), result)
				return nil
			}

			if err := os.MkdirAll(outputDir, 0750); err != nil {
				return exitWithCode(fmt.Errorf("creating output directory: %w", err), 2)
			}

			outPath := filepath.Join(outputDir, export.SafeViewKey(viewKey)+".html")
			if err := os.WriteFile(outPath, []byte(result), 0600); err != nil {
				return exitWithCode(fmt.Errorf("writing output: %w", err), 2)
			}
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Exported: %s\n", outPath)
			return nil
		}

		// Multiple views: export each as separate HTML file
		keys := sortedKeys(views)
		for _, key := range keys {
			result, err := renderFunc(m, key)
			if err != nil {
				return exitWithCode(err, 1)
			}

			if outputDir == "" {
				_, _ = fmt.Fprint(cmd.OutOrStdout(), result)
				continue
			}

			if err := os.MkdirAll(outputDir, 0750); err != nil {
				return exitWithCode(fmt.Errorf("creating output directory: %w", err), 2)
			}

			outPath := filepath.Join(outputDir, export.SafeViewKey(key)+".html")
			if err := os.WriteFile(outPath, []byte(result), 0600); err != nil {
				return exitWithCode(fmt.Errorf("writing output: %w", err), 2)
			}
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Exported: %s\n", outPath)
		}
		return nil
	}

	// For DOT and D2: export each view separately
	keys := sortedKeys(views)
	for _, key := range keys {
		result, err := renderFunc(m, key)
		if err != nil {
			return exitWithCode(err, 1)
		}

		if outputDir == "" {
			_, _ = fmt.Fprint(cmd.OutOrStdout(), result)
			continue
		}

		if err := os.MkdirAll(outputDir, 0750); err != nil {
			return exitWithCode(fmt.Errorf("creating output directory: %w", err), 2)
		}

		outPath := filepath.Join(outputDir, "architecture-"+export.SafeViewKey(key)+"."+ext)
		if err := os.WriteFile(outPath, []byte(result), 0600); err != nil {
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
