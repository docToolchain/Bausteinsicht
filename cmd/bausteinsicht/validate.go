package main

import (
	"encoding/json"
	"fmt"

	"github.com/docToolchain/Bauteinsicht/internal/model"
	"github.com/spf13/cobra"
)

type validateResult struct {
	Valid  bool              `json:"valid"`
	Errors []validateErrJSON `json:"errors"`
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
		flat := model.FlattenElements(m)
		fmt.Fprintf(cmd.ErrOrStderr(), "Validating model: %s\n", modelPath)
		fmt.Fprintf(cmd.ErrOrStderr(), "  %d elements, %d relationships, %d views\n",
			len(flat), len(m.Relationships), len(m.Views))
	}

	errs := model.Validate(m)

	if format == "json" {
		return outputJSON(cmd, errs)
	}
	return outputText(cmd, errs)
}

func outputJSON(cmd *cobra.Command, errs []model.ValidationError) error {
	result := validateResult{
		Valid:  len(errs) == 0,
		Errors: make([]validateErrJSON, len(errs)),
	}
	for i, e := range errs {
		result.Errors[i] = validateErrJSON{Path: e.Path, Message: e.Message}
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

func outputText(cmd *cobra.Command, errs []model.ValidationError) error {
	if len(errs) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "Model is valid.")
		return err
	}
	for _, e := range errs {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "ERROR: [%s] %s\n", e.Path, e.Message); err != nil {
			return err
		}
	}
	return exitWithCode(fmt.Errorf("validation failed"), 1)
}
