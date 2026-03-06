package main

import (
	"encoding/json"
	"fmt"

	"github.com/docToolchain/Bauteinsicht/internal/model"
	"github.com/spf13/cobra"
)

type validateResult struct {
	Valid    bool              `json:"valid"`
	Errors   []validateErrJSON `json:"errors"`
	Warnings []validateErrJSON `json:"warnings,omitempty"`
}

type validateErrJSON struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

func newValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "validate",
		Short:         "Validate the architecture model",
		Long:          "Validates the architecture model file for consistency and reports any errors.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runValidate,
	}
}

func runValidate(cmd *cobra.Command, args []string) error {
	format, _ := cmd.Flags().GetString("format")
	modelPath, _ := cmd.Flags().GetString("model")
	verbose, _ := cmd.Flags().GetBool("verbose")

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

	// Verbose output goes to stderr so it doesn't interfere with JSON on stdout.
	if verbose && format != "json" {
		flat, _ := model.FlattenElements(m)
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Validating model: %s\n", modelPath)
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  %d elements, %d relationships, %d views\n",
			len(flat), len(m.Relationships), len(m.Views))
	}

	result := model.ValidateWithWarnings(m)

	if format == "json" {
		return outputJSON(cmd, result)
	}
	return outputText(cmd, result)
}

func outputJSON(cmd *cobra.Command, vr model.ValidationResult) error {
	result := validateResult{
		Valid:  len(vr.Errors) == 0,
		Errors: make([]validateErrJSON, len(vr.Errors)),
	}
	for i, e := range vr.Errors {
		result.Errors[i] = validateErrJSON{Path: e.Path, Message: e.Message}
	}
	if len(vr.Warnings) > 0 {
		result.Warnings = make([]validateErrJSON, len(vr.Warnings))
		for i, w := range vr.Warnings {
			result.Warnings[i] = validateErrJSON{Path: w.Path, Message: w.Message}
		}
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), string(data)); err != nil {
		return err
	}
	if !result.Valid {
		return exitWithCode(fmt.Errorf("validation failed"), 1)
	}
	return nil
}

func outputText(cmd *cobra.Command, vr model.ValidationResult) error {
	for _, w := range vr.Warnings {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "WARNING: [%s] %s\n", w.Path, w.Message); err != nil {
			return err
		}
	}
	if len(vr.Errors) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "Model is valid.")
		return err
	}
	for _, e := range vr.Errors {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "ERROR: [%s] %s\n", e.Path, e.Message); err != nil {
			return err
		}
	}
	return exitWithCode(fmt.Errorf("validation failed"), 1)
}
