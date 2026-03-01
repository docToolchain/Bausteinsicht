package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

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

	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(newSyncCmd())
	rootCmd.AddCommand(newValidateCmd())
	rootCmd.AddCommand(newAddCmd())
	rootCmd.AddCommand(newWatchCmd())

	return rootCmd
}

type exitError struct {
	err  error
	code int
}

func (e *exitError) Error() string { return e.err.Error() }

func exitWithCode(err error, code int) *exitError {
	return &exitError{err: err, code: code}
}

// ExecuteRoot runs the root command and writes errors in the appropriate format
// (JSON or plain text) to the command's error writer.
func ExecuteRoot(cmd *cobra.Command) error {
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	err := cmd.Execute()
	if err == nil {
		return nil
	}
	format, _ := cmd.PersistentFlags().GetString("format")
	code := 1
	if e, ok := err.(*exitError); ok {
		code = e.code
	}
	if format == "json" {
		out, _ := json.Marshal(map[string]interface{}{
			"error": err.Error(),
			"code":  code,
		})
		fmt.Fprintln(cmd.ErrOrStderr(), string(out))
	} else {
		fmt.Fprintln(cmd.ErrOrStderr(), err.Error())
	}
	return err
}
