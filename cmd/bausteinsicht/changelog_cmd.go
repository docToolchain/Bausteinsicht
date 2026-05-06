package main

import (
	"fmt"
	"os"

	"github.com/docToolchain/Bausteinsicht/internal/changelog"
	"github.com/spf13/cobra"
)

func newChangelogCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "changelog",
		Short: "Generate architecture changelog between two points in time",
		Long:  "Compare two versions of the architecture model and generate a human-readable changelog showing what changed.",
		RunE:  runChangelog,
	}

	cmd.Flags().String("model", "architecture.jsonc", "Model file path")
	cmd.Flags().String("since", "", "Starting git ref or snapshot ID (default: previous tag)")
	cmd.Flags().String("until", "HEAD", "Ending git ref or snapshot ID")
	cmd.Flags().String("format", "markdown", "Output format: markdown, asciidoc, or json")
	cmd.Flags().String("output", "", "Output file path (default: stdout)")

	return cmd
}

func runChangelog(cmd *cobra.Command, _ []string) error {
	modelPath, _ := cmd.Flags().GetString("model")
	since, _ := cmd.Flags().GetString("since")
	until, _ := cmd.Flags().GetString("until")
	format, _ := cmd.Flags().GetString("format")
	output, _ := cmd.Flags().GetString("output")

	if err := validatePathContainment(modelPath); err != nil {
		return exitWithCode(err, 2)
	}

	// Validate format
	if format != "markdown" && format != "asciidoc" && format != "json" {
		return exitWithCode(fmt.Errorf("invalid format: %s (expected markdown, asciidoc, or json)", format), 2)
	}

	// Load models at the two refs
	fromModel, err := changelog.LoadModelAtGitRef(modelPath, since)
	if err != nil {
		return exitWithCode(fmt.Errorf("loading model at %q: %w", since, err), 2)
	}

	toModel, err := changelog.LoadModelAtGitRef(modelPath, until)
	if err != nil {
		return exitWithCode(fmt.Errorf("loading model at %q: %w", until, err), 2)
	}

	// Get reference info for display
	fromRef := changelog.Reference{Ref: since}
	toRef := changelog.Reference{Ref: until}

	if fromInfo, err := changelog.GetCommitInfo(since); err == nil {
		fromRef.Date = fromInfo.Date
	}
	if toInfo, err := changelog.GetCommitInfo(until); err == nil {
		toRef.Date = toInfo.Date
	}

	// Generate changelog
	cl := changelog.Generate(fromModel, toModel, fromRef, toRef)

	// Render output
	var result string
	switch format {
	case "markdown":
		result = changelog.RenderMarkdown(cl)
	case "asciidoc":
		result = changelog.RenderAsciiDoc(cl)
	case "json":
		var err error
		result, err = changelog.RenderJSON(cl)
		if err != nil {
			return exitWithCode(fmt.Errorf("rendering JSON: %w", err), 2)
		}
	}

	// Write output
	if output == "" {
		// Write to stdout
		if _, err := fmt.Fprint(cmd.OutOrStdout(), result); err != nil {
			return exitWithCode(fmt.Errorf("writing output: %w", err), 2)
		}
	} else {
		// Write to file
		if err := os.WriteFile(output, []byte(result), 0o644); err != nil {
			return exitWithCode(fmt.Errorf("writing to %q: %w", output, err), 2)
		}
	}

	return nil
}
