package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/docToolchain/Bausteinsicht/internal/template"
	"github.com/spf13/cobra"
)

func newGenerateTemplateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate-template",
		Short: "Generate a draw.io template from element specification",
		Long:  "Creates a draw.io template file with visual styles for all element kinds defined in the spec.",
		RunE:  runGenerateTemplate,
	}

	cmd.Flags().String("model", "", "Model file (default: auto-detect)")
	cmd.Flags().String("output", "architecture-template.drawio", "Output template file")
	cmd.Flags().String("style", "default", "Visual preset: default, c4, minimal, or dark")

	return cmd
}

func runGenerateTemplate(cmd *cobra.Command, _ []string) error {
	modelPath, _ := cmd.Flags().GetString("model")
	outputPath, _ := cmd.Flags().GetString("output")
	style, _ := cmd.Flags().GetString("style")

	// Validate output path containment
	if err := validatePathContainment(outputPath); err != nil {
		return exitWithCode(fmt.Errorf("--output: %w", err), 2)
	}

	// Auto-detect model if not provided
	if modelPath == "" {
		detected, err := model.AutoDetect(".")
		if err != nil {
			return exitWithCode(fmt.Errorf("auto-detecting model: %w", err), 2)
		}
		modelPath = detected
	}

	// Load model
	m, err := model.Load(modelPath)
	if err != nil {
		return exitWithCode(fmt.Errorf("loading model: %w", err), 2)
	}

	// Validate style
	validStyles := map[string]bool{
		"default":  true,
		"c4":       true,
		"minimal":  true,
		"dark":     true,
	}
	if !validStyles[style] {
		return exitWithCode(fmt.Errorf("unknown style %q: valid values are default, c4, minimal, dark", style), 2)
	}

	// Generate template
	gen := template.NewGenerator(m.Specification, style)
	templateXML := gen.Generate()

	// Write output
	outputDir := filepath.Dir(outputPath)
	if outputDir != "." && outputDir != "" {
		if err := os.MkdirAll(outputDir, 0750); err != nil {
			return exitWithCode(fmt.Errorf("creating output directory: %w", err), 2)
		}
	}

	if err := os.WriteFile(outputPath, []byte(templateXML), 0600); err != nil { //nolint:gosec
		return exitWithCode(fmt.Errorf("writing template: %w", err), 2)
	}

	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Generated template: %s\n", outputPath)
	return nil
}
