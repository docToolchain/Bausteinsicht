package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/docToolchain/Bauteinsicht/internal/model"
	"github.com/spf13/cobra"
)

func newAddRelationshipCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "relationship",
		Short: "Add a relationship between two elements",
		Long:  "Adds a new relationship to the architecture model. Both --from and --to must reference existing elements.",
		RunE:  runAddRelationship,
	}

	cmd.Flags().String("from", "", "Source element (dot-notation path, e.g. webshop.api)")
	cmd.Flags().String("to", "", "Target element (dot-notation path, e.g. webshop.db)")
	cmd.Flags().String("label", "", "Relationship label")
	cmd.Flags().String("kind", "", "Relationship kind (must be defined in specification)")
	cmd.Flags().String("description", "", "Relationship description")

	_ = cmd.MarkFlagRequired("from")
	_ = cmd.MarkFlagRequired("to")

	return cmd
}

func runAddRelationship(cmd *cobra.Command, args []string) error {
	format, _ := cmd.Flags().GetString("format")
	modelPath, _ := cmd.Flags().GetString("model")
	from, _ := cmd.Flags().GetString("from")
	to, _ := cmd.Flags().GetString("to")
	label, _ := cmd.Flags().GetString("label")
	kind, _ := cmd.Flags().GetString("kind")
	description, _ := cmd.Flags().GetString("description")

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

	if _, err := model.Resolve(m, from); err != nil {
		return exitWithCode(fmt.Errorf("--from: element %q not found", from), 1)
	}

	if _, err := model.Resolve(m, to); err != nil {
		return exitWithCode(fmt.Errorf("--to: element %q not found", to), 1)
	}

	if kind != "" {
		if m.Specification.Relationships == nil {
			return exitWithCode(fmt.Errorf("--kind: %q not defined (no relationship kinds in specification)", kind), 1)
		}
		if _, ok := m.Specification.Relationships[kind]; !ok {
			return exitWithCode(fmt.Errorf("--kind: %q not defined in specification", kind), 1)
		}
	}

	for _, r := range m.Relationships {
		if r.From == from && r.To == to {
			fmt.Fprintf(os.Stderr, "Warning: relationship %s -> %s already exists\n", from, to)
			break
		}
	}

	rel := model.Relationship{
		From:        from,
		To:          to,
		Label:       label,
		Kind:        kind,
		Description: description,
	}
	m.Relationships = append(m.Relationships, rel)

	if err := model.Save(modelPath, m); err != nil {
		return exitWithCode(fmt.Errorf("saving model: %w", err), 2)
	}

	if format == "json" {
		return printRelationshipJSON(rel)
	}
	printRelationshipText(rel)
	return nil
}

func printRelationshipText(r model.Relationship) {
	if r.Label != "" {
		fmt.Printf("Added relationship: %s -> %s (%s)\n", r.From, r.To, r.Label)
	} else {
		fmt.Printf("Added relationship: %s -> %s\n", r.From, r.To)
	}
}

func printRelationshipJSON(r model.Relationship) error {
	out := struct {
		From        string `json:"from"`
		To          string `json:"to"`
		Label       string `json:"label,omitempty"`
		Kind        string `json:"kind,omitempty"`
		Description string `json:"description,omitempty"`
	}{
		From:        r.From,
		To:          r.To,
		Label:       r.Label,
		Kind:        r.Kind,
		Description: r.Description,
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}
