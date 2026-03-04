package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/docToolchain/Bauteinsicht/internal/drawio"
	"github.com/docToolchain/Bauteinsicht/internal/model"
	"github.com/docToolchain/Bauteinsicht/internal/sync"
	"github.com/docToolchain/Bauteinsicht/templates"
	"github.com/spf13/cobra"
)

const (
	defaultModelFile  = "architecture.jsonc"
	defaultDrawioFile = "architecture.drawio"
	defaultTemplFile  = "template.drawio"
	defaultSyncState  = ".bausteinsicht-sync"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize a new architecture project",
		Long:  "Creates a sample model, template, and initial draw.io diagram in the current directory.",
		RunE:  runInit,
	}
}

func runInit(cmd *cobra.Command, _ []string) error {
	format, _ := cmd.Flags().GetString("format")

	// Check if files already exist.
	for _, name := range []string{defaultModelFile, defaultDrawioFile, defaultTemplFile} {
		if _, err := os.Stat(name); err == nil {
			return exitWithCode(
				fmt.Errorf("file %q already exists; remove it or use a different directory", name),
				2,
			)
		}
	}

	// Write sample model.
	if err := os.WriteFile(defaultModelFile, templates.SampleModel, 0600); err != nil {
		return exitWithCode(fmt.Errorf("writing %s: %w", defaultModelFile, err), 2)
	}

	// Write template.
	if err := os.WriteFile(defaultTemplFile, templates.DefaultTemplate, 0600); err != nil {
		return exitWithCode(fmt.Errorf("writing %s: %w", defaultTemplFile, err), 2)
	}

	// Load model for sync.
	m, err := model.Load(defaultModelFile)
	if err != nil {
		return exitWithCode(fmt.Errorf("loading model: %w", err), 2)
	}

	// Load template.
	tmpl, err := drawio.LoadTemplateFromBytes(templates.DefaultTemplate)
	if err != nil {
		return exitWithCode(fmt.Errorf("loading template: %w", err), 2)
	}

	// Create empty document and run initial forward sync.
	doc := drawio.NewDocument()
	emptyState := &sync.SyncState{
		Elements:      make(map[string]sync.ElementState),
		Relationships: []sync.RelationshipState{},
	}

	// Add pages for each view before sync.
	for viewID, view := range m.Views {
		doc.AddPage("view-"+viewID, view.Title)
	}

	_ = sync.Run(m, doc, emptyState, tmpl, nil)

	// Save generated draw.io file.
	if err := drawio.SaveDocument(defaultDrawioFile, doc); err != nil {
		return exitWithCode(fmt.Errorf("writing %s: %w", defaultDrawioFile, err), 2)
	}

	// Build and save sync state.
	modelPath, _ := filepath.Abs(defaultModelFile)
	drawioPath, _ := filepath.Abs(defaultDrawioFile)
	state, err := sync.BuildState(m, doc, modelPath, drawioPath)
	if err != nil {
		return exitWithCode(fmt.Errorf("building sync state: %w", err), 2)
	}
	if err := sync.SaveState(defaultSyncState, state); err != nil {
		return exitWithCode(fmt.Errorf("writing %s: %w", defaultSyncState, err), 2)
	}

	// Output result.
	createdFiles := []string{defaultModelFile, defaultTemplFile, defaultDrawioFile, defaultSyncState}

	if format == "json" {
		out := map[string]interface{}{
			"success": true,
			"files":   createdFiles,
		}
		data, _ := json.Marshal(out)
		fmt.Println(string(data))
	} else {
		fmt.Println("Initialized Bausteinsicht project:")
		for _, f := range createdFiles {
			fmt.Printf("  - %s\n", f)
		}
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Println("  1. Edit architecture.jsonc to define your architecture")
		fmt.Println("  2. Run 'bausteinsicht sync' to update the draw.io diagram")
		fmt.Println("  3. Open architecture.drawio in draw.io to arrange elements")
	}

	return nil
}
