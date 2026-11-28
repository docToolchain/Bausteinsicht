package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/docToolchain/Bausteinsicht/internal/importer/likec4"
	"github.com/docToolchain/Bausteinsicht/internal/importer/structurizr"
	"github.com/docToolchain/Bausteinsicht/internal/importer/xmi"
	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/docToolchain/Bausteinsicht/internal/template"
	"github.com/spf13/cobra"
)

func newImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import <input-file>",
		Short: "Import an architecture model from Structurizr DSL, LikeC4, or XMI",
		Long: `Imports an architecture model from an external DSL format and writes a
Bausteinsicht-compatible architecture.jsonc file.

Supported formats:
  structurizr   Structurizr DSL (.dsl)
  likec4        LikeC4 DSL (.c4)
  xmi           XMI 2.1 — Enterprise Architect exports (.xmi, .xml)

Exit codes:
  0   import successful
  1   import failed (unknown --from, unreadable input, parse/encoding error)
  2   output file already exists (use --force to overwrite, or --dry-run to skip the check)`,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runImport,
	}
	cmd.Flags().String("from", "", "Source format: structurizr, likec4, or xmi (required)")
	cmd.Flags().String("output", "architecture.jsonc", "Output model file path")
	cmd.Flags().Bool("dry-run", false, "Print generated model to stdout instead of writing file")
	cmd.Flags().Bool("force", false, "Overwrite output file if it already exists")
	cmd.Flags().String("kind-map", "", "XMI only: comma-separated Type=kind overrides (e.g. Component=service,Class=entity)")
	cmd.Flags().Bool("generate-template", false, "Also write template.drawio next to the output file, with a style for every imported kind")
	cmd.Flags().Bool("json", false, "Print the post-import summary as JSON instead of human-readable text")
	_ = cmd.MarkFlagRequired("from")
	return cmd
}

func runImport(cmd *cobra.Command, args []string) error {
	inputPath := args[0]
	from, _ := cmd.Flags().GetString("from")
	outputPath, _ := cmd.Flags().GetString("output")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")
	jsonSummary, _ := cmd.Flags().GetBool("json")
	genTemplate, _ := cmd.Flags().GetBool("generate-template")

	from = strings.ToLower(strings.TrimSpace(from))
	if from != "structurizr" && from != "likec4" && from != "xmi" {
		return exitWithCode(fmt.Errorf("unknown format %q: valid values are \"structurizr\", \"likec4\", and \"xmi\"", from), 1)
	}

	if err := validatePathContainment(inputPath); err != nil {
		return exitWithCode(fmt.Errorf("input: %w", err), 1)
	}
	if err := validatePathContainment(outputPath); err != nil {
		return exitWithCode(fmt.Errorf("--output: %w", err), 1)
	}

	var overwriting bool
	if !dryRun {
		if _, err := os.Stat(outputPath); err == nil {
			overwriting = true
		}
	}
	if !dryRun && !force && overwriting {
		return exitWithCode(
			fmt.Errorf("output file %q already exists — use --force to overwrite", outputPath),
			2,
		)
	}

	// The template belongs to the model: it lies next to it, where sync looks
	// for it. An existing one is the user's, so refuse before anything is written (#688).
	templatePath := filepath.Join(filepath.Dir(outputPath), "template.drawio")
	if genTemplate && !dryRun {
		if _, err := os.Stat(templatePath); err == nil {
			return exitWithCode(fmt.Errorf("template %q already exists — remove it or run generate-template with --output", templatePath), 2)
		}
	}

	var (
		resultModel *model.BausteinsichtModel
		warnings    []string
	)

	switch from {
	case "structurizr":
		r, err := structurizr.Import(inputPath)
		if err != nil {
			return exitWithCode(fmt.Errorf("import failed: %w", err), 1)
		}
		resultModel, warnings = r.Model, r.Warnings
	case "likec4":
		r, err := likec4.Import(inputPath)
		if err != nil {
			return exitWithCode(fmt.Errorf("import failed: %w", err), 1)
		}
		resultModel, warnings = r.Model, r.Warnings
	case "xmi":
		kindMapStr, _ := cmd.Flags().GetString("kind-map")
		kindMap, err := xmi.ParseKindMap(kindMapStr)
		if err != nil {
			return exitWithCode(fmt.Errorf("--kind-map: %w", err), 1)
		}
		r, err := xmi.Import(inputPath, kindMap)
		if err != nil {
			return exitWithCode(err, 1)
		}
		resultModel, warnings = r.Model, r.Warnings
	}

	// A file in another format (or without any model) imports as an empty model;
	// writing it would overwrite the existing model with nothing (#705).
	if flat, _ := model.FlattenElements(resultModel); len(flat) == 0 {
		return exitWithCode(fmt.Errorf("no elements found in %s - is --from %s correct?", inputPath, from), 1)
	}

	data, err := json.MarshalIndent(resultModel, "", "  ")
	if err != nil {
		return exitWithCode(fmt.Errorf("encoding model: %w", err), 1)
	}

	// The directory exists before the template is created in it; a dry run
	// writes nothing, so it does not create it.
	if !dryRun {
		if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
			return exitWithCode(fmt.Errorf("creating output directory: %w", err), 1)
		}
	}

	// The template is created first and exclusively: an import that overlaps
	// with another one in the directory, or a dangling link, is refused before
	// the model is written. A failing model write takes the template back.
	templateWritten := false
	if genTemplate && !dryRun {
		xml := template.NewGenerator(resultModel.Specification, "default").Generate()
		if err := createFileExclusively(templatePath, []byte(xml)); err != nil {
			if errors.Is(err, fs.ErrExist) {
				return exitWithCode(fmt.Errorf("template %q already exists — remove it or run generate-template with --output", templatePath), 2)
			}
			return exitWithCode(fmt.Errorf("writing %s: %w", templatePath, err), 1)
		}
		templateWritten = true
	}

	if dryRun {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), string(data)); err != nil {
			return err
		}
		for _, w := range warnings {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "WARNING: %s\n", w); err != nil {
				return err
			}
		}
		return nil
	}

	if err := os.WriteFile(outputPath, append(data, '\n'), 0o644); err != nil {
		if templateWritten {
			_ = os.Remove(templatePath)
		}
		return exitWithCode(fmt.Errorf("writing %s: %w", outputPath, err), 1)
	}

	if jsonSummary {
		return printImportSummaryJSON(cmd, resultModel, outputPath, warnings)
	}
	return printImportSummaryText(cmd, resultModel, outputPath, warnings, overwriting)
}

