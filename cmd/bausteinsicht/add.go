package main

import "github.com/spf13/cobra"

func newAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add elements or relationships to the model",
	}

	cmd.AddCommand(newAddElementCmd())
	cmd.AddCommand(newAddRelationshipCmd())
	cmd.AddCommand(newAddFromPatternCmd())

	// Create a pattern sub-group
	patternCmd := &cobra.Command{
		Use:   "pattern",
		Short: "Manage patterns",
	}
	patternCmd.AddCommand(newListPatternsCmd())

	cmd.AddCommand(patternCmd)

	return cmd
}
