package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/docToolchain/Bauteinsicht/internal/model"
	"github.com/docToolchain/Bauteinsicht/internal/table"
	"github.com/spf13/cobra"
)

func newExportTableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export-table",
		Short: "Export element attributes as AsciiDoc or Markdown table",
		Long:  "Exports view elements as a table with columns: Element, Kind, Technology, Description.",
		RunE:  runExportTable,
	}

	cmd.Flags().String("view", "", "Export only this view (by key)")
	cmd.Flags().String("table-format", "adoc", "Table format: adoc or md")
	cmd.Flags().String("output", "", "Output directory (default: stdout)")
	cmd.Flags().Bool("combined", false, "Export all elements across all views (deduplicated)")

	return cmd
}

func runExportTable(cmd *cobra.Command, _ []string) error {
	modelPath, _ := cmd.Flags().GetString("model")
	format, _ := cmd.Flags().GetString("format")
	viewKey, _ := cmd.Flags().GetString("view")
	tableFormat, _ := cmd.Flags().GetString("table-format")
	outputDir, _ := cmd.Flags().GetString("output")
	combined, _ := cmd.Flags().GetBool("combined")

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

	// When --format json is set, output structured JSON instead of a table. (#239)
	if format == "json" {
		return exportTableJSON(cmd, m, viewKey, combined)
	}

	var f table.Format
	switch tableFormat {
	case "adoc":
		f = table.AsciiDoc
	case "md":
		f = table.Markdown
	default:
		return exitWithCode(fmt.Errorf("unknown table format %q: valid values are \"adoc\" and \"md\"", tableFormat), 2)
	}

	var result string
	var filename string

	switch {
	case combined:
		result, err = table.FormatCombined(m, f)
		filename = "elements." + tableFormat
	case viewKey != "":
		result, err = table.FormatView(m, viewKey, f)
		filename = viewKey + "-elements." + tableFormat
	default:
		result, err = table.FormatAllViews(m, f)
		filename = "all-views-elements." + tableFormat
	}
	if err != nil {
		return exitWithCode(err, 1)
	}

	if outputDir == "" {
		_, _ = fmt.Fprint(cmd.OutOrStdout(), result)
		return nil
	}

	outPath := filepath.Join(outputDir, filename)
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		return exitWithCode(fmt.Errorf("creating output directory: %w", err), 2)
	}
	if err := os.WriteFile(outPath, []byte(result), 0600); err != nil { //nolint:gosec // output files are non-sensitive documentation
		return exitWithCode(fmt.Errorf("writing output: %w", err), 2)
	}

	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Exported: %s\n", outPath)
	return nil
}

// exportTableJSON outputs the table data as JSON. (#239)
func exportTableJSON(cmd *cobra.Command, m *model.BausteinsichtModel, viewKey string, combined bool) error {
	rows, err := table.CollectRows(m, viewKey, combined)
	if err != nil {
		return exitWithCode(err, 1)
	}
	data, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return exitWithCode(fmt.Errorf("marshaling JSON: %w", err), 2)
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return nil
}
