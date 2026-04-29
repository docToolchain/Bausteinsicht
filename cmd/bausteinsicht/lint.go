package main

import (
	"encoding/json"
	"fmt"

	"github.com/docToolchain/Bausteinsicht/internal/constraints"
	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/spf13/cobra"
)

func newLintCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "lint",
		Short:         "Check architecture constraints",
		Long:          "Evaluates all constraints defined in the model and reports violations.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runLint,
	}
}

func runLint(cmd *cobra.Command, _ []string) error {
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

	if len(m.Constraints) == 0 {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "No constraints defined."); err != nil {
			return err
		}
		return nil
	}

	result := constraints.Evaluate(m)

	if format == "json" {
		return lintOutputJSON(cmd, result)
	}
	return lintOutputText(cmd, result)
}

func lintOutputJSON(cmd *cobra.Command, r constraints.Result) error {
	type jsonResult struct {
		Passed     bool                 `json:"passed"`
		Total      int                  `json:"total"`
		Violations []constraints.Violation `json:"violations"`
	}

	out := jsonResult{
		Passed:     r.Total == 0,
		Total:      r.Total,
		Violations: r.Violations,
	}
	if out.Violations == nil {
		out.Violations = []constraints.Violation{}
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), string(data)); err != nil {
		return err
	}
	if r.Total > 0 {
		return exitWithCode(fmt.Errorf("lint: %d violation(s) found", r.Total), 1)
	}
	return nil
}

func lintOutputText(cmd *cobra.Command, r constraints.Result) error {
	if r.Total == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "All constraints passed.")
		return err
	}

	for _, v := range r.Violations {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "VIOLATION [%s]: %s\n", v.ConstraintID, v.Message); err != nil {
			return err
		}
		for _, el := range v.Elements {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", el); err != nil {
				return err
			}
		}
	}

	return exitWithCode(fmt.Errorf("lint: %d violation(s) found", r.Total), 1)
}
