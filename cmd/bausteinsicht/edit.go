package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/spf13/cobra"
)

// editResult is the --format json result of rename, update and remove.
type editResult struct {
	DryRun  bool           `json:"dry_run"`
	Changes []model.Change `json:"changes"`
}

func addDryRunFlag(cmd *cobra.Command) {
	cmd.Flags().Bool("dry-run", false, "Show what would change without saving")
}

// runEdit loads the model, applies one edit, refuses it if it would introduce
// validation errors, and saves it (unless --dry-run). Saving rewrites the file
// like "add view" did before #635, which is announced on stderr.
func runEdit(cmd *cobra.Command, apply func(m *model.BausteinsichtModel) ([]model.Change, error)) error {
	modelPath, _ := cmd.Flags().GetString("model")
	format, _ := cmd.Flags().GetString("format")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if modelPath == "" {
		detected, err := model.AutoDetect(".")
		if err != nil {
			return exitWithCode(fmt.Errorf("auto-detecting model: %w", err), 2)
		}
		modelPath = detected
	}
	unlock, err := lockModel(modelPath)
	if err != nil {
		return exitWithCode(err, 2)
	}
	defer unlock()

	m, err := model.Load(modelPath)
	if err != nil {
		return exitWithCode(fmt.Errorf("loading model: %w", err), 2)
	}

	known := make(map[string]bool)
	for _, e := range model.Validate(m) {
		known[e.Path+"|"+e.Message] = true
	}

	changes, err := apply(m)
	if err != nil {
		return exitWithCode(err, 1)
	}

	introduced := 0
	for _, e := range model.Validate(m) {
		if !known[e.Path+"|"+e.Message] {
			introduced++
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "ERROR: [%s] %s\n", e.Path, e.Message)
		}
	}
	if introduced > 0 {
		return exitWithCode(fmt.Errorf("the change would make the model invalid (%d new error(s)); nothing was saved", introduced), 1)
	}

	if !dryRun {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "note: model rewritten, comments and formatting were not preserved")
		if err := model.Save(modelPath, m); err != nil {
			return exitWithCode(fmt.Errorf("saving model: %w", err), 2)
		}
	}
	return printEditResult(cmd.OutOrStdout(), format, dryRun, changes)
}

func printEditResult(w io.Writer, format string, dryRun bool, changes []model.Change) error {
	if changes == nil {
		changes = []model.Change{}
	}
	if format == "json" {
		data, err := json.MarshalIndent(editResult{DryRun: dryRun, Changes: changes}, "", "  ")
		if err != nil {
			return fmt.Errorf("encoding result: %w", err)
		}
		_, err = fmt.Fprintln(w, string(data))
		return err
	}
	if dryRun {
		_, _ = fmt.Fprintf(w, "Dry run: %d change(s), nothing saved\n", len(changes))
	} else {
		_, _ = fmt.Fprintf(w, "%d change(s) saved\n", len(changes))
	}
	for _, c := range changes {
		switch {
		case c.To != "":
			_, _ = fmt.Fprintf(w, "  %s: %s -> %s\n", c.Path, c.From, c.To)
		default:
			_, _ = fmt.Fprintf(w, "  removed %s: %s\n", c.Path, c.From)
		}
	}
	return nil
}
