package main

import (
	"fmt"

	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/spf13/cobra"
)

const noValidateFlag = "no-validate"

// addNoValidateFlag makes an export command validate the model before it runs
// and registers --no-validate to skip that. A typo in the model would otherwise
// end up in generated documentation while the command exits 0 (#636).
//
// The check is a PreRunE hook so that it sits in one place instead of being
// repeated in every export command; it loads the model itself, like the
// command does afterwards.
func addNoValidateFlag(cmd *cobra.Command) {
	cmd.Flags().Bool(noValidateFlag, false, "Export even if the model has validation errors (default: abort, like 'validate')")
	cmd.PreRunE = validateModelBeforeExport
}

// validateModelBeforeExport aborts with exit code 1 when the model has
// validation errors. The errors are printed to stderr exactly as 'validate'
// prints them; warnings are not reported here.
func validateModelBeforeExport(cmd *cobra.Command, _ []string) error {
	if skip, _ := cmd.Flags().GetBool(noValidateFlag); skip {
		return nil
	}
	modelPath, _ := cmd.Flags().GetString("model")
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
	errs := model.Validate(m)
	if len(errs) == 0 {
		return nil
	}
	for _, e := range errs {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "ERROR: [%s] %s\n", e.Path, e.Message)
	}
	return exitWithCode(
		fmt.Errorf("model has %d validation error(s); fix them (see 'validate') or use --no-validate to export anyway", len(errs)),
		1,
	)
}
