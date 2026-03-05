package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	verbose, _ := cmd.Flags().GetBool("verbose")

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

	// Validate model before syncing to catch invalid view include/exclude
	// patterns and other consistency errors. Without this, typos like
	// "customer." (trailing dot) silently remove elements from draw.io. (#176)
	if validationErrs := model.Validate(m); len(validationErrs) > 0 {
		for _, ve := range validationErrs {
			fmt.Fprintln(os.Stderr, "ERROR:", ve)
		}
		return exitWithCode(fmt.Errorf("model validation failed with %d error(s); fix the model before syncing", len(validationErrs)), 1)
	}

	// Load draw.io document. If the file was deleted or is an empty mxfile
	// (no diagram pages — e.g., after all views were removed), recreate it
	// from the template and reset sync state so forward sync repopulates it
	// (#149, #175).
	var recreated bool
	doc, err := drawio.LoadDocument(drawioPath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) && !isEmptyMxfileError(err) {
			return exitWithCode(fmt.Errorf("loading draw.io file: %w", err), 2)
		}
		if errors.Is(err, fs.ErrNotExist) {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "WARNING: Draw.io file not found, recreating from template")
		} else {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "WARNING: Draw.io file has no diagram pages, recreating structure")
		}
		doc = drawio.NewDocument()
		recreated = true
	}

	// Load sync state (empty on first sync).
	// When the draw.io file was recreated, discard any stale state so the
	// sync engine treats all model elements as new.
	var state *bsync.SyncState
	if recreated {
		// Remove stale state file if it exists.
		_ = os.Remove(statePath)
		state = &bsync.SyncState{
			Elements:      make(map[string]bsync.ElementState),
			Relationships: []bsync.RelationshipState{},
		}
	} else {
		state, err = bsync.LoadState(statePath)
		if err != nil {
			return exitWithCode(fmt.Errorf("loading sync state: %w", err), 2)
		}
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

	// Verbose output goes to stderr so it doesn't interfere with JSON on stdout.
	if verbose && format != "json" {
		flat := model.FlattenElements(m)
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Syncing model: %s\n", modelPath)
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  %d elements, %d relationships, %d views\n",
			len(flat), len(m.Relationships), len(m.Views))
	}

	// Ensure pages exist for all views; track which pages are newly created
	// so the sync engine can avoid treating their missing elements as deletions
	// (#184, #188, #189).
	newPageIDs := make(map[string]bool)
	for viewID, view := range m.Views {
		pageID := "view-" + viewID
		if doc.GetPage(pageID) == nil {
			doc.AddPage(pageID, view.Title)
			newPageIDs[pageID] = true
		}
	}

	// Remove orphaned view pages (views deleted or renamed in model). (#143)
	bsync.RemoveOrphanedViewPages(doc, m)

	// Run sync.
	fwdOpts := bsync.ForwardOptions{
		ModelPath: modelPath,
		SyncTime:  time.Now().Format("2006-01-02 15:04"),
	}
	result := bsync.Run(m, doc, state, tmpl, newPageIDs, fwdOpts)

	// Save updated model: use PatchSave to preserve JSONC comments and key
	// ordering when possible, fall back to full Save for structural changes.
	if err := saveModel(modelPath, m, result); err != nil {
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

	// Verbose post-sync details to stderr.
	if verbose && format != "json" {
		if result.Forward != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Forward sync: %d elements created, %d updated, %d deleted; %d connectors created, %d updated, %d deleted\n",
				result.Forward.ElementsCreated, result.Forward.ElementsUpdated, result.Forward.ElementsDeleted,
				result.Forward.ConnectorsCreated, result.Forward.ConnectorsUpdated, result.Forward.ConnectorsDeleted)
		}
		if result.Reverse != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Reverse sync: %d elements created, %d updated, %d deleted; %d relationships created, %d updated, %d deleted\n",
				result.Reverse.ElementsCreated, result.Reverse.ElementsUpdated, result.Reverse.ElementsDeleted,
				result.Reverse.RelationshipsCreated, result.Reverse.RelationshipsUpdated, result.Reverse.RelationshipsDeleted)
		}
		if len(result.Conflicts) > 0 {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Conflicts resolved: %d (model wins)\n", len(result.Conflicts))
		}
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
	ForwardAdded    int `json:"forward_added"`
	ForwardUpdated  int `json:"forward_updated"`
	ForwardDeleted  int `json:"forward_deleted"`
	ReverseAdded    int `json:"reverse_added"`
	ReverseUpdated  int `json:"reverse_updated"`
	ReverseDeleted  int `json:"reverse_deleted"`
	MetadataUpdated int `json:"metadata_updated"`
	Conflicts       int `json:"conflicts"`
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
		s.MetadataUpdated = result.Forward.MetadataUpdated
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

// saveModel saves the model to path, preserving JSONC comments and key ordering
// when the reverse changes are simple field modifications. Falls back to full
// Save for structural changes or when patching fails.
func saveModel(path string, m *model.BausteinsichtModel, result *bsync.SyncResult) error {
	hasReverse := result.Reverse != nil &&
		(result.Reverse.ElementsUpdated+result.Reverse.ElementsCreated+
			result.Reverse.ElementsDeleted+result.Reverse.RelationshipsCreated+
			result.Reverse.RelationshipsUpdated+result.Reverse.RelationshipsDeleted) > 0

	if !hasReverse {
		// No reverse changes — model file doesn't need updating.
		return nil
	}

	if result.Changes != nil {
		ops, patchable := bsync.ReversePatchOps(result.Changes)
		if patchable && len(ops) > 0 {
			if err := model.PatchSave(path, ops); err == nil {
				return nil
			}
			// PatchSave failed — fall through to full Save.
		}
	}

	return model.Save(path, m)
}

// isEmptyMxfileError returns true if the error indicates that the draw.io file
// is a valid XML mxfile but contains no <diagram> elements. This happens when
// all views are removed from the model and sync removes all diagram pages (#175).
func isEmptyMxfileError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "no <diagram> elements")
}

func printSyncSummary(s syncSummary) {
	total := s.ForwardAdded + s.ForwardUpdated + s.ForwardDeleted +
		s.ReverseAdded + s.ReverseUpdated + s.ReverseDeleted
	if total == 0 && s.Conflicts == 0 && s.MetadataUpdated == 0 {
		fmt.Println("Already in sync. No changes.")
		return
	}
	if s.ForwardAdded+s.ForwardUpdated+s.ForwardDeleted > 0 {
		fmt.Printf("Forward (model → draw.io): %d added, %d updated, %d deleted\n",
			s.ForwardAdded, s.ForwardUpdated, s.ForwardDeleted)
	}
	if s.MetadataUpdated > 0 && total == 0 {
		fmt.Printf("Metadata/legend updated on %d view page(s).\n", s.MetadataUpdated/2)
	}
	if s.ReverseAdded+s.ReverseUpdated+s.ReverseDeleted > 0 {
		fmt.Printf("Reverse (draw.io → model): %d added, %d updated, %d deleted\n",
			s.ReverseAdded, s.ReverseUpdated, s.ReverseDeleted)
	}
	if s.Conflicts > 0 {
		fmt.Printf("Conflicts: %d (resolved: model wins)\n", s.Conflicts)
	}
}
