package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestAddSpecificationElementCmd_MissingNotationFlag(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.AddCommand(newAddCmd())

	cmd.SetArgs([]string{"add", "specification", "element", "custom_type"})
	err := cmd.Execute()

	if err == nil {
		t.Error("expected error for missing --notation flag, got nil")
	}
	if err != nil && err.Error() != "required flag(s) \"notation\" not set" {
		t.Logf("got error: %v", err)
	}
}

func TestAddSpecificationRelationshipCmd_MissingNotationFlag(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.AddCommand(newAddCmd())

	cmd.SetArgs([]string{"add", "specification", "relationship", "custom_rel"})
	err := cmd.Execute()

	if err == nil {
		t.Error("expected error for missing --notation flag, got nil")
	}
	if err != nil && err.Error() != "required flag(s) \"notation\" not set" {
		t.Logf("got error: %v", err)
	}
}

func TestAddSpecificationElementCmd_WithNotation(t *testing.T) {
	dir := t.TempDir()
	modelPath := writeSpecTestModel(t, dir)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"add", "specification", "element", "custom_type",
		"--model", modelPath,
		"--notation", "Custom Component",
		"--description", "A custom type",
		"--container",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddSpecificationRelationshipCmd_WithNotation(t *testing.T) {
	dir := t.TempDir()
	modelPath := writeSpecTestModel(t, dir)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"add", "specification", "relationship", "custom_rel",
		"--model", modelPath,
		"--notation", "custom calls",
		"--dashed",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// invalidKeys are rejected by isValidKey for every command that names an
// identifier. Kept in one place so element, specification, and view keys are
// asserted against the same set -- they drifted apart once already (#582).
var invalidKeys = []string{
	"foo.bar",    // dot (hierarchy separator)
	"foo bar",    // space
	"foo@bar",    // special char
	"foo/bar",    // slash
	"",           // empty
	"123invalid", // starts with digit
	"_leading",   // starts with underscore
}

// TestAddSpecificationElementCmd_InvalidKeys covers the key rejection branch of
// "add specification element" at the command level, not just the validator.
func TestAddSpecificationElementCmd_InvalidKeys(t *testing.T) {
	for _, key := range invalidKeys {
		t.Run(key, func(t *testing.T) {
			dir := t.TempDir()
			modelPath := writeSpecTestModel(t, dir)

			cmd := NewRootCmd()
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetArgs([]string{"add", "specification", "element", key,
				"--model", modelPath,
				"--notation", "Box",
			})
			err := cmd.Execute()
			if err == nil {
				t.Fatalf("expected error for invalid specification key %q", key)
			}
			if e, ok := err.(*exitError); ok && e.code != 1 {
				t.Errorf("expected exit code 1, got %d", e.code)
			}
		})
	}
}

// TestAddSpecificationRelationshipCmd_InvalidKeys is the relationship
// counterpart of TestAddSpecificationElementCmd_InvalidKeys.
func TestAddSpecificationRelationshipCmd_InvalidKeys(t *testing.T) {
	for _, key := range invalidKeys {
		t.Run(key, func(t *testing.T) {
			dir := t.TempDir()
			modelPath := writeSpecTestModel(t, dir)

			cmd := NewRootCmd()
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetArgs([]string{"add", "specification", "relationship", key,
				"--model", modelPath,
				"--notation", "->",
			})
			err := cmd.Execute()
			if err == nil {
				t.Fatalf("expected error for invalid specification key %q", key)
			}
			if e, ok := err.(*exitError); ok && e.code != 1 {
				t.Errorf("expected exit code 1, got %d", e.code)
			}
		})
	}
}

// TestAddSpecificationCmd_CamelCaseKeys is the regression test for #582: the
// specification commands used a lowercase-only validator while the model,
// schema, validate, and "add element" all accepted camelCase.
func TestAddSpecificationCmd_CamelCaseKeys(t *testing.T) {
	for _, key := range []string{"boundaryObject", "applicationService"} {
		t.Run(key, func(t *testing.T) {
			dir := t.TempDir()
			modelPath := writeSpecTestModel(t, dir)

			cmd := NewRootCmd()
			cmd.SetArgs([]string{"add", "specification", "element", key,
				"--model", modelPath,
				"--notation", "Box",
			})
			if err := cmd.Execute(); err != nil {
				t.Errorf("unexpected error for camelCase key %q: %v", key, err)
			}
		})
	}
}

func writeSpecTestModel(t *testing.T, dir string) string {
	t.Helper()
	p := filepath.Join(dir, "architecture.jsonc")
	content := `{
  "specification": {
    "elements": {
      "system": {"notation": "box"}
    },
    "relationships": {
      "uses": {"notation": "->"}
    }
  },
  "model": {},
  "relationships": [],
  "views": {}
}`
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}
