package main

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/spf13/cobra"
)

type statusResult struct {
	Summary  map[string]int `json:"summary"`
	Elements []statusElement `json:"elements"`
}

type statusElement struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	Title string `json:"title"`
	Status string `json:"status"`
}

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show element lifecycle status",
		Long:  "Lists all elements and their lifecycle status (proposed, design, implementation, deployed, deprecated, archived).",
		RunE:  runStatus,
	}

	cmd.Flags().StringP("filter", "f", "", "Filter elements by status (proposed, design, implementation, deployed, deprecated, archived)")
	cmd.Flags().StringP("model", "m", "architecture.jsonc", "Path to architecture model file")

	return cmd
}

func runStatus(cmd *cobra.Command, _ []string) error {
	modelPath, _ := cmd.Flags().GetString("model")
	filter, _ := cmd.Flags().GetString("filter")
	format, _ := cmd.Flags().GetString("format")

	if err := validatePathContainment(modelPath); err != nil {
		return exitWithCode(fmt.Errorf("model path: %w", err), 1)
	}

	m, err := model.Load(modelPath)
	if err != nil {
		return exitWithCode(fmt.Errorf("loading model: %w", err), 1)
	}

	// Validate filter if provided
	if filter != "" {
		valid := false
		for _, status := range model.ValidStatuses {
			if filter == status {
				valid = true
				break
			}
		}
		if !valid {
			return exitWithCode(
				fmt.Errorf("invalid status filter %q; valid values: %v", filter, model.ValidStatuses), 1)
		}
	}

	// Collect all elements with their status
	flatElements, _ := model.FlattenElements(m)

	result := statusResult{
		Summary:  make(map[string]int),
		Elements: []statusElement{},
	}

	// Initialize summary counts
	for _, status := range model.ValidStatuses {
		result.Summary[status] = 0
	}
	result.Summary["unset"] = 0

	// Process elements
	for id, elem := range flatElements {
		status := elem.Status
		if status == "" {
			status = "unset"
		}

		// Apply filter
		if filter != "" && status != filter {
			continue
		}

		// Count
		if status != "unset" {
			result.Summary[status]++
		} else {
			result.Summary["unset"]++
		}

		// Collect element
		result.Elements = append(result.Elements, statusElement{
			ID:    id,
			Kind:  elem.Kind,
			Title: elem.Title,
			Status: status,
		})
	}

	// Sort by status, then by ID
	sort.Slice(result.Elements, func(i, j int) bool {
		if result.Elements[i].Status != result.Elements[j].Status {
			return result.Elements[i].Status < result.Elements[j].Status
		}
		return result.Elements[i].ID < result.Elements[j].ID
	})

	if format == "json" {
		return outputStatusJSON(cmd, result)
	}
	return outputStatusText(cmd, result)
}

func outputStatusJSON(cmd *cobra.Command, result statusResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return err
}

func outputStatusText(cmd *cobra.Command, result statusResult) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Element Lifecycle Status\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "==================================================\n\n"); err != nil {
		return err
	}

	// Print summary
	for _, status := range append(model.ValidStatuses, "unset") {
		count := result.Summary[status]
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s (%d):\n", status, count); err != nil {
			return err
		}

		// Print elements for this status
		for _, elem := range result.Elements {
			if elem.Status == status {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  %-20s [%-12s] %q\n", elem.ID, elem.Kind, elem.Title); err != nil {
					return err
				}
			}
		}
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "\n"); err != nil {
			return err
		}
	}

	return nil
}
