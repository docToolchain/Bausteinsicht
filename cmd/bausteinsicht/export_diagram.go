package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/docToolchain/Bausteinsicht/internal/diagram"
	"github.com/docToolchain/Bausteinsicht/internal/export"
	dslexport "github.com/docToolchain/Bausteinsicht/internal/exporter/structurizr"
	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/spf13/cobra"
)

// exportJSONEntry is the JSON shape for one exported view or sequence.
// Exactly one of Source or Path is set per entry:
//   - Source (*string, omitempty): pointer so the field is present even when the
//     diagram renders to an empty string (source-mode). nil in path-mode → omitted.
//   - Path (string, omitempty): absolute path to the written file, present in path-mode.
type exportJSONEntry struct {
	View   string  `json:"view"`
	Format string  `json:"format"`
	Source *string `json:"source,omitempty"`
	Path   string  `json:"path,omitempty"`
}

// emitExportJSON marshals entries and writes them to cmd's stdout.
// Returns a plain error; callers wrap with exitWithCode as needed.
func emitExportJSON(cmd *cobra.Command, entries []exportJSONEntry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling JSON output: %w", err)
	}
	if _, err := cmd.OutOrStdout().Write(append(data, '\n')); err != nil {
		return fmt.Errorf("writing JSON output: %w", err)
	}
	return nil
}

// writeExportFile creates the parent directory if needed, writes content to
// outPath, and returns its absolute path. The absolute path is resolved before
// writing so that a failure to resolve (e.g. deleted working directory) does
// not orphan a file on disk.
func writeExportFile(outPath string, content []byte) (string, error) {
	absPath, err := filepath.Abs(outPath)
	if err != nil {
		return "", fmt.Errorf("resolving output path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(absPath), 0750); err != nil {
		return "", fmt.Errorf("creating output directory: %w", err)
	}
	if err := os.WriteFile(absPath, content, 0600); err != nil { //nolint:gosec // output files are non-sensitive documentation
		return "", fmt.Errorf("writing output: %w", err)
	}
	return absPath, nil
}

// buildExportEntry constructs one exportJSONEntry for a rendered view.
// In source-mode (outputDir == "") the diagram text is stored as Source.
// In path-mode the file is written and Source is cleared; Path holds the
// absolute path. Copying content into a local variable before taking its
// address prevents pointer-aliasing if callers ever change from := to =.
func buildExportEntry(viewKey, format, content, outputDir, filename string) (exportJSONEntry, error) {
	src := content // copy so &src is a distinct pointer per call
	entry := exportJSONEntry{View: viewKey, Format: format, Source: &src}
	if outputDir != "" {
		absPath, err := writeExportFile(filepath.Join(outputDir, filename), []byte(content))
		if err != nil {
			return exportJSONEntry{}, err
		}
		entry.Source = nil
		entry.Path = absPath
	}
	return entry, nil
}

func newExportDiagramCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export-diagram",
		Short: "Export views as C4 diagrams (PlantUML, Mermaid, DOT, D2, HTML5, Structurizr DSL)",
		Long:  "Exports architecture views as text-based C4 diagrams (PlantUML, Mermaid, DOT, D2), interactive HTML5 viewer, or Structurizr DSL workspace.",
		RunE:  runExportDiagram,
	}

	cmd.Flags().String("view", "", "Export only this view (by key)")
	cmd.Flags().String("diagram-format", "plantuml", "Diagram format: plantuml, mermaid, dot, d2, html, or structurizr")
	cmd.Flags().String("output", "", "Output directory (default: stdout)")

	return cmd
}

