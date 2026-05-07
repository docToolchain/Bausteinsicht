package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/docToolchain/Bausteinsicht/internal/workspace"
	"github.com/spf13/cobra"
)

func newWorkspaceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace",
		Short: "Manage multi-model workspaces",
		Long:  "Work with multi-model workspaces combining multiple architecture models.",
	}

	cmd.AddCommand(newWorkspaceMergeCmd())
	cmd.AddCommand(newWorkspaceValidateCmd())
	cmd.AddCommand(newWorkspaceListCmd())

	return cmd
}

func newWorkspaceMergeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "merge <config-file> <output-file>",
		Short: "Merge multiple models into a single unified model",
		Long:  "Reads a workspace configuration file and merges all referenced models into a single output file. Element IDs are prefixed to avoid collisions.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWorkspaceMerge(cmd, args[0], args[1])
		},
	}
}

func runWorkspaceMerge(cmd *cobra.Command, configPath, outputPath string) error {
	cfg, err := workspace.LoadConfig(configPath)
	if err != nil {
		return exitWithCode(fmt.Errorf("loading workspace config: %w", err), 2)
	}

	baseDir := filepath.Dir(configPath)
	loaded, err := workspace.LoadModels(cfg, baseDir)
	if err != nil {
		return exitWithCode(fmt.Errorf("loading models: %w", err), 2)
	}

	merged, err := workspace.MergeModels(loaded)
	if err != nil {
		return exitWithCode(fmt.Errorf("merging models: %w", err), 2)
	}

	// Validate merged model
	if errs := model.Validate(merged); len(errs) > 0 {
		return exitWithCode(fmt.Errorf("validation failed: %v", errs), 2)
	}

	// Save merged model
	data, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return exitWithCode(fmt.Errorf("marshaling merged model: %w", err), 2)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return exitWithCode(fmt.Errorf("writing output file: %w", err), 2)
	}

	format, _ := cmd.Flags().GetString("format")
	if format == "json" {
		out, _ := json.Marshal(map[string]interface{}{
			"message": "Models merged successfully",
			"output":  outputPath,
			"models":  len(loaded),
		})
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(out))
	} else {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Merged %d models into %s\n", len(loaded), outputPath)
	}

	return nil
}

func newWorkspaceValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate <config-file>",
		Short: "Validate a workspace configuration",
		Long:  "Validates that a workspace configuration is well-formed and all referenced models are valid.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWorkspaceValidate(cmd, args[0])
		},
	}
}

func runWorkspaceValidate(cmd *cobra.Command, configPath string) error {
	cfg, err := workspace.LoadConfig(configPath)
	if err != nil {
		return exitWithCode(fmt.Errorf("loading workspace config: %w", err), 2)
	}

	baseDir := filepath.Dir(configPath)
	loaded, err := workspace.LoadModels(cfg, baseDir)
	if err != nil {
		return exitWithCode(fmt.Errorf("loading models: %w", err), 2)
	}

	// Validate each model individually
	var validationErrs []error
	for _, lm := range loaded {
		if errs := model.Validate(lm.Model); len(errs) > 0 {
			for _, e := range errs {
				validationErrs = append(validationErrs, fmt.Errorf("%s: %v", lm.Ref.ID, e))
			}
		}
	}

	if len(validationErrs) > 0 {
		format, _ := cmd.Flags().GetString("format")
		if format == "json" {
			var errMsgs []string
			for _, e := range validationErrs {
				errMsgs = append(errMsgs, e.Error())
			}
			out, _ := json.Marshal(map[string]interface{}{
				"valid":  false,
				"errors": errMsgs,
			})
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(out))
		} else {
			for _, e := range validationErrs {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), e.Error())
			}
		}
		return exitWithCode(fmt.Errorf("validation failed"), 2)
	}

	format, _ := cmd.Flags().GetString("format")
	if format == "json" {
		out, _ := json.Marshal(map[string]interface{}{
			"valid":  true,
			"models": len(loaded),
		})
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(out))
	} else {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ Workspace configuration is valid (%d models)\n", len(loaded))
	}

	return nil
}

func newWorkspaceListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <config-file>",
		Short: "List models in a workspace configuration",
		Long:  "Shows all models referenced in a workspace configuration with their IDs, paths, and prefixes.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWorkspaceList(cmd, args[0])
		},
	}
}

func runWorkspaceList(cmd *cobra.Command, configPath string) error {
	cfg, err := workspace.LoadConfig(configPath)
	if err != nil {
		return exitWithCode(fmt.Errorf("loading workspace config: %w", err), 2)
	}

	format, _ := cmd.Flags().GetString("format")
	if format == "json" {
		out, _ := json.MarshalIndent(cfg, "", "  ")
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(out))
		return nil
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Workspace: %s\n", cfg.Workspace.Name)
	if cfg.Workspace.Description != "" {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Description: %s\n\n", cfg.Workspace.Description)
	} else {
		_, _ = fmt.Fprint(cmd.OutOrStdout(), "\n")
	}

	_, _ = fmt.Fprint(cmd.OutOrStdout(), "Models:\n")
	for i, ref := range cfg.Models {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %d. ID: %s, Path: %s", i+1, ref.ID, ref.Path)
		if ref.Prefix != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), ", Prefix: %s", ref.Prefix)
		}
		_, _ = fmt.Fprint(cmd.OutOrStdout(), "\n")
	}

	if len(cfg.CrossRels) > 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nCross-Model Relationships: %d\n", len(cfg.CrossRels))
	}

	if len(cfg.Views) > 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Workspace Views: %d\n", len(cfg.Views))
	}

	return nil
}
