package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestIsValidSpecKey(t *testing.T) {
	tests := []struct {
		key   string
		valid bool
	}{
		{"system", true},
		{"my_component", true},
		{"my-component", true},
		{"custom123", true},
		{"_invalid", false},    // starts with underscore
		{"123invalid", false},  // starts with digit
		{"Component", false},   // uppercase
		{"my component", false}, // space
		{"", false},            // empty
	}

	for _, tt := range tests {
		got := isValidSpecKey(tt.key)
		if got != tt.valid {
			t.Errorf("isValidSpecKey(%q) = %v, want %v", tt.key, got, tt.valid)
		}
	}
}

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
