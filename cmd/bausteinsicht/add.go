package main

import "github.com/spf13/cobra"

func newAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add elements or relationships to the model",
	}

	cmd.AddCommand(newAddElementCmd())
	cmd.AddCommand(newAddRelationshipCmd())

	return cmd
}
