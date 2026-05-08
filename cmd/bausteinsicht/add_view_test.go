package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestAddViewCmd_MissingTitleFlag(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.AddCommand(newAddCmd())

	cmd.SetArgs([]string{"add", "view", "myview"})
	err := cmd.Execute()

	if err == nil {
		t.Error("expected error for missing --title flag, got nil")
	}
	if err != nil && err.Error() != "required flag(s) \"title\" not set" {
		t.Logf("got error: %v", err)
	}
}

func TestAddViewCmd_WithTitle(t *testing.T) {
	dir := t.TempDir()
	modelPath := writeViewTestModel(t, dir)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"add", "view", "myview",
		"--model", modelPath,
		"--title", "My View",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddViewCmd_MissingViewKey(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.AddCommand(newAddCmd())

	cmd.SetArgs([]string{"add", "view"})
	err := cmd.Execute()

	if err == nil {
		t.Error("expected error for missing view key, got nil")
	}
}

func writeViewTestModel(t *testing.T, dir string) string {
	t.Helper()
	p := filepath.Join(dir, "architecture.jsonc")
	content := `{
  "specification": {
    "elements": {
      "system": {"notation": "box", "container": true}
    }
  },
  "model": {
    "webshop": {
      "kind": "system",
      "title": "Webshop"
    }
  },
  "relationships": [],
  "views": {}
}`
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}
