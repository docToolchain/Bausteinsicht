package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/spf13/cobra"
)

func newShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <element-id>",
		Short: "Show full details of a model element",
		Long: `Display all fields, relationships, and views for a single element.

The element ID uses dot-notation for nested elements (e.g. "system.backend.api").
Use 'bausteinsicht find <query>' to discover element IDs.`,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runShow,
	}
}

type showRelEntry struct {
	Direction string
	Other     string
	Label     string
	Kind      string
}

type showRelJSON struct {
	Direction string `json:"direction"`
	Other     string `json:"other"`
	Label     string `json:"label,omitempty"`
	Kind      string `json:"kind,omitempty"`
}

type showJSONOutput struct {
	ID          string            `json:"id"`
	Kind        string            `json:"kind"`
	Title       string            `json:"title,omitempty"`
	Description string            `json:"description,omitempty"`
	Technology  string            `json:"technology,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Rels        []showRelJSON     `json:"relationships"`
	Views       []string          `json:"views"`
}

func runShow(cmd *cobra.Command, args []string) error {
	elementID := args[0]
	format, _ := cmd.Flags().GetString("format")
	modelPath, _ := cmd.Flags().GetString("model")

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

	flat, err := model.FlattenElements(m)
	if err != nil {
		return exitWithCode(err, 2)
	}

	elem, ok := flat[elementID]
	if !ok {
		return exitWithCode(fmt.Errorf("element %q not found", elementID), 1)
	}

	var rels []showRelEntry
	for _, rel := range m.Relationships {
		if rel.From == elementID {
			rels = append(rels, showRelEntry{"→", rel.To, rel.Label, rel.Kind})
		} else if rel.To == elementID {
			rels = append(rels, showRelEntry{"←", rel.From, rel.Label, rel.Kind})
		}
	}
	sort.Slice(rels, func(i, j int) bool {
		return rels[i].Direction+rels[i].Other < rels[j].Direction+rels[j].Other
	})

	var viewKeys []string
	for key, view := range m.Views {
		for _, inc := range view.Include {
			prefix := strings.TrimSuffix(inc, ".*")
			if inc == elementID || (strings.HasSuffix(inc, ".*") && strings.HasPrefix(elementID, prefix+".")) {
				viewKeys = append(viewKeys, key)
				break
			}
		}
	}
	sort.Strings(viewKeys)

	if format == "json" {
		return printShowJSON(cmd, elementID, elem, rels, viewKeys)
	}
	return printShowText(cmd, elementID, elem, rels, viewKeys)
}

func printShowJSON(cmd *cobra.Command, id string, elem *model.Element, rels []showRelEntry, viewKeys []string) error {
	out := showJSONOutput{
		ID:          id,
		Kind:        elem.Kind,
		Title:       elem.Title,
		Description: elem.Description,
		Technology:  elem.Technology,
		Tags:        elem.Tags,
		Metadata:    elem.Metadata,
		Views:       viewKeys,
		Rels:        []showRelJSON{},
	}
	if out.Views == nil {
		out.Views = []string{}
	}
	for _, r := range rels {
		out.Rels = append(out.Rels, showRelJSON(r))
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return err
}

func printShowText(cmd *cobra.Command, id string, elem *model.Element, rels []showRelEntry, viewKeys []string) error {
	o := cmd.OutOrStdout()
	header := "Element: " + id
	if _, err := fmt.Fprintln(o, header); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(o, strings.Repeat("=", len(header))); err != nil {
		return err
	}

	printField := func(label, value string) error {
		if value == "" {
			return nil
		}
		_, err := fmt.Fprintf(o, "%-14s %s\n", label+":", value)
		return err
	}

	if err := printField("Kind", elem.Kind); err != nil {
		return err
	}
	if err := printField("Title", elem.Title); err != nil {
		return err
	}
	if err := printField("Description", elem.Description); err != nil {
		return err
	}
	if err := printField("Technology", elem.Technology); err != nil {
		return err
	}
	if len(elem.Tags) > 0 {
		if err := printField("Tags", "["+strings.Join(elem.Tags, ", ")+"]"); err != nil {
			return err
		}
	}
	for k, v := range elem.Metadata {
		if err := printField(k, v); err != nil {
			return err
		}
	}

	if len(rels) > 0 {
		if _, err := fmt.Fprintln(o, "\nRelationships:"); err != nil {
			return err
		}
		for _, r := range rels {
			label := ""
			if r.Label != "" {
				label = fmt.Sprintf("  %q", r.Label)
			}
			kind := ""
			if r.Kind != "" {
				kind = "  [" + r.Kind + "]"
			}
			if _, err := fmt.Fprintf(o, "  %s %-30s%s%s\n", r.Direction, r.Other, label, kind); err != nil {
				return err
			}
		}
	}

	if len(viewKeys) > 0 {
		if _, err := fmt.Fprintf(o, "\nViews: %s\n", strings.Join(viewKeys, ", ")); err != nil {
			return err
		}
	}

	return nil
}
