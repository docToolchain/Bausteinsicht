package main

import (
	"github.com/spf13/cobra"
)

func newSnapshotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "Manage versioned architecture snapshots",
		Long:  "Save, list, delete, diff, and restore architecture snapshots stored in .bausteinsicht-snapshots/",
	}

	cmd.AddCommand(newSnapshotSaveCmd())
	cmd.AddCommand(newSnapshotListCmd())
	cmd.AddCommand(newSnapshotDeleteCmd())
	cmd.AddCommand(newSnapshotDiffCmd())
	cmd.AddCommand(newSnapshotRestoreCmd())

	return cmd
}
