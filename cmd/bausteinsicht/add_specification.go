package main

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/spf13/cobra"
)

// validSpecKeyPattern matches specification keys: lowercase letters, digits, underscores, hyphens.
var validSpecKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

// isValidSpecKey checks if the given spec key is valid.
func isValidSpecKey(key string) bool {
	return validSpecKeyPattern.MatchString(key)
}

func newAddSpecificationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "specification",
		Short: "Add element or relationship types to the specification",
	}

	cmd.AddCommand(newAddSpecificationElementCmd())
	cmd.AddCommand(newAddSpecificationRelationshipCmd())

	return cmd
}

func newAddSpecificationElementCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "element <key>",
		Short: "Add an element type to the specification",
		Args:  cobra.ExactArgs(1),
		RunE:  runAddSpecificationElement,
	}

	cmd.Flags().String("notation", "", "Notation/display text for this element type (required)")
	cmd.Flags().String("description", "", "Description of this element type")
	cmd.Flags().Bool("container", false, "Whether this element can contain children")

	_ = cmd.MarkFlagRequired("notation")

	return cmd
}

func runAddSpecificationElement(cmd *cobra.Command, args []string) error {
	key := args[0]
	notation, _ := cmd.Flags().GetString("notation")
	description, _ := cmd.Flags().GetString("description")
	container, _ := cmd.Flags().GetBool("container")

	modelPath, _ := cmd.Flags().GetString("model")
	format, _ := cmd.Flags().GetString("format")

	// Validate key format
	if !isValidSpecKey(key) {
		return exitWithCode(
			fmt.Errorf("invalid specification key %q: must contain only lowercase letters, digits, hyphens, or underscores", key),
			1,
		)
	}

	// Load model
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

	// Add element kind
	err = m.AddSpecificationElement(key, model.ElementKind{
		Notation:    notation,
		Description: description,
		Container:   container,
	})
	if err != nil {
		return exitWithCode(fmt.Errorf("adding element: %w", err), 1)
	}

	// Save model
	if err := model.Save(modelPath, m); err != nil {
		return exitWithCode(fmt.Errorf("saving model: %w", err), 2)
	}

	// Output result
	if format == "json" {
		result := map[string]interface{}{
			"key":       key,
			"notation":  notation,
			"container": container,
		}
		if description != "" {
			result["description"] = description
		}
		jsonBytes, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("encoding result: %w", err)
		}
		fmt.Println(string(jsonBytes))
	} else {
		fmt.Printf("Element type '%s' added to specification\n", key)
		fmt.Printf("  Notation: %s\n", notation)
		if description != "" {
			fmt.Printf("  Description: %s\n", description)
		}
		if container {
			fmt.Printf("  Container: yes\n")
		}
	}

	return nil
}

func newAddSpecificationRelationshipCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "relationship <key>",
		Short: "Add a relationship type to the specification",
		Args:  cobra.ExactArgs(1),
		RunE:  runAddSpecificationRelationship,
	}

	cmd.Flags().String("notation", "", "Notation/display text for this relationship type (required)")
	cmd.Flags().String("description", "", "Description of this relationship type")
	cmd.Flags().Bool("dashed", false, "Whether this relationship is displayed as a dashed line")

	_ = cmd.MarkFlagRequired("notation")

	return cmd
}

func runAddSpecificationRelationship(cmd *cobra.Command, args []string) error {
	key := args[0]
	notation, _ := cmd.Flags().GetString("notation")
	description, _ := cmd.Flags().GetString("description")
	dashed, _ := cmd.Flags().GetBool("dashed")

	modelPath, _ := cmd.Flags().GetString("model")
	format, _ := cmd.Flags().GetString("format")

	// Validate key format
	if !isValidSpecKey(key) {
		return exitWithCode(
			fmt.Errorf("invalid specification key %q: must contain only lowercase letters, digits, hyphens, or underscores", key),
			1,
		)
	}

	// Load model
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

	// Add relationship kind
	err = m.AddSpecificationRelationship(key, model.RelationshipKind{
		Notation: notation,
		Dashed:   dashed,
	})
	if err != nil {
		return exitWithCode(fmt.Errorf("adding relationship: %w", err), 1)
	}

	// Save model
	if err := model.Save(modelPath, m); err != nil {
		return exitWithCode(fmt.Errorf("saving model: %w", err), 2)
	}

	// Output result
	if format == "json" {
		result := map[string]interface{}{
			"key":      key,
			"notation": notation,
			"dashed":   dashed,
		}
		if description != "" {
			result["description"] = description
		}
		jsonBytes, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("encoding result: %w", err)
		}
		fmt.Println(string(jsonBytes))
	} else {
		fmt.Printf("Relationship type '%s' added to specification\n", key)
		fmt.Printf("  Notation: %s\n", notation)
		if description != "" {
			fmt.Printf("  Description: %s\n", description)
		}
		if dashed {
			fmt.Printf("  Style: dashed\n")
		}
	}

	return nil
}
