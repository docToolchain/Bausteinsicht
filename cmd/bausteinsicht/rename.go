package main

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/docToolchain/Bausteinsicht/internal/drawio"
	"github.com/docToolchain/Bausteinsicht/internal/model"
	bsync "github.com/docToolchain/Bausteinsicht/internal/sync"
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
			}, model.PatchRenameChanges, migrateRenamedCells)
		},
	}
	cmd.Flags().String("id", "", "Element to rename (dot notation, e.g. shop.web)")
	cmd.Flags().String("to", "", "New key of the element (its new last path segment, no dots)")
	addDryRunFlag(cmd)
	_ = cmd.MarkFlagRequired("id")
	_ = cmd.MarkFlagRequired("to")
	return cmd
}

// migrateRenamedCells carries the rename into the draw.io file and the sync
// state next to the model, so the next sync keeps position, size and style of
// the renamed element and of the connectors at it instead of deleting and
// re-creating them (#686). The model is already saved; if the draw.io file is
// missing there is nothing to migrate, and if it cannot be migrated the next
// sync falls back to deleting and re-creating the cells. The model lock is
// still held by the caller.
func migrateRenamedCells(cmd *cobra.Command, modelPath string, changes []model.Change) {
	if len(changes) == 0 {
		return
	}
	oldID, newID := changes[0].From, changes[0].To
	dir := filepath.Dir(modelPath)
	drawioPath := filepath.Join(dir, "architecture.drawio")
	statePath := filepath.Join(dir, ".bausteinsicht-sync")
	warn := func(err error) {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "note: the draw.io file was not migrated (%v); the next sync re-creates the cells of %s\n", err, newID)
	}

	doc, err := drawio.LoadDocument(drawioPath)
	if errors.Is(err, fs.ErrNotExist) {
		return
	}
	if err != nil {
		warn(err)
		return
	}
	cells := doc.RenameElement(oldID, newID)
	if err := drawio.SaveDocument(drawioPath, doc); err != nil {
		warn(err)
		return
	}
	state, err := bsync.LoadState(statePath)
	if err != nil {
		warn(err)
		return
	}
	if len(state.Elements) > 0 {
		state.RenameElement(oldID, newID)
		if h, err := bsync.ComputeHash(modelPath); err == nil {
			state.ModelHash = h
		}
		if h, err := bsync.ComputeHash(drawioPath); err == nil {
			state.DrawioHash = h
		}
		if err := bsync.SaveState(statePath, state); err != nil {
			warn(err)
			return
		}
	}
	if cells > 0 {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "note: %d draw.io cell(s) of %s renamed to %s, their position and style are kept\n", cells, oldID, newID)
	}
}