// printImportSummaryText prints the human-readable post-import summary:
// per-kind element counts, output path, generated view count (omitted when
// zero, since e.g. XMI never populates views), warnings, and a next-steps
// hint. On a --force overwrite of an existing file the hint points at
// `bausteinsicht diff` instead of `sync`, since the existing file likely
// already has synced draw.io state.
func printImportSummaryText(cmd *cobra.Command, m *model.BausteinsichtModel, outputPath string, warnings []string, overwriting bool) error {
	out := cmd.OutOrStdout()
	if _, err := fmt.Fprintf(out, "Imported model written to %s\n", outputPath); err != nil {
		return err
	}
	for _, w := range warnings {
		if _, err := fmt.Fprintf(out, "WARNING: %s\n", w); err != nil {
			return err
		}
	}

	counts, err := elementCountsByKind(m)
	if err != nil {
		return err
	}
	kinds := make([]string, 0, len(counts))
	for kind := range counts {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	for _, kind := range kinds {
		if _, err := fmt.Fprintf(out, "%s: %d\n", kind, counts[kind]); err != nil {
			return err
		}
	}
	if len(m.Views) > 0 {
		if _, err := fmt.Fprintf(out, "views: %d\n", len(m.Views)); err != nil {
			return err
		}
	}

	if overwriting {
		_, err = fmt.Fprintln(out, "Next steps: bausteinsicht diff")
	} else {
		_, err = fmt.Fprintln(out, "Next steps: bausteinsicht sync, bausteinsicht export diagram")
	}
	return err
}

// importSummary is the --json shape for the post-import summary.
type importSummary struct {
	Elements   map[string]int `json:"elements"`
	OutputPath string         `json:"outputPath"`
	Views      int            `json:"views"`
	Warnings   []string       `json:"warnings"`
}

// printImportSummaryJSON prints the post-import summary as a single JSON
// object, for LLM/CI workflows that would otherwise have to parse the
// human-readable text (QG-3, LLM-friendliness). Unlike the text summary,
// "views" is always present (0 rather than omitted), since machine consumers
// should not have to distinguish "zero" from "absent".
func printImportSummaryJSON(cmd *cobra.Command, m *model.BausteinsichtModel, outputPath string, warnings []string) error {
	counts, err := elementCountsByKind(m)
	if err != nil {
		return err
	}
	if warnings == nil {
		warnings = []string{}
	}
	summary := importSummary{
		Elements:   counts,
		OutputPath: outputPath,
		Views:      len(m.Views),
		Warnings:   warnings,
	}
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding summary: %w", err)
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return err
}

// elementCountsByKind returns the number of elements per kind in the model.
func elementCountsByKind(m *model.BausteinsichtModel) (map[string]int, error) {
	elements, err := model.FlattenElements(m)
	if err != nil {
		return nil, fmt.Errorf("flattening imported model: %w", err)
	}
	counts := make(map[string]int)
	for _, elem := range elements {
		counts[elem.Kind]++
	}
	return counts, nil
}

// createFileExclusively writes data to a new file; an existing file or link of
// that name is an fs.ErrExist error and is left alone.
func createFileExclusively(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644) //nolint:gosec
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return err
	}
	return f.Close()
}
