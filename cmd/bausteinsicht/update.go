package main

import (
	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Change fields of existing model entries",
	}
	cmd.AddCommand(newUpdateElementCmd())
	return cmd
}

func newUpdateElementCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "element",
		Short: "Change the title, technology, description or kind of an element",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			id, _ := cmd.Flags().GetString("id")
			var u model.ElementUpdate
			for name, dst := range map[string]**string{
				"title": &u.Title, "technology": &u.Technology, "description": &u.Description, "kind": &u.Kind,
			} {
				if cmd.Flags().Changed(name) {
					value, _ := cmd.Flags().GetString(name)
					*dst = &value
				}
			}
			return runEdit(cmd, func(m *model.BausteinsichtModel) ([]model.Change, error) {
				return m.UpdateElement(id, u)
			}, model.PatchUpdateChanges)
		},
	}
	cmd.Flags().String("id", "", "Element to change (dot notation)")
	cmd.Flags().String("title", "", "New title")
	cmd.Flags().String("technology", "", "New technology (empty clears it)")
	cmd.Flags().String("description", "", "New description (empty clears it)")
	cmd.Flags().String("kind", "", "New element kind (must be defined in the specification)")
	addDryRunFlag(cmd)
	_ = cmd.MarkFlagRequired("id")
	return cmd
}
