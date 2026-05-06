package main

import (
	"fmt"
	"os"

	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/docToolchain/Bausteinsicht/internal/snapshot"
	"github.com/spf13/cobra"
)

func newSnapshotRestoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore <snapshot-id> <output-path>",
		Short: "Restore a snapshot to a file",
		Long:  "Export a snapshot's model to a JSONC file.",
		Args:  cobra.ExactArgs(2),
		RunE:  runSnapshotRestore,
	}

	cmd.Flags().Bool("force", false, "Overwrite output file if it exists")

	return cmd
}

func runSnapshotRestore(cmd *cobra.Command, args []string) error {
	snapshotID := args[0]
	outputPath := args[1]
	force, _ := cmd.Flags().GetBool("force")

	if err := validatePathContainment(outputPath); err != nil {
		return exitWithCode(err, 2)
	}

	manager := snapshot.NewManager(".")
	if !manager.Exists(snapshotID) {
		return exitWithCode(fmt.Errorf("snapshot not found: %s", snapshotID), 2)
	}

	snap, err := manager.Load(snapshotID)
	if err != nil {
		return exitWithCode(fmt.Errorf("loading snapshot: %w", err), 2)
	}

	// Check if output file already exists
	if _, err := os.Stat(outputPath); err == nil && !force {
		return exitWithCode(fmt.Errorf("output file already exists: %s (use --force to overwrite)", outputPath), 2)
	}

	// Save the model to the output file
	if err := model.Save(outputPath, snap.Model); err != nil {
		return exitWithCode(fmt.Errorf("saving model: %w", err), 2)
	}

	output := fmt.Sprintf("Snapshot restored: %s → %s\n", snapshotID, outputPath)
	_, _ = fmt.Fprint(cmd.OutOrStdout(), output)

	return nil
}
