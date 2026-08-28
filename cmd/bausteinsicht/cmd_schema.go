package main

import (
	"fmt"
	"os"

	"github.com/docToolchain/Bausteinsicht/internal/codegraph"
	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/docToolchain/Bausteinsicht/internal/schema"
	"github.com/spf13/cobra"
)

func newSchemaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Manage JSON Schema for architecture models",
		Long:  "Generate and manage JSON Schema definitions for Bausteinsicht models.",
	}

	cmd.AddCommand(newSchemaGenerateCmd())

	return cmd
}

func newSchemaGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate JSON Schema from Go types",
		Long: `Generate the JSON Schema from Go type definitions.

--type model (default) generates the architecture model schema, saved to
schemas/bausteinsicht.schema.json.

--type codegraph generates the reverse code-governance-bridge contract
schema (#562, ADR-013), saved to schemas/codegraph.schema.json.`,
		RunE: runSchemaGenerate,
	}

	cmd.Flags().String("type", "model", "Schema to generate: model or codegraph")
	cmd.Flags().String("output", "", "Output file (default: schemas/bausteinsicht.schema.json or schemas/codegraph.schema.json, depending on --type)")

	return cmd
}

func runSchemaGenerate(cmd *cobra.Command, _ []string) error {
	schemaType, _ := cmd.Flags().GetString("type")
	outputFile, _ := cmd.Flags().GetString("output")

	var (
		target      any
		title, desc string
	)
	switch schemaType {
	case "model":
		target = model.BausteinsichtModel{}
		title, desc = "Bausteinsicht Model", "Architecture model in Bausteinsicht format"
		if outputFile == "" {
			outputFile = "schemas/bausteinsicht.schema.json"
		}
	case "codegraph":
		target = codegraph.CodeGraph{}
		title, desc = codegraph.SchemaTitle, codegraph.SchemaDescription
		if outputFile == "" {
			outputFile = "schemas/codegraph.schema.json"
		}
	default:
		return exitWithCode(fmt.Errorf("invalid --type %q: expected \"model\" or \"codegraph\"", schemaType), 2)
	}

	// Validate output path to prevent directory traversal (SEC-001)
	if err := validatePathContainment(outputFile); err != nil {
		return exitWithCode(fmt.Errorf("--output: %w", err), 1)
	}

	gen := schema.NewGenerator()
	schemaObj := gen.Generate(target)
	schemaObj.Title = title
	schemaObj.Description = desc

	jsonBytes, err := schemaObj.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to convert schema to JSON: %w", err)
	}

	if err := os.WriteFile(outputFile, jsonBytes, 0600); err != nil {
		return fmt.Errorf("failed to write schema file: %w", err)
	}

	fmt.Printf("✅ Schema generated: %s\n", outputFile)
	fmt.Printf("📊 Properties: %d\n", len(schemaObj.Properties))
	fmt.Printf("📌 Required fields: %d\n", len(schemaObj.Required))

	return nil
}
