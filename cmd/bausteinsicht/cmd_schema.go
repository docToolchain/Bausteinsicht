package main

import (
	"fmt"
	"os"
	"path/filepath"

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
		Long:  "Generate the JSON Schema from model type definitions and save to schemas/bausteinsicht.schema.json.",
		RunE:  runSchemaGenerate,
	}

	cmd.Flags().String("output", "schemas/bausteinsicht.schema.json", "Output file for the schema")

	return cmd
}

func runSchemaGenerate(cmd *cobra.Command, _ []string) error {
	outputFile, _ := cmd.Flags().GetString("output")

	// Validate output path to prevent directory traversal (SEC-001)
	if err := validatePathContainment(outputFile); err != nil {
		return exitWithCode(fmt.Errorf("--output: %w", err), 1)
	}

	// Create schema generator
	gen := schema.NewGenerator()

	// Generate schema for BausteinsichtModel
	schemaObj := gen.Generate(model.BausteinsichtModel{})

	// Convert to JSON
	jsonBytes, err := schemaObj.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to convert schema to JSON: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputFile, jsonBytes, 0600); err != nil {
		return fmt.Errorf("failed to write schema file: %w", err)
	}

	// Print success message
	fmt.Printf("✅ Schema generated: %s\n", outputFile)
	fmt.Printf("📊 Properties: %d\n", len(schemaObj.Properties))
	fmt.Printf("📌 Required fields: %d\n", len(schemaObj.Required))

	return nil
}
