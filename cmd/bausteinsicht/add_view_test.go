package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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

// TestAddViewCmd_InvalidKeys covers the key rejection branch of "add view" at
// the command level. Shares invalidKeys with the specification tests so all
// three identifier commands are held to the same contract (#582).
func TestAddViewCmd_InvalidKeys(t *testing.T) {
	for _, key := range invalidKeys {
		t.Run(key, func(t *testing.T) {
			dir := t.TempDir()
			modelPath := writeViewTestModel(t, dir)

			cmd := NewRootCmd()
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetArgs([]string{"add", "view", key,
				"--model", modelPath,
				"--title", "Test",
			})
			err := cmd.Execute()
			if err == nil {
				t.Fatalf("expected error for invalid view key %q", key)
			}
			if e, ok := err.(*exitError); ok && e.code != 1 {
				t.Errorf("expected exit code 1, got %d", e.code)
			}
		})
	}
}

// TestAddViewCmd_CamelCaseKey is the "add view" half of the #582 regression:
// the view validator was lowercase-only.
func TestAddViewCmd_CamelCaseKey(t *testing.T) {
	dir := t.TempDir()
	modelPath := writeViewTestModel(t, dir)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"add", "view", "systemLandscape",
		"--model", modelPath,
		"--title", "System Landscape",
	})
	if err := cmd.Execute(); err != nil {
		t.Errorf("unexpected error for camelCase view key: %v", err)
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

// captureJSON runs the root command with args and returns what it printed to
// stdout, which is the "add view --format json" result.
func captureJSON(t *testing.T, args ...string) []byte {
	t.Helper()
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	cmd := NewRootCmd()
	cmd.SetArgs(args)
	execErr := cmd.Execute()
	_ = w.Close()
	out, readErr := io.ReadAll(r)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if execErr != nil {
		t.Fatalf("unexpected error: %v", execErr)
	}
	return out
}

// TestAddViewCmd_JSONReportsResultingView covers #641 at the command level:
// the JSON result describes the view after the change, not just this call's
// arguments.
func TestAddViewCmd_JSONReportsResultingView(t *testing.T) {
	modelPath := writeViewTestModel(t, t.TempDir())

	var created addViewResult
	out := captureJSON(t, "add", "view", "containers", "--model", modelPath,
		"--format", "json", "--title", "Containers", "--scope", "webshop",
		"--description", "All containers", "--include", "payments")
	if err := json.Unmarshal(out, &created); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if !created.Created || created.Title != "Containers" || created.Scope != "webshop" ||
		created.Description != "All containers" ||
		!reflect.DeepEqual(created.Include, []string{"payments"}) ||
		!reflect.DeepEqual(created.Added, []string{"payments"}) {
		t.Errorf("created view reported as %+v", created)
	}

	var merged addViewResult
	out = captureJSON(t, "add", "view", "containers", "--model", modelPath,
		"--format", "json", "--include", "webshop", "--include", "payments")
	if err := json.Unmarshal(out, &merged); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if merged.Created || merged.Title != "Containers" || merged.Scope != "webshop" ||
		merged.Description != "All containers" {
		t.Errorf("merge must keep the existing fields, got %+v", merged)
	}
	if !reflect.DeepEqual(merged.Include, []string{"payments", "webshop"}) {
		t.Errorf("include = %v, want the merged list [payments webshop]", merged.Include)
	}
	if !reflect.DeepEqual(merged.Added, []string{"webshop"}) {
		t.Errorf("added = %v, want only the new entry [webshop]", merged.Added)
	}
}

func TestAddViewCmd_JSONEmptyListsAreArrays(t *testing.T) {
	modelPath := writeViewTestModel(t, t.TempDir())

	out := captureJSON(t, "add", "view", "empty", "--model", modelPath,
		"--format", "json", "--title", "Empty")
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(out, &raw); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	for _, key := range []string{"include", "added"} {
		if string(raw[key]) != "[]" {
			t.Errorf("%s = %s, want []", key, raw[key])
		}
	}
}

func TestNewIncludes(t *testing.T) {
	tests := []struct {
		name              string
		existing, request []string
		want              []string
	}{
		{"nothing requested", []string{"a"}, nil, []string{}},
		{"all new, order kept", nil, []string{"b", "a"}, []string{"b", "a"}},
		{"already present is skipped", []string{"a"}, []string{"a", "b"}, []string{"b"}},
		{"duplicate in the request is reported once", nil, []string{"a", "a"}, []string{"a"}},
		{"empty entries are ignored", nil, []string{"", "a"}, []string{"a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newIncludes(tt.existing, tt.request); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newIncludes(%v, %v) = %v, want %v", tt.existing, tt.request, got, tt.want)
			}
		})
	}
}

func TestNonNil(t *testing.T) {
	if got := nonNil(nil); got == nil || len(got) != 0 {
		t.Errorf("nonNil(nil) = %#v, want an empty non-nil slice", got)
	}
	in := []string{"a"}
	if got := nonNil(in); !reflect.DeepEqual(got, in) {
		t.Errorf("nonNil(%v) = %v", in, got)
	}
}

// add view on an existing key says that it updated the view and what it added
// to the include list, not that it added the view (#704).
func TestAddView_ReportsAnUpdate(t *testing.T) {
	t.Chdir(t.TempDir())
	if out, err := executeImportCmd("init"); err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}

	out, err := executeImportCmd("add", "view", "v1", "--title", "V1", "--include", "onlineshop.api")
	if err != nil || !strings.Contains(out, "View 'v1' added to model") {
		t.Fatalf("a new view is reported as added, got %q (%v)", out, err)
	}

	out, err = executeImportCmd("add", "view", "v1", "--include", "onlineshop.db")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "added to model") || !strings.Contains(out, "View 'v1' updated") || !strings.Contains(out, "+onlineshop.db") {
		t.Errorf("want \"View 'v1' updated (include: +onlineshop.db)\", got %q", out)
	}
	if !strings.Contains(out, "onlineshop.api") || !strings.Contains(out, "onlineshop.db") {
		t.Errorf("the final include list is shown, got %q", out)
	}

	out, err = executeImportCmd("add", "view", "v1", "--include", "onlineshop.db")
	if err != nil || !strings.Contains(out, "View 'v1' updated") || strings.Contains(out, "+onlineshop.db") || !strings.Contains(out, "already") {
		t.Errorf("an include that is already there is skipped and said so, got %q (%v)", out, err)
	}
}
