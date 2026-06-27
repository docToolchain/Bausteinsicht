package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/docToolchain/Bausteinsicht/internal/importer/likec4"
	"github.com/docToolchain/Bausteinsicht/internal/importer/structurizr"
	"github.com/docToolchain/Bausteinsicht/internal/importer/xmi"
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
	_ = cmd.MarkFlagRequired("from")
	return cmd
}

func runImport(cmd *cobra.Command, args []string) error {
	inputPath := args[0]
	from, _ := cmd.Flags().GetString("from")
	outputPath, _ := cmd.Flags().GetString("output")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

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

	if !dryRun && !force {
		if _, err := os.Stat(outputPath); err == nil {
			return exitWithCode(
				fmt.Errorf("output file %q already exists — use --force to overwrite", outputPath),
				2,
			)
		}
	}

	var (
		importedModel any
		warnings      []string
	)

	switch from {
	case "structurizr":
		r, err := structurizr.Import(inputPath)
		if err != nil {
			return exitWithCode(fmt.Errorf("import failed: %w", err), 1)
		}
		importedModel, warnings = r.Model, r.Warnings
	case "likec4":
		r, err := likec4.Import(inputPath)
		if err != nil {
			return exitWithCode(fmt.Errorf("import failed: %w", err), 1)
		}
		importedModel, warnings = r.Model, r.Warnings
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
		importedModel, warnings = r.Model, r.Warnings
	}

	data, err := json.MarshalIndent(importedModel, "", "  ")
	if err != nil {
		return exitWithCode(fmt.Errorf("encoding model: %w", err), 1)
	}

	if dryRun {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), string(data)); err != nil {
			return err
		}
	} else {
		if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
			return exitWithCode(fmt.Errorf("creating output directory: %w", err), 1)
		}
		if err := os.WriteFile(outputPath, append(data, '\n'), 0o644); err != nil {
			return exitWithCode(fmt.Errorf("writing %s: %w", outputPath, err), 1)
		}
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Imported model written to %s\n", outputPath); err != nil {
			return err
		}
	}

	for _, w := range warnings {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "WARNING: %s\n", w); err != nil {
			return err
		}
	}

	return nil
}
