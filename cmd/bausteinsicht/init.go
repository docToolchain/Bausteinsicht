package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/docToolchain/Bausteinsicht/internal/drawio"
	"github.com/docToolchain/Bausteinsicht/internal/model"
	bsync "github.com/docToolchain/Bausteinsicht/internal/sync"
	"github.com/docToolchain/Bausteinsicht/internal/template"
	"github.com/docToolchain/Bausteinsicht/templates"
	"github.com/spf13/cobra"
)

const (
	defaultModelFile  = "architecture.jsonc"
	defaultDrawioFile = "architecture.drawio"
	defaultTemplFile  = "template.drawio"
	defaultSyncState  = ".bausteinsicht-sync"
)

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new architecture project",
		Long:  "Creates a sample model, template, and initial draw.io diagram in the current directory.",
		RunE:  runInit,
	}
	cmd.Flags().Bool("generate-template", false, "Generate template from spec instead of using default")
	return cmd
}

func runInit(cmd *cobra.Command, _ []string) error {
	format, _ := cmd.Flags().GetString("format")
	generateTemplate, _ := cmd.Flags().GetBool("generate-template")

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

	// Load model for sync.
	m, err := model.Load(defaultModelFile)
	if err != nil {
		return exitWithCode(fmt.Errorf("loading model: %w", err), 2)
	}

	// Generate or use default template.
	var templateBytes []byte
	if generateTemplate {
		gen := template.NewGenerator(m.Specification, "default")
		templateXML := gen.Generate()
		templateBytes = []byte(templateXML)
	} else {
		templateBytes = templates.DefaultTemplate
	}

	// Write template.
	if err := os.WriteFile(defaultTemplFile, templateBytes, 0600); err != nil {
		return exitWithCode(fmt.Errorf("writing %s: %w", defaultTemplFile, err), 2)
	}

	// Load template.
	tmpl, err := drawio.LoadTemplateFromBytes(templateBytes)
	if err != nil {
		return exitWithCode(fmt.Errorf("loading template: %w", err), 2)
	}

	// Create empty document and run initial forward sync.
	doc := drawio.NewDocument()
	emptyState := &bsync.SyncState{
		Elements:      make(map[string]bsync.ElementState),
		Relationships: []bsync.RelationshipState{},
	}

	// Add pages for each view before sync.
	for viewID, view := range m.Views {
		doc.AddPage("view-"+viewID, view.Title)
	}

	// Pass ForwardOptions so metadata/legend boxes are created during init,
	// preventing the first sync from reporting metadata changes (#265).
	// Use the same relative model path that sync would use.
	fwdOpts := bsync.ForwardOptions{
		ModelPath: defaultModelFile,
		SyncTime:  time.Now().Format("2006-01-02 15:04"),
	}
	_ = bsync.Run(m, doc, emptyState, tmpl, nil, fwdOpts)

	// Save generated draw.io file.
	if err := drawio.SaveDocument(defaultDrawioFile, doc); err != nil {
		return exitWithCode(fmt.Errorf("writing %s: %w", defaultDrawioFile, err), 2)
	}

	// Build and save sync state.
	absModel, _ := filepath.Abs(defaultModelFile)
	absDrawio, _ := filepath.Abs(defaultDrawioFile)
	state, err := bsync.BuildState(m, doc, absModel, absDrawio)
	if err != nil {
		return exitWithCode(fmt.Errorf("building sync state: %w", err), 2)
	}
	if err := bsync.SaveState(defaultSyncState, state); err != nil {
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
