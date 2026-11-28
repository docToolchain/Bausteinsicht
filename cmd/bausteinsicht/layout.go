package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/docToolchain/Bausteinsicht/internal/drawio"
	"github.com/docToolchain/Bausteinsicht/internal/layout"
	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/spf13/cobra"
)

func newLayoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "layout",
		Short: "Auto-layout elements in draw.io diagram",
		Long: `Computes a layered layout for every page of the draw.io diagram and writes the positions back.
Each page is laid out on its own, from the elements and connectors on that page; elements keep their size,
elements inside a container or boundary get coordinates relative to it, and the order is the same on every run.
Pinned elements (with bausteinsicht-pinned=true) are preserved by default.`,
		RunE: runLayout,
	}

	cmd.Flags().String("algorithm", "hierarchical", "Layout algorithm: hierarchical (currently only option)")
	cmd.Flags().String("rank-dir", "TB", "Ranking direction: TB (top-to-bottom) or LR (left-to-right)")
	cmd.Flags().Bool("preserve-pinned", true, "Don't move pinned elements (bausteinsicht-pinned=true)")
	cmd.Flags().String("view", "", "Lay out only this view (page); default: every page")

	return cmd
}

func runLayout(cmd *cobra.Command, _ []string) error {
	modelPath, _ := cmd.Flags().GetString("model")
	if modelPath == "" {
		detected, err := model.AutoDetect(".")
		if err != nil {
			return exitWithCode(fmt.Errorf("auto-detecting model: %w", err), 2)
		}
		modelPath = detected
	}

	// The diagram sits next to the model; the layout itself works on the pages
	// of the diagram and does not need the model.
	drawioPath := filepath.Join(filepath.Dir(modelPath), "architecture.drawio")
	if _, statErr := os.Stat(drawioPath); errors.Is(statErr, fs.ErrNotExist) {
		return exitWithCode(errNoDiagram(drawioPath), 2)
	}
	doc, err := drawio.LoadDocument(drawioPath)
	if isEmptyMxfileError(err) {
		return exitWithCode(errNoDiagram(drawioPath), 2)
	}
	if err != nil {
		return exitWithCode(fmt.Errorf("loading diagram: %w", err), 2)
	}

	algorithm, _ := cmd.Flags().GetString("algorithm")
	rankDir, _ := cmd.Flags().GetString("rank-dir")
	preservePinned, _ := cmd.Flags().GetBool("preserve-pinned")
	view, _ := cmd.Flags().GetString("view")

	report, err := layout.LayoutDocument(doc, layout.Options{Algorithm: algorithm, RankDir: rankDir, PreservePinned: preservePinned, View: view})
	if err != nil {
		return exitWithCode(fmt.Errorf("applying layout: %w", err), 1)
	}

	if err := drawio.SaveDocument(drawioPath, doc); err != nil {
		return exitWithCode(fmt.Errorf("saving diagram: %w", err), 2)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Layout applied (hierarchical): %s\n", drawioPath)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %d page(s), %d element(s) positioned, %d pinned element(s) left in place\n",
		report.Pages, report.Elements, report.Pinned)
	return nil
}

// errNoDiagram is the error for a diagram that does not exist yet or has no pages
// (sync creates it from the views of the model; a model without views has none).
func errNoDiagram(path string) error {
	return fmt.Errorf("no diagram yet (%s): run 'bausteinsicht sync' first; a model without views has no diagram, add one with 'bausteinsicht add view'", path)
}
