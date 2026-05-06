package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/docToolchain/Bausteinsicht/internal/stale"
	"github.com/spf13/cobra"
)

func isDrawioFile(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".drawio") && strings.Contains(strings.ToLower(filename), "architecture")
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
	cmd.Flags().String("drawio-file", "", "Path to draw.io diagram (auto-detected if empty)")

	return cmd
}

func runStale(cmd *cobra.Command, _ []string) error {
	modelPath, _ := cmd.Flags().GetString("model")
	days, _ := cmd.Flags().GetInt("days")
	format, _ := cmd.Flags().GetString("format")
	markDrawio, _ := cmd.Flags().GetBool("mark-drawio")
	drawioFile, _ := cmd.Flags().GetString("drawio-file")

	// Validate input
	if err := validatePathContainment(modelPath); err != nil {
		return exitWithCode(fmt.Errorf("model path: %w", err), 1)
	}

	if days < 0 {
		return exitWithCode(fmt.Errorf("--days must be non-negative"), 1)
	}

	if format != "text" && format != "json" {
		return exitWithCode(fmt.Errorf("invalid format %q: must be text or json", format), 1)
	}

	// Load model
	m, err := model.Load(modelPath)
	if err != nil {
		return exitWithCode(fmt.Errorf("loading model: %w", err), 1)
	}

	// Get absolute path for git integration
	absModelPath, err := filepath.Abs(modelPath)
	if err != nil {
		return exitWithCode(fmt.Errorf("resolving model path: %w", err), 1)
	}

	// Load configuration from model
	config := stale.LoadConfigFromModel(m)
	config.ThresholdDays = days

	// Run detection
	result, err := stale.Detect(m, absModelPath, config)
	if err != nil {
		return exitWithCode(fmt.Errorf("detection failed: %w", err), 1)
	}

	// Output results
	switch format {
	case "json":
		output, err := stale.FormatJSON(result)
		if err != nil {
			return exitWithCode(fmt.Errorf("formatting JSON: %w", err), 1)
		}
		if _, err := fmt.Fprint(cmd.OutOrStdout(), output); err != nil {
			return err
		}

	case "text":
		output := stale.FormatText(result)
		if _, err := fmt.Fprint(cmd.OutOrStdout(), output); err != nil {
			return err
		}
	}

	// Mark stale elements in draw.io if requested
	if markDrawio && len(result.StaleElements) > 0 {
		// Determine draw.io file path
		if drawioFile == "" {
			// Auto-detect: look for *architecture*.drawio in model directory
			dir := filepath.Dir(absModelPath)
			entries, err := os.ReadDir(dir)
			if err == nil {
				for _, entry := range entries {
					if !entry.IsDir() && isDrawioFile(entry.Name()) {
						drawioFile = filepath.Join(dir, entry.Name())
						break
					}
				}
			}
		}

		// Mark if found
		if drawioFile != "" {
			if _, err := os.Stat(drawioFile); err == nil {
				if err := stale.MarkInDrawio(result.StaleElements, drawioFile); err != nil {
					if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Warning: Failed to mark draw.io: %v\n", err); err != nil {
						return err
					}
				} else {
					if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Marked %d stale elements in %s\n", len(result.StaleElements), filepath.Base(drawioFile)); err != nil {
						return err
					}
				}
			} else if _, ok := err.(*os.PathError); !ok {
				if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Warning: Could not find draw.io file: %v\n", err); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
