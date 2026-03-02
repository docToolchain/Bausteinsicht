package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/docToolchain/Bauteinsicht/internal/drawio"
	"github.com/docToolchain/Bauteinsicht/internal/export"
	"github.com/docToolchain/Bauteinsicht/internal/model"
	"github.com/spf13/cobra"
)

func newExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export diagram views to PNG or SVG",
		Long:  "Exports draw.io diagram pages to image files using the draw.io CLI.",
		RunE:  runExport,
	}
	cmd.Flags().String("image-format", "png", "Image format: png or svg")
	cmd.Flags().String("view", "", "Export only this view (by key)")
	cmd.Flags().String("output", ".", "Output directory")
	cmd.Flags().Bool("embed-diagram", false, "Embed draw.io XML source in output")
	return cmd
}

type exportResultJSON struct {
	Files   []string `json:"files"`
	Errors  []string `json:"errors,omitempty"`
	Success bool     `json:"success"`
}

func runExport(cmd *cobra.Command, _ []string) error {
	format, _ := cmd.Flags().GetString("format")
	modelPath, _ := cmd.Flags().GetString("model")
	verbose, _ := cmd.Flags().GetBool("verbose")
	imageFormat, _ := cmd.Flags().GetString("image-format")
	viewFilter, _ := cmd.Flags().GetString("view")
	outputDir, _ := cmd.Flags().GetString("output")
	embedDiagram, _ := cmd.Flags().GetBool("embed-diagram")

	// Validate image format.
	if imageFormat != "png" && imageFormat != "svg" {
		return exitWithCode(fmt.Errorf("unsupported image format %q; use png or svg", imageFormat), 2)
	}

	// Auto-detect model file.
	if modelPath == "" {
		detected, err := model.AutoDetect(".")
		if err != nil {
			return exitWithCode(fmt.Errorf("auto-detecting model: %w", err), 2)
		}
		modelPath = detected
	}

	// Derive drawio path from model path.
	dir := filepath.Dir(modelPath)
	drawioPath := filepath.Join(dir, "architecture.drawio")

	// Load model to get view keys.
	m, err := model.Load(modelPath)
	if err != nil {
		return exitWithCode(fmt.Errorf("loading model: %w", err), 2)
	}

	if len(m.Views) == 0 {
		return exitWithCode(fmt.Errorf("no views to export"), 2)
	}

	// If a specific view was requested, check it exists.
	if viewFilter != "" {
		if _, ok := m.Views[viewFilter]; !ok {
			return exitWithCode(fmt.Errorf("view %q not found in model", viewFilter), 2)
		}
	}

	// Load draw.io document to get page ordering.
	doc, err := drawio.LoadDocument(drawioPath)
	if err != nil {
		return exitWithCode(fmt.Errorf("loading draw.io file: %w", err), 2)
	}

	// Detect draw.io CLI binary.
	binary, err := export.DetectDrawioBinary()
	if err != nil {
		return exitWithCode(err, 2)
	}

	if verbose && format != "json" {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Using draw.io CLI: %s\n", binary)
	}

	// Ensure output directory exists.
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return exitWithCode(fmt.Errorf("creating output directory: %w", err), 2)
	}

	// Build the list of pages to export.
	pages := doc.Pages()
	type viewExport struct {
		key       string
		pageIndex int // 1-based
	}
	var exports []viewExport

	for viewKey := range m.Views {
		if viewFilter != "" && viewKey != viewFilter {
			continue
		}
		pageID := "view-" + viewKey
		for i, p := range pages {
			if p.ID() == pageID {
				exports = append(exports, viewExport{key: viewKey, pageIndex: i + 1})
				break
			}
		}
	}

	if len(exports) == 0 {
		return exitWithCode(fmt.Errorf("no matching pages found in draw.io file"), 2)
	}

	// Export each page.
	var files []string
	var exportErrors []string

	for _, ex := range exports {
		outFile := filepath.Join(outputDir, export.OutputFileName(ex.key, imageFormat))

		if verbose && format != "json" {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Exporting view %q to %s\n", ex.key, outFile)
		}

		err := export.ExportPage(binary, export.ExportOptions{
			Format:       imageFormat,
			PageIndex:    ex.pageIndex,
			OutputPath:   outFile,
			EmbedDiagram: embedDiagram,
			InputFile:    drawioPath,
		})
		if err != nil {
			exportErrors = append(exportErrors, fmt.Sprintf("view %q: %v", ex.key, err))
			continue
		}
		files = append(files, outFile)
	}

	// Output results.
	if format == "json" {
		result := exportResultJSON{
			Files:   files,
			Errors:  exportErrors,
			Success: len(exportErrors) == 0,
		}
		out, _ := json.MarshalIndent(result, "", "  ")
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(out))
	} else {
		for _, f := range files {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Exported: %s\n", f)
		}
		for _, e := range exportErrors {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "ERROR: %s\n", e)
		}
	}

	if len(exportErrors) > 0 {
		return exitWithCode(fmt.Errorf("%d export(s) failed", len(exportErrors)), 1)
	}
	return nil
}
