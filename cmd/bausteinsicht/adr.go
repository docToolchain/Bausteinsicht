package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/spf13/cobra"
)

func newADRCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "adr",
		Short: "Manage Architecture Decision Records (ADRs)",
		Long:  "List, show, and manage architecture decision records linked to model elements.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newADRListCmd())
	cmd.AddCommand(newADRShowCmd())

	return cmd
}

func newADRListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all ADRs or ADRs linked to an element",
		Long:  "List architecture decision records, optionally filtered by element.",
		RunE: func(cmd *cobra.Command, args []string) error {
			modelPath := cmd.Flag("model").Value.String()
			elementID := cmd.Flag("element").Value.String()
			format := cmd.Flag("format").Value.String()

			if modelPath == "" {
				modelPath = "architecture.jsonc"
			}

			m, err := model.Load(modelPath)
			if err != nil {
				return fmt.Errorf("loading model: %w", err)
			}

			// Collect decisions to display
			var decisions []model.DecisionRecord
			if elementID != "" {
				// Filter decisions for a specific element
				elem, ok := findElementByID(m, elementID)
				if !ok || elem == nil {
					return fmt.Errorf("element not found: %s", elementID)
				}

				for _, decisionID := range elem.Decisions {
					for _, d := range m.Specification.Decisions {
						if d.ID == decisionID {
							decisions = append(decisions, d)
							break
						}
					}
				}
			} else {
				// All decisions
				decisions = m.Specification.Decisions
			}

			// Sort by ID
			sort.Slice(decisions, func(i, j int) bool {
				return decisions[i].ID < decisions[j].ID
			})

			// Format output
			if format == "json" {
				b, err := json.MarshalIndent(decisions, "", "  ")
				if err != nil {
					return fmt.Errorf("marshaling JSON: %w", err)
				}
				fmt.Println(string(b))
				return nil
			}

			// Default text format
			if len(decisions) == 0 {
				if elementID != "" {
					fmt.Printf("No decisions linked to element %q\n", elementID)
				} else {
					fmt.Println("No decisions defined")
				}
				return nil
			}

			fmt.Printf("Decisions (%d):\n", len(decisions))
			fmt.Println("──────────────────────────────────────────")
			for _, d := range decisions {
				statusIcon := getStatusIcon(d.Status)
				fmt.Printf("%-20s %s %s\n", d.ID, statusIcon, d.Title)
				if d.Date != "" {
					fmt.Printf("  Date: %s\n", d.Date)
				}
				if d.FilePath != "" {
					fmt.Printf("  File: %s\n", d.FilePath)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringP("model", "m", "", "Path to architecture model (default: architecture.jsonc)")
	cmd.Flags().String("element", "", "Filter decisions linked to this element")
	cmd.Flags().String("format", "text", "Output format: text or json")

	return cmd
}

func newADRShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <adr-id>",
		Short: "Show details of a specific ADR",
		Long:  "Display detailed information about an architecture decision record.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			modelPath := cmd.Flag("model").Value.String()
			decisionID := args[0]

			if modelPath == "" {
				modelPath = "architecture.jsonc"
			}

			m, err := model.Load(modelPath)
			if err != nil {
				return fmt.Errorf("loading model: %w", err)
			}

			// Find the decision
			var decision *model.DecisionRecord
			for i, d := range m.Specification.Decisions {
				if d.ID == decisionID {
					decision = &m.Specification.Decisions[i]
					break
				}
			}

			if decision == nil {
				return fmt.Errorf("decision not found: %s", decisionID)
			}

			// Collect elements and relationships that reference this decision
			var references []string
			flat, _ := model.FlattenElements(m)
			for elemID, elem := range flat {
				for _, dID := range elem.Decisions {
					if dID == decisionID {
						references = append(references, "element: "+elemID)
						break
					}
				}
			}
			for _, rel := range m.Relationships {
				for _, dID := range rel.Decisions {
					if dID == decisionID {
						references = append(references, fmt.Sprintf("relationship: %s → %s", rel.From, rel.To))
						break
					}
				}
			}

			// Display information
			statusIcon := getStatusIcon(decision.Status)
			fmt.Printf("ADR: %s %s\n", decision.ID, statusIcon)
			fmt.Println("──────────────────────────────────────────")
			fmt.Printf("Title:  %s\n", decision.Title)
			fmt.Printf("Status: %s\n", decision.Status)
			if decision.Date != "" {
				fmt.Printf("Date:   %s\n", decision.Date)
			}
			if decision.FilePath != "" {
				fmt.Printf("File:   %s\n", decision.FilePath)
			}

			if len(references) > 0 {
				sort.Strings(references)
				fmt.Println("\nReferenced by:")
				for _, ref := range references {
					fmt.Printf("  - %s\n", ref)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringP("model", "m", "", "Path to architecture model (default: architecture.jsonc)")

	return cmd
}

func getStatusIcon(status model.ADRStatus) string {
	switch status {
	case model.ADRActive:
		return "✓"
	case model.ADRProposed:
		return "◯"
	case model.ADRDeprecated:
		return "⚠"
	case model.ADRSuperseded:
		return "✗"
	default:
		return "?"
	}
}

func findElementByID(m *model.BausteinsichtModel, id string) (*model.Element, bool) {
	parts := strings.Split(id, ".")
	if len(parts) == 0 {
		return nil, false
	}

	elem, ok := m.Model[parts[0]]
	if !ok {
		return nil, false
	}

	// Navigate through child elements
	current := &elem
	for _, part := range parts[1:] {
		child, ok := current.Children[part]
		if !ok {
			return nil, false
		}
		current = &child
	}

	return current, true
}
