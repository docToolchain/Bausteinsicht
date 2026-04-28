package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/docToolchain/Bausteinsicht/internal/search"
	"github.com/spf13/cobra"
)

func newFindCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "find <query>",
		Short: "Search elements, relationships, and views by free-text query",
		Long: `Search all model objects (elements, relationships, views) for the given query.

All words in a multi-word query must match (AND semantics). Matching is
case-insensitive and partial (e.g. "pay" matches "payment-service").

Results are ranked by relevance score. Use --format json for LLM workflows.`,
		Args:          cobra.MinimumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runFind,
	}
	cmd.Flags().String("type", "all", "Limit results to: element, relationship, view, all")
	return cmd
}

func runFind(cmd *cobra.Command, args []string) error {
	query := strings.Join(args, " ")
	format, _ := cmd.Flags().GetString("format")
	modelPath, _ := cmd.Flags().GetString("model")
	typeFlag, _ := cmd.Flags().GetString("type")

	if modelPath == "" {
		detected, err := model.AutoDetect(".")
		if err != nil {
			return exitWithCode(err, 2)
		}
		modelPath = detected
	}

	m, err := model.Load(modelPath)
	if err != nil {
		return exitWithCode(err, 2)
	}

	opts := search.Options{}
	switch typeFlag {
	case "element":
		opts.Type = search.ResultElement
	case "relationship":
		opts.Type = search.ResultRelationship
	case "view":
		opts.Type = search.ResultView
	case "all", "":
		// no filter
	default:
		return exitWithCode(fmt.Errorf("unknown --type %q: use element, relationship, view, or all", typeFlag), 2)
	}

	resp := search.Run(query, m, opts)

	if format == "json" {
		data, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return err
	}

	return printFindText(cmd, resp)
}

func printFindText(cmd *cobra.Command, resp search.Response) error {
	out := cmd.OutOrStdout()
	if resp.Total == 0 {
		_, err := fmt.Fprintf(out, "No results for %q.\n", resp.Query)
		return err
	}

	header := fmt.Sprintf("Search results for %q (%d match", resp.Query, resp.Total)
	if resp.Total != 1 {
		header += "es"
	}
	header += ")"
	if _, err := fmt.Fprintln(out, header); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, strings.Repeat("=", len(header))); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out); err != nil {
		return err
	}

	// Group by type for display.
	var elements, relationships, views []search.Result
	for _, r := range resp.Results {
		switch r.Type {
		case search.ResultElement:
			elements = append(elements, r)
		case search.ResultRelationship:
			relationships = append(relationships, r)
		case search.ResultView:
			views = append(views, r)
		}
	}

	if len(elements) > 0 {
		if _, err := fmt.Fprintf(out, "Elements (%d):\n", len(elements)); err != nil {
			return err
		}
		for _, r := range elements {
			extra := ""
			if r.Technology != "" {
				extra = "  technology: " + r.Technology
			} else if r.Status != "" {
				extra = "  status: " + r.Status
			}
			if _, err := fmt.Fprintf(out, "  %-28s [%-10s]  %-35s%s  score: %d\n",
				r.ID, r.Kind, fmt.Sprintf("%q", r.Title), extra, r.Score); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(out); err != nil {
			return err
		}
	}

	if len(relationships) > 0 {
		if _, err := fmt.Fprintf(out, "Relationships (%d):\n", len(relationships)); err != nil {
			return err
		}
		for _, r := range relationships {
			label := ""
			if r.Title != "" {
				label = fmt.Sprintf("%q", r.Title)
			}
			if _, err := fmt.Fprintf(out, "  %-28s  %s → %s  %s  score: %d\n",
				r.ID, r.From, r.To, label, r.Score); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(out); err != nil {
			return err
		}
	}

	if len(views) > 0 {
		if _, err := fmt.Fprintf(out, "Views (%d):\n", len(views)); err != nil {
			return err
		}
		for _, r := range views {
			if _, err := fmt.Fprintf(out, "  %-28s  %-35s  score: %d\n",
				r.ID, fmt.Sprintf("%q", r.Title), r.Score); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(out); err != nil {
			return err
		}
	}

	return nil
}
