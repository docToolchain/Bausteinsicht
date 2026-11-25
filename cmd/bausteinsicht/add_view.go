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

	// Save model: insert the new view or update the existing one in place, so
	// comments and formatting of the rest of the file stay as they are (#635).
	added := newIncludes(prior.Include, includes)
	patch := func(data []byte) ([]byte, error) {
		return patchView(data, viewKey, viewExists, m.Views[viewKey], view, added)
	}
	if err := savePatched(cmd.ErrOrStderr(), modelPath, m, patch); err != nil {
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
			Added:       added,
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

// patchView applies "add view" to the raw model text: a new view is inserted
// into "views"; for an existing view the fields given on the command line are
// replaced and the new include entries are appended, nothing else is touched.
func patchView(data []byte, key string, exists bool, final, requested model.View, added []string) ([]byte, error) {
	viewsPath := []string{"views"}
	if !exists {
		entry, err := marshalEntryJSON(final, 2)
		if err != nil {
			return nil, err
		}
		return model.InsertObjectEntry(data, viewsPath, key, entry)
	}

	viewPath := append(append([]string{}, viewsPath...), key)
	var err error
	for _, f := range []struct{ name, value string }{
		{"title", requested.Title},
		{"scope", requested.Scope},
		{"description", requested.Description},
	} {
		if f.value == "" {
			continue
		}
		quoted, qerr := marshalEntryJSON(f.value, 0)
		if qerr != nil {
			return nil, qerr
		}
		if data, err = model.UpsertObjectField(data, viewPath, f.name, quoted); err != nil {
			return nil, err
		}
	}
	if len(added) == 0 {
		return data, nil
	}
	includePath := append(append([]string{}, viewPath...), "include")
	if !model.HasPath(data, includePath) {
		list, lerr := marshalEntryJSON(added, 0)
		if lerr != nil {
			return nil, lerr
		}
		return model.InsertObjectEntry(data, viewPath, "include", list)
	}
	for _, id := range added {
		quoted, qerr := marshalEntryJSON(id, 0)
		if qerr != nil {
			return nil, qerr
		}
		if data, err = model.AppendArrayEntry(data, includePath, quoted); err != nil {
			return nil, err
		}
	}
	return data, nil
}