func runExportDiagram(cmd *cobra.Command, _ []string) error {
	modelPath, _ := cmd.Flags().GetString("model")
	viewKey, _ := cmd.Flags().GetString("view")
	diagramFormat, _ := cmd.Flags().GetString("diagram-format")
	outputDir, _ := cmd.Flags().GetString("output")

	if outputDir != "" {
		if err := validatePathContainment(outputDir); err != nil {
			return exitWithCode(fmt.Errorf("--output: %w", err), 2)
		}
	}

	if modelPath == "" {
		detected, err := model.AutoDetect(".")
		if err != nil {
			return exitWithCode(fmt.Errorf("auto-detecting model: %w", err), 2)
		}
		modelPath = detected
	}

	m, err := model.Load(modelPath)
	if err != nil {
		return exitWithCode(fmt.Errorf("loading model: %w", err), 2)
	}

	outputFormat, _ := cmd.Flags().GetString("format")

	// Structurizr DSL export: outputs the whole workspace in one file.
	if diagramFormat == "structurizr" {
		// Structurizr exports the entire workspace, not individual views
		if viewKey != "" {
			return exitWithCode(fmt.Errorf("--view is not supported with structurizr format (exports entire workspace)"), 1)
		}
		dsl := dslexport.Export(m)
		if outputFormat == "json" {
			entry, entryErr := buildExportEntry("workspace", "structurizr", dsl, outputDir, "workspace.dsl")
			if entryErr != nil {
				return exitWithCode(entryErr, 2)
			}
			if err := emitExportJSON(cmd, []exportJSONEntry{entry}); err != nil {
				return exitWithCode(err, 2)
			}
			return nil
		}
		if outputDir == "" {
			_, _ = fmt.Fprint(cmd.OutOrStdout(), dsl)
			return nil
		}
		absPath, writeErr := writeExportFile(filepath.Join(outputDir, "workspace.dsl"), []byte(dsl))
		if writeErr != nil {
			return exitWithCode(writeErr, 2)
		}
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Exported: %s\n", absPath)
		return nil
	}

	// Determine which views to export.
	views := make(map[string]model.View)
	if viewKey != "" {
		v, ok := m.Views[viewKey]
		if !ok {
			return exitWithCode(fmt.Errorf("view %q not found", viewKey), 1)
		}
		views[viewKey] = v
	} else {
		views = m.Views
	}

	// Warn once per view up front, regardless of which format/output branch
	// below ends up rendering it (#512).
	for key, view := range views {
		warnIfEmptyView(cmd, m, key, view)
	}

	// Handle new export formats (DOT, D2, HTML) — with JSON envelope support
	switch diagramFormat {
	case "dot", "d2", "html":
		return handleNewFormats(cmd, m, views, diagramFormat, outputFormat, outputDir, viewKey)
	}

	var f diagram.Format
	var ext string
	switch diagramFormat {
	case "plantuml":
		f = diagram.PlantUML
		ext = "puml"
	case "mermaid":
		f = diagram.Mermaid
		ext = "mmd"
	default:
		return exitWithCode(fmt.Errorf("unknown diagram format %q: valid values are \"plantuml\", \"mermaid\", \"dot\", \"d2\", \"html\", or \"structurizr\"", diagramFormat), 2)
	}

	// When --format json, output structured JSON. (#241, #631)
	// With --output: write files and report absolute "path"; without: report "source".
	if outputFormat == "json" {
		keys := sortedKeys(views)
		entries := make([]exportJSONEntry, 0, len(keys))
		for _, key := range keys {
			result, fmtErr := diagram.FormatView(m, key, f)
			if fmtErr != nil {
				return exitWithCode(fmtErr, 1)
			}
			entry, entryErr := buildExportEntry(key, diagramFormat, result, outputDir, export.SafeViewKey(key)+"."+ext)
			if entryErr != nil {
				return exitWithCode(entryErr, 2)
			}
			entries = append(entries, entry)
		}
		if err := emitExportJSON(cmd, entries); err != nil {
			return exitWithCode(err, 2)
		}
		return nil
	}

	for key := range views {
		result, fmtErr := diagram.FormatView(m, key, f)
		if fmtErr != nil {
			return exitWithCode(fmtErr, 1)
		}

		if outputDir == "" {
			_, _ = fmt.Fprint(cmd.OutOrStdout(), result)
			continue
		}

		if err := os.MkdirAll(outputDir, 0750); err != nil {
			return exitWithCode(fmt.Errorf("creating output directory: %w", err), 2)
		}
		outPath := filepath.Join(outputDir, export.SafeViewKey(key)+"."+ext)
		if err := os.WriteFile(outPath, []byte(result), 0600); err != nil { //nolint:gosec // output files are non-sensitive documentation
			return exitWithCode(fmt.Errorf("writing output: %w", err), 2)
		}
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Exported: %s\n", outPath)
	}

	return nil
}

