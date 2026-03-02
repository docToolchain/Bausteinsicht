package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/docToolchain/Bauteinsicht/internal/drawio"
	"github.com/docToolchain/Bauteinsicht/internal/model"
	bsync "github.com/docToolchain/Bauteinsicht/internal/sync"
	"github.com/docToolchain/Bauteinsicht/internal/watcher"
	"github.com/docToolchain/Bauteinsicht/templates"
	"github.com/spf13/cobra"
)

func newWatchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "watch",
		Short: "Watch model and diagram for changes and auto-sync",
		Long:  "Watches the model and draw.io files for changes and automatically runs a sync cycle on each change.",
		RunE:  runWatch,
	}
}

func runWatch(cmd *cobra.Command, _ []string) error {
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

	// Derive drawio path from model path.
	dir := filepath.Dir(modelPath)
	drawioPath := filepath.Join(dir, "architecture.drawio")

	// Verify both files exist before starting the watcher.
	if _, err := os.Stat(modelPath); err != nil {
		return exitWithCode(fmt.Errorf("model file not found: %w", err), 2)
	}
	if _, err := os.Stat(drawioPath); err != nil {
		return exitWithCode(fmt.Errorf("draw.io file not found: %w", err), 2)
	}

	absModel, _ := filepath.Abs(modelPath)
	absDrawio, _ := filepath.Abs(drawioPath)

	var err error

	fmt.Printf("Watching %s and %s...\n", modelPath, drawioPath)

	// Create the file watcher. Use a variable so the callback can access the watcher.
	var w *watcher.Watcher
	w, err = watcher.New(
		[]string{absModel, absDrawio},
		watcher.DefaultDebounce,
		func(changedFile string) {
			w.SetSyncing(true)
			defer w.SetSyncing(false)
			doSync(changedFile, modelPath, drawioPath, templatePath)
		},
	)
	if err != nil {
		return exitWithCode(fmt.Errorf("creating watcher: %w", err), 2)
	}

	if err := w.Start(); err != nil {
		return exitWithCode(fmt.Errorf("starting watcher: %w", err), 2)
	}

	// Block until SIGINT/SIGTERM.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	w.Stop()
	fmt.Println("Stopped watching.")
	return nil
}

func doSync(changedFile, modelPath, drawioPath, templatePath string) {
	fmt.Printf("[%s] Sync triggered by %s\n", time.Now().Format("15:04:05"), changedFile)

	dir := filepath.Dir(modelPath)
	statePath := filepath.Join(dir, ".bausteinsicht-sync")

	m, err := model.Load(modelPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR loading model: %v\n", err)
		return
	}

	// Load draw.io document. If the file is an empty mxfile (no diagram
	// pages — e.g., after all views were removed), recreate it and reset
	// sync state (#175).
	var watchRecreated bool
	doc, err := drawio.LoadDocument(drawioPath)
	if err != nil {
		if isEmptyMxfileError(err) {
			fmt.Fprintln(os.Stderr, "WARNING: Draw.io file has no diagram pages, recreating structure")
			doc = drawio.NewDocument()
			watchRecreated = true
		} else {
			fmt.Fprintf(os.Stderr, "ERROR loading draw.io file: %v\n", err)
			return
		}
	}

	var state *bsync.SyncState
	if watchRecreated {
		_ = os.Remove(statePath)
		state = &bsync.SyncState{
			Elements:      make(map[string]bsync.ElementState),
			Relationships: []bsync.RelationshipState{},
		}
	} else {
		state, err = bsync.LoadState(statePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR loading sync state: %v\n", err)
			return
		}
	}

	var tmpl *drawio.TemplateSet
	if templatePath != "" {
		tmpl, err = drawio.LoadTemplate(templatePath)
	} else {
		tmpl, err = drawio.LoadTemplateFromBytes(templates.DefaultTemplate)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR loading template: %v\n", err)
		return
	}

	// Ensure pages exist for all views; track new pages (#184, #188, #189).
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

	result := bsync.Run(m, doc, state, tmpl, newPageIDs)

	if err := saveModel(modelPath, m, result); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR saving model: %v\n", err)
		return
	}
	if err := drawio.SaveDocument(drawioPath, doc); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR saving draw.io file: %v\n", err)
		return
	}

	absModel, _ := filepath.Abs(modelPath)
	absDrawio, _ := filepath.Abs(drawioPath)
	newState, err := bsync.BuildState(m, doc, absModel, absDrawio)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR building sync state: %v\n", err)
		return
	}
	if err := bsync.SaveState(statePath, newState); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR saving sync state: %v\n", err)
		return
	}

	for _, w := range result.Warnings {
		fmt.Fprintln(os.Stderr, "WARNING:", w)
	}

	printSyncSummary(buildSyncSummary(result))
}
