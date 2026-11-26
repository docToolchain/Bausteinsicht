package main

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/spf13/cobra"
)

func newAddFromPatternCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-from-pattern <pattern-id>",
		Short: "Add elements and relationships from a pattern",
		Long:  "Expand a pattern from the specification into concrete elements and relationships in the model.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAddFromPattern(cmd, args)
		},
	}

	cmd.Flags().String("id", "", "Base ID for generated elements (required)")
	cmd.Flags().String("title", "", "Base title for generated elements (default: --id)")
	cmd.Flags().String("prefix", "", "Namespace prefix (modifies {base} to prefix-base)")
	_ = cmd.MarkFlagRequired("id")

	return cmd
}

func newListPatternsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all available patterns",
		Long:  "List all patterns defined in specification with their element and relationship counts.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListPatterns(cmd)
		},
	}
}

func runAddFromPattern(cmd *cobra.Command, args []string) error {
	patternID := args[0]
	modelPath, _ := cmd.Flags().GetString("model")
	baseID, _ := cmd.Flags().GetString("id")
	title, _ := cmd.Flags().GetString("title")
	prefix, _ := cmd.Flags().GetString("prefix")

	// Apply prefix if provided
	if prefix != "" {
		baseID = prefix + "-" + baseID
	}
	// The same ID rule as "add element" (#703).
	if !isValidKey(baseID) {
		return exitWithCode(
			fmt.Errorf("invalid element ID %q: must start with a letter and contain only letters, digits, hyphens, or underscores", baseID),
			1,
		)
	}

	if modelPath == "" {
		detected, err := model.AutoDetect(".")
		if err != nil {
			return exitWithCode(err, 2)
		}
		modelPath = detected
	}

	unlock, err := lockModel(modelPath)
	if err != nil {
		return exitWithCode(err, 2)
	}
	defer unlock()

	m, err := model.Load(modelPath)
	if err != nil {
		return exitWithCode(fmt.Errorf("loading model: %w", err), 2)
	}

	// Errors the model has before the pattern is applied do not count against it.
	known := make(map[string]bool)
	for _, e := range model.Validate(m) {
		known[e.Path+"|"+e.Message] = true
	}

	// Check if pattern exists
	pattern, exists := m.Specification.Patterns[patternID]
	if !exists {
		return exitWithCode(fmt.Errorf("pattern %q not found in specification", patternID), 2)
	}

	// Check for conflicts
	conflicts, err := model.CheckPatternConflicts(m, pattern, baseID)
	if err != nil {
		return exitWithCode(err, 2)
	}

	if len(conflicts) > 0 {
		return exitWithCode(fmt.Errorf("conflict: elements already exist: %v (use a different --id)", conflicts), 2)
	}

	// Expand the pattern
	elements, relationships, err := model.ExpandPattern(pattern, baseID, title)
	if err != nil {
		return exitWithCode(err, 2)
	}

	// The top-level elements, in the order of the expanded elements (#702).
	topIDs := model.ExpandPatternTopLevelIDs(pattern, baseID)
	for _, id := range topIDs {
		if !isValidKey(id) {
			return exitWithCode(fmt.Errorf("the pattern %q derives the invalid element ID %q from --id %q: IDs must start with a letter and contain only letters, digits, hyphens, or underscores", patternID, id, baseID), 1)
		}
	}
	_, relIDs, err := model.ExpandPatternIDs(pattern, baseID)
	if err != nil {
		return exitWithCode(err, 2)
	}

	// Add elements to model (at top level for now)
	if m.Model == nil {
		m.Model = make(map[string]model.Element)
	}
	for i, elem := range elements {
		m.Model[topIDs[i]] = elem
	}

	// Add relationships (From and To are already expanded by ExpandPattern)
	m.Relationships = append(m.Relationships, relationships...)

	// Refuse a pattern that would leave the model invalid (an unknown kind, a
	// relationship to an element that does not exist, an ID that breaks the rules),
	// like "add element" and "add relationship" do; nothing is saved (#703).
	introduced := 0
	for _, e := range model.Validate(m) {
		if !known[e.Path+"|"+e.Message] {
			introduced++
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "ERROR: [%s] %s\n", e.Path, e.Message)
		}
	}
	if introduced > 0 {
		return exitWithCode(fmt.Errorf("the pattern %q would make the model invalid (%d new error(s)); nothing was saved", patternID, introduced), 1)
	}

	// Save the updated model
	if err := model.Save(modelPath, m); err != nil {
		return exitWithCode(fmt.Errorf("saving model: %w", err), 2)
	}

	// Output summary: every created element with its full ID, children below
	// their parent.
	out := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(out, "✅ Pattern '%s' applied with base ID '%s':\n", patternID, baseID)
	for i, id := range topIDs {
		printPatternElement(out, id, elements[i])
	}
	for i, id := range relIDs {
		_, _ = fmt.Fprintf(out, "   + %-20s %s → %s  \"%s\"\n", id, relationships[i].From, relationships[i].To, relationships[i].Label)
	}

	return nil
}

// printPatternElement prints an element created from a pattern and, indented by
// its depth in the ID, its children.
func printPatternElement(out io.Writer, id string, el model.Element) {
	_, _ = fmt.Fprintf(out, "   + %-20s [%-10s] \"%s\"\n", id, el.Kind, el.Title)
	childKeys := make([]string, 0, len(el.Children))
	for k := range el.Children {
		childKeys = append(childKeys, k)
	}
	sort.Strings(childKeys)
	for _, k := range childKeys {
		printPatternElement(out, id+"."+k, el.Children[k])
	}
}

func runListPatterns(cmd *cobra.Command) error {
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
		return exitWithCode(fmt.Errorf("loading model: %w", err), 2)
	}

	if len(m.Specification.Patterns) == 0 {
		fmt.Println("No patterns defined in specification")
		return nil
	}

	// Sort pattern IDs
	var patternIDs []string
	for id := range m.Specification.Patterns {
		patternIDs = append(patternIDs, id)
	}
	sort.Strings(patternIDs)

	format, _ := cmd.Flags().GetString("format")
	if format == "json" {
		// Output as JSON
		type patternInfo struct {
			ID                string `json:"id"`
			Description       string `json:"description"`
			ElementCount      int    `json:"elementCount"`
			RelationshipCount int    `json:"relationshipCount"`
		}
		var patterns []patternInfo
		for _, id := range patternIDs {
			p := m.Specification.Patterns[id]
			patterns = append(patterns, patternInfo{
				ID:                id,
				Description:       p.Description,
				ElementCount:      len(p.Elements),
				RelationshipCount: len(p.Relationships),
			})
		}
		b, err := json.MarshalIndent(patterns, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	}

	// Text output
	fmt.Println("Available patterns:")
	fmt.Println("──────────────────────────────────────────────────────────────")
	for _, id := range patternIDs {
		p := m.Specification.Patterns[id]
		elemCount := len(p.Elements)
		relCount := len(p.Relationships)
		fmt.Printf("  %-25s %s (%d elements, %d relationships)\n",
			id, p.Description, elemCount, relCount)
	}

	return nil
}
