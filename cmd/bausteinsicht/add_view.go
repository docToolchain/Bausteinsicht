package main

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/spf13/cobra"
)

// validViewKeyPattern matches view keys: lowercase letters, digits, hyphens.
var validViewKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

// isValidViewKey checks if the given view key is valid.
func isValidViewKey(key string) bool {
	return validViewKeyPattern.MatchString(key)
}

func newAddViewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "view <view-key>",
		Short: "Create a new view or modify a view's include list",
		Args:  cobra.ExactArgs(1),
		RunE:  runAddView,
	}

	cmd.Flags().String("scope", "", "Scope element ID (parent element to show)")
	cmd.Flags().StringSlice("include", []string{}, "Elements to include in view (repeatable)")
	cmd.Flags().String("title", "", "View title (display name) (required for new views)")
	cmd.Flags().String("description", "", "View description")

	return cmd
}

func runAddView(cmd *cobra.Command, args []string) error {
	viewKey := args[0]
	scope, _ := cmd.Flags().GetString("scope")
	includes, _ := cmd.Flags().GetStringSlice("include")
	title, _ := cmd.Flags().GetString("title")
	description, _ := cmd.Flags().GetString("description")

	modelPath, _ := cmd.Flags().GetString("model")
	format, _ := cmd.Flags().GetString("format")

	// Validate view key format
	if !isValidViewKey(viewKey) {
		return exitWithCode(
			fmt.Errorf("invalid view key %q: must contain only lowercase letters, digits, hyphens, or underscores", viewKey),
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

	// Check if this is a new view or update
	_, viewExists := m.Views[viewKey]
	if !viewExists && title == "" {
		return exitWithCode(
			fmt.Errorf("title is required for new views"),
			1,
		)
	}

	// Create view struct
	view := model.View{
		Title:       title,
		Scope:       scope,
		Include:     includes,
		Description: description,
	}

	// Add or update view
	err = m.AddView(viewKey, view)
	if err != nil {
		return exitWithCode(fmt.Errorf("adding view: %w", err), 1)
	}

	// Save model
	if err := model.Save(modelPath, m); err != nil {
		return exitWithCode(fmt.Errorf("saving model: %w", err), 2)
	}

	// Output result
	if format == "json" {
		result := map[string]interface{}{
			"view_key": viewKey,
			"title":    title,
			"scope":    scope,
			"include":  includes,
		}
		jsonBytes, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("encoding result: %w", err)
		}
		fmt.Println(string(jsonBytes))
	} else {
		fmt.Printf("View '%s' added to model\n", viewKey)
		if title != "" {
			fmt.Printf("  Title: %s\n", title)
		}
		if scope != "" {
			fmt.Printf("  Scope: %s\n", scope)
		}
		if len(includes) > 0 {
			fmt.Printf("  Includes: %v\n", includes)
		}
	}

	return nil
}
