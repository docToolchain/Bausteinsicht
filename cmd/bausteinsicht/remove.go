package main

import (
	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/spf13/cobra"
)

func newRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove an element or a relationship from the model",
	}
	cmd.AddCommand(newRemoveElementCmd(), newRemoveRelationshipCmd())
	return cmd
}

func newRemoveElementCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "element",
		Short: "Remove an element and its children",
		Long: "Removes the element --id and its children. If relationships, view include/exclude entries or dynamic view steps " +
			"still reference it, the command fails and lists them; --cascade drops them as well.\n" +
			"A view whose scope is the element, or a constraint that names it, always blocks the removal.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			id, _ := cmd.Flags().GetString("id")
			cascade, _ := cmd.Flags().GetBool("cascade")
			return runEdit(cmd, func(m *model.BausteinsichtModel) ([]model.Change, error) {
				return m.RemoveElement(id, cascade)
			}, model.PatchRemoveChanges)
		},
	}
	cmd.Flags().String("id", "", "Element to remove (dot notation)")
	cmd.Flags().Bool("cascade", false, "Also drop the relationships, view entries and dynamic view steps that reference the element")
	addDryRunFlag(cmd)
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func newRemoveRelationshipCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "relationship",
		Short: "Remove a relationship",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			from, _ := cmd.Flags().GetString("from")
			to, _ := cmd.Flags().GetString("to")
			label, _ := cmd.Flags().GetString("label")
			return runEdit(cmd, func(m *model.BausteinsichtModel) ([]model.Change, error) {
				return m.RemoveRelationship(from, to, label)
			}, model.PatchRemoveChanges)
		},
	}
	cmd.Flags().String("from", "", "Source element (dot notation)")
	cmd.Flags().String("to", "", "Target element (dot notation)")
	cmd.Flags().String("label", "", "Only remove the relationship with this label (default: the first one between the two elements)")
	addDryRunFlag(cmd)
	_ = cmd.MarkFlagRequired("from")
	_ = cmd.MarkFlagRequired("to")
	return cmd
}
