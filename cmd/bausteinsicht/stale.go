package main

import (
	"fmt"
	"path/filepath"

	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/docToolchain/Bausteinsicht/internal/stale"
	"github.com/spf13/cobra"
)

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
	cmd.Flags().Bool("mark-drawio", false, "Mark stale elements in draw.io diagram (TODO)")

	return cmd
}

func runStale(cmd *cobra.Command, _ []string) error {
	modelPath, _ := cmd.Flags().GetString("model")
	days, _ := cmd.Flags().GetInt("days")
	format, _ := cmd.Flags().GetString("format")
	markDrawio, _ := cmd.Flags().GetBool("mark-drawio")

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

	// TODO: Implement --mark-drawio functionality
	if markDrawio {
		if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Note: --mark-drawio is not yet implemented\n"); err != nil {
			return err
		}
	}

	return nil
}
