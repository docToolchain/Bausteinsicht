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
	fmt.Fprintln(cmd.OutOrStdout(), string(data))
	if !result.Valid {
		return exitWithCode(fmt.Errorf("validation failed"), 1)
	}
	return nil
}

func outputText(cmd *cobra.Command, errs []model.ValidationError) error {
	if len(errs) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "Model is valid.")
		return nil
	}
	for _, e := range errs {
		fmt.Fprintf(cmd.OutOrStdout(), "ERROR: [%s] %s\n", e.Path, e.Message)
	}
	return exitWithCode(fmt.Errorf("validation failed"), 1)
}
