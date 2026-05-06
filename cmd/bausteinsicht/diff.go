package main

import (
	"encoding/json"
	"fmt"

	"github.com/docToolchain/Bausteinsicht/internal/diff"
	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/spf13/cobra"
)

func newDiffCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Show differences between as-is and to-be architecture",
		Long:  "Compare as-is and to-be sections of the model and report changes.",
		RunE:  runDiff,
	}

	cmd.Flags().String("model", "architecture.jsonc", "Model file path")
	cmd.Flags().String("view", "", "Show diff for one view only (optional)")
	cmd.Flags().String("format", "text", "Output format: text or json")

	return cmd
}

func runDiff(cmd *cobra.Command, _ []string) error {
	modelPath, _ := cmd.Flags().GetString("model")
	format, _ := cmd.Flags().GetString("format")

	if err := validatePathContainment(modelPath); err != nil {
		return exitWithCode(err, 2)
	}

	m, err := model.Load(modelPath)
	if err != nil {
		return exitWithCode(fmt.Errorf("loading model: %w", err), 2)
	}

	if m.AsIs == nil || m.ToBe == nil {
		return exitWithCode(fmt.Errorf("model does not contain asIs and toBe sections"), 1)
	}

	result := diff.Compare(m.AsIs, m.ToBe)

	switch format {
	case "json":
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return exitWithCode(fmt.Errorf("marshaling JSON: %w", err), 2)
		}
		if _, err := fmt.Fprint(cmd.OutOrStdout(), string(data)); err != nil {
			return exitWithCode(fmt.Errorf("writing output: %w", err), 2)
		}
	case "text":
		output := formatDiffAsText(result)
		if _, err := fmt.Fprint(cmd.OutOrStdout(), output); err != nil {
			return exitWithCode(fmt.Errorf("writing output: %w", err), 2)
		}
	default:
		return exitWithCode(fmt.Errorf("invalid format: %s (expected text or json)", format), 2)
	}

	return nil
}

func formatDiffAsText(result *diff.DiffResult) string {
	output := "Architecture Diff\n"
	output += "=================\n\n"

	// Added elements
	if result.Summary.AddedElements > 0 {
		output += fmt.Sprintf("Added (%d):\n", result.Summary.AddedElements)
		for _, change := range result.Elements {
			if change.Type == diff.ChangeAdded && change.ToBe != nil {
				output += fmt.Sprintf("  + %-20s [%s] \"%s\"\n",
					change.ID, change.ToBe.Kind, change.ToBe.Title)
			}
		}
		output += "\n"
	}

	// Removed elements
	if result.Summary.RemovedElements > 0 {
		output += fmt.Sprintf("Removed (%d):\n", result.Summary.RemovedElements)
		for _, change := range result.Elements {
			if change.Type == diff.ChangeRemoved && change.AsIs != nil {
				output += fmt.Sprintf("  - %-20s [%s] \"%s\"\n",
					change.ID, change.AsIs.Kind, change.AsIs.Title)
			}
		}
		output += "\n"
	}

	// Changed elements
	if result.Summary.ChangedElements > 0 {
		output += fmt.Sprintf("Changed (%d):\n", result.Summary.ChangedElements)
		for _, change := range result.Elements {
			if change.Type == diff.ChangeChanged && change.AsIs != nil && change.ToBe != nil {
				output += fmt.Sprintf("  ~ %-20s [%s]\n", change.ID, change.AsIs.Kind)

				// Show what changed
				if change.AsIs.Title != change.ToBe.Title {
					output += fmt.Sprintf("      title: \"%s\" → \"%s\"\n",
						change.AsIs.Title, change.ToBe.Title)
				}
				if change.AsIs.Technology != change.ToBe.Technology {
					output += fmt.Sprintf("      technology: \"%s\" → \"%s\"\n",
						change.AsIs.Technology, change.ToBe.Technology)
				}
				if change.AsIs.Description != change.ToBe.Description {
					output += "      description: changed\n"
				}
				if change.AsIs.Status != change.ToBe.Status {
					output += fmt.Sprintf("      status: \"%s\" → \"%s\"\n",
						change.AsIs.Status, change.ToBe.Status)
				}
			}
		}
		output += "\n"
	}

	// Relationship changes
	if result.Summary.AddedRelationships > 0 {
		output += fmt.Sprintf("Added Relationships (%d):\n", result.Summary.AddedRelationships)
		for _, change := range result.Relationships {
			if change.Type == diff.ChangeAdded && change.ToBe != nil {
				output += fmt.Sprintf("  + %s → %s (%s)\n",
					change.From, change.To, change.ToBe.Label)
			}
		}
		output += "\n"
	}

	if result.Summary.RemovedRelationships > 0 {
		output += fmt.Sprintf("Removed Relationships (%d):\n", result.Summary.RemovedRelationships)
		for _, change := range result.Relationships {
			if change.Type == diff.ChangeRemoved && change.AsIs != nil {
				output += fmt.Sprintf("  - %s → %s (%s)\n",
					change.From, change.To, change.AsIs.Label)
			}
		}
		output += "\n"
	}

	if result.Summary.AddedElements == 0 && result.Summary.RemovedElements == 0 &&
		result.Summary.ChangedElements == 0 && result.Summary.AddedRelationships == 0 &&
		result.Summary.RemovedRelationships == 0 {
		output += "No changes found between as-is and to-be architecture.\n"
	}

	return output
}
