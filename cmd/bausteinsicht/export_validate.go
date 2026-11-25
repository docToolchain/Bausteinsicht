package main

import (
	"fmt"

	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/spf13/cobra"
)

const noValidateFlag = "no-validate"

// addNoValidateFlag registers --no-validate on an export command.
func addNoValidateFlag(cmd *cobra.Command) {
	cmd.Flags().Bool(noValidateFlag, false, "Export even if the model has validation errors (default: abort, like 'validate')")
}

// validateForExport aborts an export when the model has validation errors, so
// that a typo in the model does not end up in generated documentation (#636).
// The errors are printed to stderr exactly as 'validate' prints them; warnings
// are not reported here. --no-validate skips the check.
func validateForExport(cmd *cobra.Command, m *model.BausteinsichtModel) error {
	if skip, _ := cmd.Flags().GetBool(noValidateFlag); skip {
		return nil
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
