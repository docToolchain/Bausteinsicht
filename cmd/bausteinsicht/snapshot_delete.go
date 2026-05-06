package main

import (
	"fmt"

	"github.com/docToolchain/Bausteinsicht/internal/snapshot"
	"github.com/spf13/cobra"
)

func newSnapshotDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <snapshot-id>",
		Short: "Delete a saved snapshot",
		Long:  "Remove a snapshot from .bausteinsicht-snapshots/",
		Args:  cobra.ExactArgs(1),
		RunE:  runSnapshotDelete,
	}

	return cmd
}

func runSnapshotDelete(cmd *cobra.Command, args []string) error {
	snapshotID := args[0]

	manager := snapshot.NewManager(".")
	if !manager.Exists(snapshotID) {
		return exitWithCode(fmt.Errorf("snapshot not found: %s", snapshotID), 2)
	}

	if err := manager.Delete(snapshotID); err != nil {
		return exitWithCode(fmt.Errorf("deleting snapshot: %w", err), 2)
	}

	output := fmt.Sprintf("Snapshot deleted: %s\n", snapshotID)
	_, _ = fmt.Fprint(cmd.OutOrStdout(), output)

	return nil
}
