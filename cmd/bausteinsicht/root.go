package main

import "github.com/spf13/cobra"

// NewRootCmd creates and returns the root cobra command with global flags.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "bausteinsicht",
		Short:   "Architecture-as-code with draw.io synchronization",
		Version: version,
	}

	rootCmd.PersistentFlags().String("format", "text", "Output format: text or json")
	rootCmd.PersistentFlags().String("model", "", "Path to model file (.jsonc)")
	rootCmd.PersistentFlags().String("template", "", "Path to draw.io template file")
	rootCmd.PersistentFlags().Bool("verbose", false, "Enable verbose output")

	rootCmd.AddCommand(newAddCmd())

	return rootCmd
}

// exitError wraps an error with an exit code for structured error handling.
type exitError struct {
	err  error
	code int
}

func (e *exitError) Error() string { return e.err.Error() }

func exitWithCode(err error, code int) *exitError {
	return &exitError{err: err, code: code}
}
