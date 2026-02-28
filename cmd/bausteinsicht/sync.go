package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/docToolchain/Bauteinsicht/internal/drawio"
	"github.com/docToolchain/Bauteinsicht/internal/model"
	bsync "github.com/docToolchain/Bauteinsicht/internal/sync"
	"github.com/docToolchain/Bauteinsicht/templates"
	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Synchronize model and draw.io diagram",
		Long:  "Runs one bidirectional sync cycle between the architecture model and the draw.io diagram.",
		RunE:  runSync,
	}
}

func runSync(cmd *cobra.Command, _ []string) error {
	format, _ := cmd.Flags().GetString("format")
	modelPath, _ := cmd.Flags().GetString("model")
	templatePath, _ := cmd.Flags().GetString("template")

	// Auto-detect model file.
	if modelPath == "" {
		detected, err := model.AutoDetect(".")
		if err != nil {
			return exitWithCode(fmt.Errorf("auto-detecting model: %w", err), 2)
		}
		modelPath = detected
	}

	// Derive drawio and state paths from model path.
	dir := filepath.Dir(modelPath)
	drawioPath := filepath.Join(dir, "architecture.drawio")
	statePath := filepath.Join(dir, ".bausteinsicht-sync")

	// Load model.
	m, err := model.Load(modelPath)
	if err != nil {
		return exitWithCode(fmt.Errorf("loading model: %w", err), 2)
	}

	// Load draw.io document.
	doc, err := drawio.LoadDocument(drawioPath)
	if err != nil {
		return exitWithCode(fmt.Errorf("loading draw.io file: %w", err), 2)
	}

	// Load sync state (empty on first sync).
	state, err := bsync.LoadState(statePath)
	if err != nil {
		return exitWithCode(fmt.Errorf("loading sync state: %w", err), 2)
	}

	// Load template.
	var tmpl *drawio.TemplateSet
	if templatePath != "" {
		tmpl, err = drawio.LoadTemplate(templatePath)
	} else {
		tmpl, err = drawio.LoadTemplateFromBytes(templates.DefaultTemplate)
	}
	if err != nil {
		return exitWithCode(fmt.Errorf("loading template: %w", err), 2)
	}

	// Ensure pages exist for all views.
	for viewID, view := range m.Views {
		pageID := "view-" + viewID
		if doc.GetPage(pageID) == nil {
			doc.AddPage(pageID, view.Title)
		}
	}

	// Run sync.
	result := bsync.Run(m, doc, state, tmpl)

	// Save updated files.
	if err := model.Save(modelPath, m); err != nil {
		return exitWithCode(fmt.Errorf("saving model: %w", err), 2)
	}
	if err := drawio.SaveDocument(drawioPath, doc); err != nil {
		return exitWithCode(fmt.Errorf("saving draw.io file: %w", err), 2)
	}

	absModel, _ := filepath.Abs(modelPath)
	absDrawio, _ := filepath.Abs(drawioPath)
	newState, err := bsync.BuildState(m, doc, absModel, absDrawio)
	if err != nil {
		return exitWithCode(fmt.Errorf("building sync state: %w", err), 2)
	}
	if err := bsync.SaveState(statePath, newState); err != nil {
		return exitWithCode(fmt.Errorf("saving sync state: %w", err), 2)
	}

	// Print warnings to stderr.
	for _, w := range result.Warnings {
		fmt.Fprintln(os.Stderr, "WARNING:", w)
	}

	// Output summary.
	summary := buildSyncSummary(result)
	if format == "json" {
		data, _ := json.MarshalIndent(summary, "", "  ")
		fmt.Println(string(data))
	} else {
		printSyncSummary(summary)
	}

	// Exit code 1 if conflicts detected.
	if len(result.Conflicts) > 0 {
		return exitWithCode(fmt.Errorf("%d conflict(s) resolved (model wins)", len(result.Conflicts)), 1)
	}

	return nil
}

type syncSummary struct {
	ForwardAdded   int `json:"forward_added"`
	ForwardUpdated int `json:"forward_updated"`
	ForwardDeleted int `json:"forward_deleted"`
	ReverseAdded   int `json:"reverse_added"`
	ReverseUpdated int `json:"reverse_updated"`
	ReverseDeleted int `json:"reverse_deleted"`
	Conflicts      int `json:"conflicts"`
}

func buildSyncSummary(result *bsync.SyncResult) syncSummary {
	s := syncSummary{
		Conflicts: len(result.Conflicts),
	}
	if result.Forward != nil {
		s.ForwardAdded = result.Forward.ElementsCreated
		s.ForwardUpdated = result.Forward.ElementsUpdated
		s.ForwardDeleted = result.Forward.ElementsDeleted
		s.ForwardAdded += result.Forward.ConnectorsCreated
		s.ForwardUpdated += result.Forward.ConnectorsUpdated
		s.ForwardDeleted += result.Forward.ConnectorsDeleted
	}
	if result.Reverse != nil {
		s.ReverseAdded = result.Reverse.ElementsCreated
		s.ReverseUpdated = result.Reverse.ElementsUpdated
		s.ReverseDeleted = result.Reverse.ElementsDeleted
		s.ReverseAdded += result.Reverse.RelationshipsCreated
		s.ReverseUpdated += result.Reverse.RelationshipsUpdated
		s.ReverseDeleted += result.Reverse.RelationshipsDeleted
	}
	return s
}

func printSyncSummary(s syncSummary) {
	total := s.ForwardAdded + s.ForwardUpdated + s.ForwardDeleted +
		s.ReverseAdded + s.ReverseUpdated + s.ReverseDeleted
	if total == 0 && s.Conflicts == 0 {
		fmt.Println("Already in sync. No changes.")
		return
	}
	if s.ForwardAdded+s.ForwardUpdated+s.ForwardDeleted > 0 {
		fmt.Printf("Forward (model → draw.io): %d added, %d updated, %d deleted\n",
			s.ForwardAdded, s.ForwardUpdated, s.ForwardDeleted)
	}
	if s.ReverseAdded+s.ReverseUpdated+s.ReverseDeleted > 0 {
		fmt.Printf("Reverse (draw.io → model): %d added, %d updated, %d deleted\n",
			s.ReverseAdded, s.ReverseUpdated, s.ReverseDeleted)
	}
	if s.Conflicts > 0 {
		fmt.Printf("Conflicts: %d (resolved: model wins)\n", s.Conflicts)
	}
}
