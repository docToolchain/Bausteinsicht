package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestAddViewCmd_MissingTitleForNewView(t *testing.T) {
	dir := t.TempDir()
	modelPath := writeViewTestModel(t, dir)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"add", "view", "newview",
		"--model", modelPath,
	})
	err := cmd.Execute()

	if err == nil {
		t.Error("expected error for missing --title on new view")
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

func TestAddViewCmd_MergeViewFields(t *testing.T) {
	dir := t.TempDir()
	modelPath := writeViewTestModel(t, dir)

	// Create first view
	cmd1 := NewRootCmd()
	cmd1.SetArgs([]string{"add", "view", "containers",
		"--model", modelPath,
		"--title", "System Containers",
		"--include", "webshop",
	})
	if err := cmd1.Execute(); err != nil {
		t.Fatalf("unexpected error on first add: %v", err)
	}

	// Merge: update title
	cmd2 := NewRootCmd()
	cmd2.SetArgs([]string{"add", "view", "containers",
		"--model", modelPath,
		"--title", "Updated Containers",
	})
	if err := cmd2.Execute(); err != nil {
		t.Fatalf("unexpected error on merge title: %v", err)
	}

	// Merge: add to include list
	cmd3 := NewRootCmd()
	cmd3.SetArgs([]string{"add", "view", "containers",
		"--model", modelPath,
		"--include", "webshop",
		"--include", "payments",
	})
	if err := cmd3.Execute(); err != nil {
		t.Fatalf("unexpected error on merge include: %v", err)
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
    },
    "payments": {
      "kind": "system",
      "title": "Payment Service"
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
