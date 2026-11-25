package main

import (
	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/spf13/cobra"
)

func newRenameCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rename",
		Short: "Rename an element and rewrite every reference to it",
		Long: "Renames the element --id to the new key --to (its new last path segment) and rewrites the relationships, " +
			"view scope/include/exclude, dynamic view steps and constraint selectors that name it or one of its children.\n" +
			"The model is validated before it is saved; nothing is written if the rename would make it invalid.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			id, _ := cmd.Flags().GetString("id")
			to, _ := cmd.Flags().GetString("to")
			return runEdit(cmd, func(m *model.BausteinsichtModel) ([]model.Change, error) {
				return m.RenameElement(id, to)
			})
		},
	}
	cmd.Flags().String("id", "", "Element to rename (dot notation, e.g. shop.web)")
	cmd.Flags().String("to", "", "New key of the element (its new last path segment, no dots)")
	addDryRunFlag(cmd)
	_ = cmd.MarkFlagRequired("id")
	_ = cmd.MarkFlagRequired("to")
	return cmd
}
