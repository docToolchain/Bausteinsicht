package main

import (
	"encoding/json"
	"fmt"

	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/spf13/cobra"
)

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
	if !isValidKey(viewKey) {
		return exitWithCode(
			fmt.Errorf("invalid view key %q: must start with a letter and contain only letters, digits, hyphens, or underscores", viewKey),
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
	prior, viewExists := m.Views[viewKey]
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
		// Report the view as it is after the change, not just this call's
		// arguments, so a merge into an existing view does not look like it
		// cleared the title or replaced the include list (#641).
		final := m.Views[viewKey]
		result := addViewResult{
			ViewKey:     viewKey,
			Created:     !viewExists,
			Title:       final.Title,
			Scope:       final.Scope,
			Description: final.Description,
			Include:     nonNil(final.Include),
			Added:       newIncludes(prior.Include, includes),
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

// addViewResult is the JSON result of "add view": the view after the change
// plus what this call changed.
type addViewResult struct {
	ViewKey     string   `json:"view_key"`
	Created     bool     `json:"created"`
	Title       string   `json:"title"`
	Scope       string   `json:"scope"`
	Description string   `json:"description,omitempty"`
	Include     []string `json:"include"`
	Added       []string `json:"added"`
}

// newIncludes returns the entries of requested that are not yet in existing,
// without duplicates and in the order requested. It never returns nil, so the
// JSON output has an empty array instead of null.
func newIncludes(existing, requested []string) []string {
	have := make(map[string]bool, len(existing)+len(requested))
	for _, id := range existing {
		have[id] = true
	}
	added := []string{}
	for _, id := range requested {
		if id == "" || have[id] {
			continue
		}
		have[id] = true
		added = append(added, id)
	}
	return added
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