func handleNewFormats(cmd *cobra.Command, m *model.BausteinsichtModel, views map[string]model.View, diagramFormat, outputFormat, outputDir, viewKey string) error {
	var renderFunc func(*model.BausteinsichtModel, string) (string, error)
	var ext string

	switch diagramFormat {
	case "dot":
		renderFunc = diagram.RenderDOT
		ext = "dot"
	case "d2":
		renderFunc = diagram.RenderD2
		ext = "d2"
	case "html":
		renderFunc = diagram.RenderHTML
		ext = "html"
	default:
		return exitWithCode(fmt.Errorf("unsupported format: %s", diagramFormat), 2)
	}

	// fileNameFor returns the canonical output filename for a view key.
	// Defined once so the JSON and non-JSON paths share the same convention.
	fileNameFor := func(key string) string {
		if diagramFormat == "html" {
			return export.SafeViewKey(key) + ".html"
		}
		return export.OutputFileName(key, ext)
	}

	// When --format json, output structured JSON. (#631)
	// With --output: write files and report absolute "path"; without: report "source".
	// views is pre-filtered by the caller (runExportDiagram passes only the requested
	// view(s)), so viewKey is intentionally not re-checked here.
	if outputFormat == "json" {
		keys := sortedKeys(views)
		entries := make([]exportJSONEntry, 0, len(keys))
		for _, key := range keys {
			result, fmtErr := renderFunc(m, key)
			if fmtErr != nil {
				return exitWithCode(fmtErr, 1)
			}
			entry, entryErr := buildExportEntry(key, diagramFormat, result, outputDir, fileNameFor(key))
			if entryErr != nil {
				return exitWithCode(entryErr, 2)
			}
			entries = append(entries, entry)
		}
		if err := emitExportJSON(cmd, entries); err != nil {
			return exitWithCode(err, 2)
		}
		return nil
	}

	// For HTML, create a single file containing all views
	if diagramFormat == "html" {
		// When exporting to HTML, we need to handle multiple views in a single file
		if viewKey != "" {
			// Single view HTML export
			result, err := renderFunc(m, viewKey)
			if err != nil {
				return exitWithCode(err, 1)
			}

			if outputDir == "" {
				_, _ = fmt.Fprint(cmd.OutOrStdout(), result)
				return nil
			}
			absPath, writeErr := writeExportFile(filepath.Join(outputDir, fileNameFor(viewKey)), []byte(result))
			if writeErr != nil {
				return exitWithCode(writeErr, 2)
			}
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Exported: %s\n", absPath)
			return nil
		}

		// Multiple views: export each as separate file
		keys := sortedKeys(views)
		for _, key := range keys {
			result, err := renderFunc(m, key)
			if err != nil {
				return exitWithCode(err, 1)
			}

			if outputDir == "" {
				_, _ = fmt.Fprint(cmd.OutOrStdout(), result)
				continue
			}

			absPath, writeErr := writeExportFile(filepath.Join(outputDir, fileNameFor(key)), []byte(result))
			if writeErr != nil {
				return exitWithCode(writeErr, 2)
			}
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Exported: %s\n", absPath)
		}
		return nil
	}

	// For DOT, D2 and other formats: export each view separately
	keys := sortedKeys(views)
	for _, key := range keys {
		result, err := renderFunc(m, key)
		if err != nil {
			return exitWithCode(err, 1)
		}

		if outputDir == "" {
			_, _ = fmt.Fprint(cmd.OutOrStdout(), result)
			continue
		}

		absPath, writeErr := writeExportFile(filepath.Join(outputDir, fileNameFor(key)), []byte(result))
		if writeErr != nil {
			return exitWithCode(writeErr, 2)
		}
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Exported: %s\n", absPath)
	}

	return nil
}

// warnIfEmptyView prints a stderr warning when a view resolves to zero
// elements, so an empty exported diagram doesn't silently look like success
// (#512: import produced views with a scope but no include list, and
// export-diagram wrote 0-node files without any indication anything was
// wrong).
func warnIfEmptyView(cmd *cobra.Command, m *model.BausteinsichtModel, key string, view model.View) {
	resolved, err := model.ResolveView(m, &view)
	if err != nil {
		return
	}
	if len(resolved) == 0 {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(),
			"WARNING: view %q resolves to 0 elements — check its scope/include/exclude; the exported diagram will be empty\n", key)
	}
}

func sortedKeys(views map[string]model.View) []string {
	keys := make([]string, 0, len(views))
	for k := range views {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
