package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/docToolchain/Bausteinsicht/internal/snapshot"
	"github.com/spf13/cobra"
)

func newSnapshotDiffCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff <snapshot-id-1> [snapshot-id-2]",
		Short: "Diff two snapshots or a snapshot vs current state",
		Long:  "Compare two snapshots or compare a snapshot against the current model state.",
		Args:  cobra.RangeArgs(1, 2),
		RunE:  runSnapshotDiff,
	}

	cmd.Flags().String("format", "text", "Output format: text or json")

	return cmd
}

func runSnapshotDiff(cmd *cobra.Command, args []string) error {
	snapshotID1 := args[0]
	format, _ := cmd.Flags().GetString("format")
	modelPath, _ := cmd.Flags().GetString("model")

	manager := snapshot.NewManager(".")

	// Load first snapshot
	if !manager.Exists(snapshotID1) {
		return exitWithCode(fmt.Errorf("snapshot not found: %s", snapshotID1), 2)
	}

	snap1, err := manager.Load(snapshotID1)
	if err != nil {
		return exitWithCode(fmt.Errorf("loading snapshot %s: %w", snapshotID1, err), 2)
	}

	var model2 *model.BausteinsichtModel

	// Load second snapshot or current state
	if len(args) == 2 {
		snapshotID2 := args[1]
		if !manager.Exists(snapshotID2) {
			return exitWithCode(fmt.Errorf("snapshot not found: %s", snapshotID2), 2)
		}
		snap2, err := manager.Load(snapshotID2)
		if err != nil {
			return exitWithCode(fmt.Errorf("loading snapshot %s: %w", snapshotID2, err), 2)
		}
		model2 = snap2.Model
	} else {
		// Load current model
		if modelPath == "" {
			modelPath = "architecture.jsonc"
		}
		m, err := model.Load(modelPath)
		if err != nil {
			return exitWithCode(fmt.Errorf("loading current model: %w", err), 2)
		}
		model2 = m
	}

	// Compare models
	diffs := diffModels(snap1.Model, model2)

	// Output results
	switch format {
	case "json":
		data, err := json.MarshalIndent(diffs, "", "  ")
		if err != nil {
			return exitWithCode(fmt.Errorf("marshaling diff: %w", err), 2)
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))
	case "text":
		_, _ = fmt.Fprint(cmd.OutOrStdout(), formatDiffText(diffs))
	default:
		return exitWithCode(fmt.Errorf("unknown format: %s", format), 2)
	}

	return nil
}

type ModelDiff struct {
	AddedElements        []string            `json:"addedElements,omitempty"`
	RemovedElements      []string            `json:"removedElements,omitempty"`
	ChangedElements      map[string][]string `json:"changedElements,omitempty"`
	AddedRelationships   []RelDiff           `json:"addedRelationships,omitempty"`
	RemovedRelationships []RelDiff           `json:"removedRelationships,omitempty"`
}

type RelDiff struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Label string `json:"label,omitempty"`
}

func diffModels(m1, m2 *model.BausteinsichtModel) *ModelDiff {
	result := &ModelDiff{
		ChangedElements: make(map[string][]string),
	}

	// Flatten elements for easier comparison
	flat1 := flattenAll(m1.Model)
	flat2 := flattenAll(m2.Model)

	// Find added and removed elements
	for id := range flat2 {
		if _, exists := flat1[id]; !exists {
			result.AddedElements = append(result.AddedElements, id)
		}
	}

	for id := range flat1 {
		if _, exists := flat2[id]; !exists {
			result.RemovedElements = append(result.RemovedElements, id)
		}
	}

	// Find changed elements
	for id, elem1 := range flat1 {
		if elem2, exists := flat2[id]; exists {
			changes := compareElements(elem1, elem2)
			if len(changes) > 0 {
				result.ChangedElements[id] = changes
			}
		}
	}

	// Compare relationships
	relMap1 := relationshipMapString(m1.Relationships)
	relMap2 := relationshipMapString(m2.Relationships)

	for key, rel2 := range relMap2 {
		if _, exists := relMap1[key]; !exists {
			result.AddedRelationships = append(result.AddedRelationships, rel2)
		}
	}

	for key, rel1 := range relMap1 {
		if _, exists := relMap2[key]; !exists {
			result.RemovedRelationships = append(result.RemovedRelationships, rel1)
		}
	}

	return result
}

