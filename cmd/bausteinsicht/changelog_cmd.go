package main

import (
	"fmt"
	"os"

	"github.com/docToolchain/Bausteinsicht/internal/changelog"
	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/docToolchain/Bausteinsicht/internal/snapshot"
	"github.com/spf13/cobra"
)

func newChangelogCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "changelog",
		Short: "Generate architecture changelog between two points in time",
		Long:  "Compare two versions of the architecture model and generate a human-readable changelog showing what changed.",
		RunE:  runChangelog,
	}

	cmd.Flags().String("model", "", "Model file path (default: auto-detect)")
	cmd.Flags().String("since", "", "Starting git ref or snapshot ID (default: the previous tag)")
	cmd.Flags().String("until", "HEAD", "Ending git ref or snapshot ID")
	// Use a dedicated flag name (not the global --format, which only accepts
	// text|json) so the markdown/asciidoc defaults are not rejected by the
	// root command's global format validation (#425).
	cmd.Flags().String("changelog-format", "markdown", "Output format: markdown, asciidoc, or json")
	cmd.Flags().String("output", "", "Output file path (default: stdout)")

	return cmd
}

func runChangelog(cmd *cobra.Command, _ []string) error {
	modelPath, _ := cmd.Flags().GetString("model")
	since, _ := cmd.Flags().GetString("since")
	until, _ := cmd.Flags().GetString("until")
	format, _ := cmd.Flags().GetString("changelog-format")
	output, _ := cmd.Flags().GetString("output")

	// Validate format
	if format != "markdown" && format != "asciidoc" && format != "json" {
		return exitWithCode(fmt.Errorf("invalid format: %s (expected markdown, asciidoc, or json)", format), 2)
	}

	modelPath, err := resolveModelPath(modelPath)
	if err != nil {
		return err
	}

	snapshots := snapshot.NewManager(".")

	// The default of --since is the previous tag (#698); a snapshot has no tag.
	if since == "" {
		if snapshots.Exists(until) {
			return exitWithCode(fmt.Errorf("--since is required when --until is a snapshot (%s)", until), 2)
		}
		tag, err := changelog.PreviousTag(until)
		if err != nil {
			return exitWithCode(err, 2)
		}
		since = tag
	}

	// Load the models at the two refs: a git ref or a snapshot ID (#728).
	fromModel, fromRef, err := loadChangelogSide(snapshots, modelPath, since)
	if err != nil {
		return exitWithCode(fmt.Errorf("loading model at %q: %w", since, err), 2)
	}
	toModel, toRef, err := loadChangelogSide(snapshots, modelPath, until)
	if err != nil {
		return exitWithCode(fmt.Errorf("loading model at %q: %w", until, err), 2)
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

// loadChangelogSide loads the model at one end of the changelog: the model of the
// snapshot with that ID, or the model at the git ref. The reference carries the
// date of the snapshot or commit for the display.
func loadChangelogSide(snapshots *snapshot.Manager, modelPath, ref string) (*model.BausteinsichtModel, changelog.Reference, error) {
	if snapshots.Exists(ref) {
		snap, err := snapshots.Load(ref)
		if err != nil {
			return nil, changelog.Reference{}, err
		}
		if snap.Model == nil {
			return nil, changelog.Reference{}, fmt.Errorf("snapshot %s contains no model data", ref)
		}
		return snap.Model, changelog.Reference{Ref: ref, Date: snap.Timestamp}, nil
	}
	m, err := changelog.LoadModelAtGitRef(modelPath, ref)
	if err != nil {
		return nil, changelog.Reference{}, err
	}
	r := changelog.Reference{Ref: ref}
	if info, err := changelog.GetCommitInfo(ref); err == nil {
		r.Date = info.Date
	}
	return m, r, nil
}
