package main

import (
	"fmt"

	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/docToolchain/Bausteinsicht/internal/snapshot"
	"github.com/spf13/cobra"
)

func newSnapshotSaveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "save",
		Short: "Save current architecture model as a snapshot",
		Long:  "Capture the current architecture model state with an optional message and store it in .bausteinsicht-snapshots/",
		RunE:  runSnapshotSave,
	}

	cmd.Flags().String("model", "architecture.jsonc", "Model file path")
	cmd.Flags().String("message", "", "Optional message describing the snapshot")

	return cmd
}

func runSnapshotSave(cmd *cobra.Command, _ []string) error {
	modelPath, _ := cmd.Flags().GetString("model")
	message, _ := cmd.Flags().GetString("message")

	if err := validatePathContainment(modelPath); err != nil {
		return exitWithCode(err, 2)
	}

	// Load current model
	m, err := model.Load(modelPath)
	if err != nil {
		return exitWithCode(fmt.Errorf("loading model: %w", err), 2)
	}

	// Create snapshot
	snap := snapshot.NewSnapshot(message, m)

	// Save snapshot
	manager := snapshot.NewManager(".")
	if err := manager.Save(snap); err != nil {
		return exitWithCode(fmt.Errorf("saving snapshot: %w", err), 2)
	}

	// Report success
	elementCount := len(flattenElements(m.Model))
	relCount := len(m.Relationships)

	output := fmt.Sprintf("Snapshot saved: %s\n", snap.ID)
	output += fmt.Sprintf("  Timestamp: %s\n", snap.Timestamp.Format("2006-01-02T15:04:05Z"))
	output += fmt.Sprintf("  Elements: %d\n", elementCount)
	output += fmt.Sprintf("  Relationships: %d\n", relCount)
	if message != "" {
		output += fmt.Sprintf("  Message: %s\n", message)
	}

	if _, err := fmt.Fprint(cmd.OutOrStdout(), output); err != nil {
		return exitWithCode(fmt.Errorf("writing output: %w", err), 2)
	}

	return nil
}

// flattenElements counts total elements including nested ones
func flattenElements(elems map[string]model.Element) map[string]model.Element {
	result := make(map[string]model.Element)
	for key, elem := range elems {
		result[key] = elem
		if len(elem.Children) > 0 {
			children := flattenElements(elem.Children)
			for k, v := range children {
				result[key+"."+k] = v
			}
		}
	}
	return result
}