func flattenAll(elems map[string]model.Element) map[string]model.Element {
	result := make(map[string]model.Element)
	for key, elem := range elems {
		result[key] = elem
		if len(elem.Children) > 0 {
			children := flattenAll(elem.Children)
			for k, v := range children {
				result[key+"."+k] = v
			}
		}
	}
	return result
}

func compareElements(e1, e2 model.Element) []string {
	var changes []string
	if e1.Title != e2.Title {
		changes = append(changes, fmt.Sprintf("title: %q → %q", e1.Title, e2.Title))
	}
	if e1.Kind != e2.Kind {
		changes = append(changes, fmt.Sprintf("kind: %q → %q", e1.Kind, e2.Kind))
	}
	if e1.Description != e2.Description {
		changes = append(changes, fmt.Sprintf("description: %q → %q", e1.Description, e2.Description))
	}
	if e1.Technology != e2.Technology {
		changes = append(changes, fmt.Sprintf("technology: %q → %q", e1.Technology, e2.Technology))
	}
	if e1.Status != e2.Status {
		changes = append(changes, fmt.Sprintf("status: %q → %q", e1.Status, e2.Status))
	}
	return changes
}

func relationshipMapString(rels []model.Relationship) map[string]RelDiff {
	m := make(map[string]RelDiff)
	for _, rel := range rels {
		key := fmt.Sprintf("%s:%s", rel.From, rel.To)
		m[key] = RelDiff{From: rel.From, To: rel.To, Label: rel.Label}
	}
	return m
}

func formatDiffText(diffs *ModelDiff) string {
	var sb strings.Builder

	totalChanges := len(diffs.AddedElements) + len(diffs.RemovedElements) +
		len(diffs.ChangedElements) + len(diffs.AddedRelationships) +
		len(diffs.RemovedRelationships)

	if totalChanges == 0 {
		return "No differences found.\n"
	}

	fmt.Fprintf(&sb, "Architecture Differences (Total Changes: %d)\n", totalChanges)
	sb.WriteString(strings.Repeat("=", 50))
	sb.WriteString("\n\n")

	if len(diffs.AddedElements) > 0 {
		fmt.Fprintf(&sb, "Added Elements (%d):\n", len(diffs.AddedElements))
		for _, id := range diffs.AddedElements {
			fmt.Fprintf(&sb, "  + %s\n", id)
		}
		sb.WriteString("\n")
	}

	if len(diffs.RemovedElements) > 0 {
		sb.WriteString(fmt.Sprintf("Removed Elements (%d):\n", len(diffs.RemovedElements)))
		for _, id := range diffs.RemovedElements {
			sb.WriteString(fmt.Sprintf("  - %s\n", id))
		}
		sb.WriteString("\n")
	}

	if len(diffs.ChangedElements) > 0 {
		sb.WriteString(fmt.Sprintf("Changed Elements (%d):\n", len(diffs.ChangedElements)))
		for id, changes := range diffs.ChangedElements {
			sb.WriteString(fmt.Sprintf("  ~ %s\n", id))
			for _, change := range changes {
				sb.WriteString(fmt.Sprintf("      %s\n", change))
			}
		}
		sb.WriteString("\n")
	}

	if len(diffs.AddedRelationships) > 0 {
		sb.WriteString(fmt.Sprintf("Added Relationships (%d):\n", len(diffs.AddedRelationships)))
		for _, rel := range diffs.AddedRelationships {
			sb.WriteString(fmt.Sprintf("  + %s → %s", rel.From, rel.To))
			if rel.Label != "" {
				sb.WriteString(fmt.Sprintf(" (%s)", rel.Label))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	if len(diffs.RemovedRelationships) > 0 {
		sb.WriteString(fmt.Sprintf("Removed Relationships (%d):\n", len(diffs.RemovedRelationships)))
		for _, rel := range diffs.RemovedRelationships {
			sb.WriteString(fmt.Sprintf("  - %s → %s", rel.From, rel.To))
			if rel.Label != "" {
				sb.WriteString(fmt.Sprintf(" (%s)", rel.Label))
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
