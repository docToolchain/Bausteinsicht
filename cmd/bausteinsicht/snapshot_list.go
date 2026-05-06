package main

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/docToolchain/Bausteinsicht/internal/snapshot"
	"github.com/spf13/cobra"
)

func newSnapshotListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all saved snapshots",
		Long:  "Display all snapshots with timestamps, messages, and element/relationship counts.",
		RunE:  runSnapshotList,
	}

	cmd.Flags().String("format", "table", "Output format: table or json")

	return cmd
}

func runSnapshotList(cmd *cobra.Command, _ []string) error {
	format, _ := cmd.Flags().GetString("format")

	manager := snapshot.NewManager(".")
	snapshots, err := manager.List()
	if err != nil {
		return exitWithCode(fmt.Errorf("listing snapshots: %w", err), 2)
	}

	if len(snapshots) == 0 {
		_, _ = fmt.Fprint(cmd.OutOrStdout(), "No snapshots found.\n")
		return nil
	}

	switch format {
	case "json":
		data, err := json.MarshalIndent(snapshots, "", "  ")
		if err != nil {
			return exitWithCode(fmt.Errorf("marshaling snapshots: %w", err), 2)
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))
	case "table":
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tTIMESTAMP\tELEMENTS\tRELATIONSHIPS\tMESSAGE")
		for _, s := range snapshots {
			message := s.Message
			if len(message) > 30 {
				message = message[:27] + "..."
			}
			fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%s\n",
				s.ID,
				s.Timestamp.Format("2006-01-02 15:04:05"),
				s.ElementCount,
				s.RelCount,
				message,
			)
		}
		w.Flush()
	default:
		return exitWithCode(fmt.Errorf("unknown format: %s", format), 2)
	}

	return nil
}
