package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// NewRootCmd creates and returns the root cobra command with global flags.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "bausteinsicht",
		Short:   "Architecture-as-code with draw.io synchronization",
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			format, _ := cmd.Flags().GetString("format")
			format = strings.ToLower(format)
			if format != "" && format != "text" && format != "json" {
				return fmt.Errorf("unknown format %q: valid values are \"text\" and \"json\"", format)
			}
			// Normalize to lowercase for all subcommands.
			_ = cmd.Flags().Set("format", format)

			// Validate --template extension when provided.
			templatePath, _ := cmd.Flags().GetString("template")
			if templatePath != "" && filepath.Ext(templatePath) != ".drawio" {
				return fmt.Errorf("template file %q must have a .drawio extension", templatePath)
			}

			// Validate --model path is under working directory (SEC-001).
			modelPath, _ := cmd.Flags().GetString("model")
			if modelPath != "" {
				if err := validatePathContainment(modelPath); err != nil {
					return fmt.Errorf("--model: %w", err)
				}
			}
			if templatePath != "" {
				if err := validatePathContainment(templatePath); err != nil {
					return fmt.Errorf("--template: %w", err)
				}
			}

			return nil
		},
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
	rootCmd.AddCommand(newExportCmd())
	rootCmd.AddCommand(newExportTableCmd())
	rootCmd.AddCommand(newExportDiagramCmd())
	rootCmd.AddCommand(newSchemaCmd())
	rootCmd.AddCommand(newImportCmd())

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

// validatePathContainment normalizes a path and rejects directory traversal
// sequences that could be used to write files at unexpected locations
// (SEC-001, SEC-016).
func validatePathContainment(path string) error {
	cleaned := filepath.Clean(path)
	for _, component := range strings.Split(cleaned, string(filepath.Separator)) {
		if component == ".." {
			return fmt.Errorf("path %q contains directory traversal", path)
		}
	}
	return nil
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
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), string(out))
	} else {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), err.Error())
	}
	return err
}
