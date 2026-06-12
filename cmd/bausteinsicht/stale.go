package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/docToolchain/Bausteinsicht/internal/stale"
	"github.com/spf13/cobra"
)

func isDrawioFile(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".drawio")
}

func newStaleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stale",
		Short: "Detect unused or forgotten architecture elements",
		Long: `Detect elements that have not been referenced in git commits for a
configurable period and have no lifecycle status or ADR link.

These are likely forgotten components — either undocumented active services
or candidates for archiving.

Example:
  bausteinsicht stale --model architecture.jsonc --days 90
  bausteinsicht stale --format json --model architecture.jsonc`,
		RunE: runStale,
	}

	cmd.Flags().StringP("model", "m", "architecture.jsonc", "Path to architecture model file")
	cmd.Flags().IntP("days", "d", 90, "Consider elements stale if not modified in this many days")
	cmd.Flags().StringP("format", "f", "text", "Output format: text or json")
	cmd.Flags().Bool("mark-drawio", false, "Mark stale elements in draw.io diagram")
	cmd.Flags().Bool("unmark-drawio", false, "Remove stale markers from draw.io diagram")
	cmd.Flags().String("drawio-file", "", "Path to draw.io diagram (auto-detected if empty)")

	return cmd
}

func runStale(cmd *cobra.Command, _ []string) error {
	modelPath, _ := cmd.Flags().GetString("model")
	days, _ := cmd.Flags().GetInt("days")
	format, _ := cmd.Flags().GetString("format")
	markDrawio, _ := cmd.Flags().GetBool("mark-drawio")
	unmarkDrawio, _ := cmd.Flags().GetBool("unmark-drawio")
	drawioFile, _ := cmd.Flags().GetString("drawio-file")

	if err := validatePathContainment(modelPath); err != nil {
		return exitWithCode(fmt.Errorf("model path: %w", err), 1)
	}
	if days < 0 {
		return exitWithCode(fmt.Errorf("--days must be non-negative"), 1)
	}
	if format != "text" && format != "json" {
		return exitWithCode(fmt.Errorf("invalid format %q: must be text or json", format), 1)
	}

	m, err := model.Load(modelPath)
	if err != nil {
		return exitWithCode(fmt.Errorf("loading model: %w", err), 1)
	}
	absModelPath, err := filepath.Abs(modelPath)
	if err != nil {
		return exitWithCode(fmt.Errorf("resolving model path: %w", err), 1)
	}

	config := stale.LoadConfigFromModel(m)
	if cmd.Flags().Changed("days") {
		config.ThresholdDays = days
	} else if config.ThresholdDays < 0 {
		return exitWithCode(fmt.Errorf("meta.staleDetection.thresholdDays must be non-negative, got %d", config.ThresholdDays), 1)
	}

	if unmarkDrawio {
		return applyDrawioUnmarking(absModelPath, drawioFile, cmd.ErrOrStderr())
	}

	result, err := stale.Detect(m, absModelPath, config)
	if err != nil {
		return exitWithCode(fmt.Errorf("detection failed: %w", err), 1)
	}

	if err := printStaleResult(result, format, cmd.OutOrStdout()); err != nil {
		return err
	}

	if markDrawio && len(result.StaleElements) > 0 {
		return applyDrawioMarking(result.StaleElements, absModelPath, drawioFile, cmd.ErrOrStderr())
	}

	return nil
}

// printStaleResult writes the detection result to w in the requested format.
func printStaleResult(result stale.DetectionResult, format string, w io.Writer) error {
	switch format {
	case "json":
		output, err := stale.FormatJSON(result)
		if err != nil {
			return exitWithCode(fmt.Errorf("formatting JSON: %w", err), 1)
		}
		_, err = fmt.Fprint(w, output)
		return err
	default: // "text"
		_, err := fmt.Fprint(w, stale.FormatText(result))
		return err
	}
}

// findDrawioFile returns the resolved draw.io file path. If drawioFile is empty,
// auto-detects by scanning the model's directory for the first .drawio file.
func findDrawioFile(absModelPath, drawioFile string) string {
	if drawioFile != "" {
		return drawioFile
	}
	entries, err := os.ReadDir(filepath.Dir(absModelPath))
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if !entry.IsDir() && isDrawioFile(entry.Name()) {
			return filepath.Join(filepath.Dir(absModelPath), entry.Name())
		}
	}
	return ""
}

// applyDrawioUnmarking removes stale markers from the draw.io file.
func applyDrawioUnmarking(absModelPath, explicitDrawioFile string, stderr io.Writer) error {
	if explicitDrawioFile != "" {
		if err := validatePathContainment(explicitDrawioFile); err != nil {
			return exitWithCode(fmt.Errorf("drawio-file path: %w", err), 1)
		}
	}
	drawioFile := findDrawioFile(absModelPath, explicitDrawioFile)
	if drawioFile == "" {
		return nil
	}
	if _, err := os.Stat(drawioFile); err != nil {
		if explicitDrawioFile != "" {
			return exitWithCode(fmt.Errorf("drawio-file not found: %w", err), 1)
		}
		return nil
	}
	count, err := stale.UnmarkInDrawio(drawioFile)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Warning: Failed to unmark draw.io: %v\n", err)
		return err
	}
	if count > 0 {
		_, _ = fmt.Fprintf(stderr, "Removed stale markers from %d element(s) in %s\n", count, filepath.Base(drawioFile))
	}
	return nil
}

// applyDrawioMarking marks stale elements in the draw.io file, writing status to stderr.
func applyDrawioMarking(elements []stale.StaleElement, absModelPath, explicitDrawioFile string, stderr io.Writer) error {
	if explicitDrawioFile != "" {
		if err := validatePathContainment(explicitDrawioFile); err != nil {
			return exitWithCode(fmt.Errorf("drawio-file path: %w", err), 1)
		}
	}
	drawioFile := findDrawioFile(absModelPath, explicitDrawioFile)
	if drawioFile == "" {
		return nil
	}
	if _, err := os.Stat(drawioFile); err != nil {
		if explicitDrawioFile != "" {
			return exitWithCode(fmt.Errorf("drawio-file not found: %w", err), 1)
		}
		return nil // silently skip auto-detected path that no longer exists
	}
	if markErr := stale.MarkInDrawio(elements, drawioFile); markErr != nil {
		_, _ = fmt.Fprintf(stderr, "Warning: Failed to mark draw.io: %v\n", markErr)
		return markErr
	}
	_, _ = fmt.Fprintf(stderr, "Marked %d stale elements in %s\n", len(elements), filepath.Base(drawioFile))
	return nil
}
