package main

import (
	"fmt"
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
		Long: `Computes hierarchical layout for diagram elements and writes positions back to draw.io.
Pinned elements (with bausteinsicht-pinned=true) are preserved by default.`,
		RunE: runLayout,
	}

	cmd.Flags().String("algorithm", "hierarchical", "Layout algorithm: hierarchical (currently only option)")
	cmd.Flags().String("rank-dir", "TB", "Ranking direction: TB (top-to-bottom) or LR (left-to-right)")
	cmd.Flags().Bool("preserve-pinned", true, "Don't move pinned elements (bausteinsicht-pinned=true)")

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

	// Load model
	m, err := model.Load(modelPath)
	if err != nil {
		return exitWithCode(fmt.Errorf("loading model: %w", err), 2)
	}

	// Validate model
	if errs := model.Validate(m); len(errs) > 0 {
		return exitWithCode(fmt.Errorf("model validation failed: %v", errs), 2)
	}

	// Derive draw.io path from model path
	dir := filepath.Dir(modelPath)
	drawioPath := filepath.Join(dir, "architecture.drawio")

	doc, err := drawio.LoadDocument(drawioPath)
	if err != nil {
		return exitWithCode(fmt.Errorf("loading diagram: %w", err), 2)
	}

	rankDir, _ := cmd.Flags().GetString("rank-dir")
	preservePinned, _ := cmd.Flags().GetBool("preserve-pinned")

	// Compute hierarchical layout
	h := layout.NewHierarchicalLayout(m, rankDir)
	result := h.Compute()

	// Apply layout to diagram
	if err := layout.Apply(doc, result, preservePinned); err != nil {
		return exitWithCode(fmt.Errorf("applying layout: %w", err), 2)
	}

	// Save diagram
	if err := drawio.SaveDocument(drawioPath, doc); err != nil {
		return exitWithCode(fmt.Errorf("saving diagram: %w", err), 2)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Layout applied (hierarchical): %s\n", drawioPath)
	return nil
}
