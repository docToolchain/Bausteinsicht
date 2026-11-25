package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

const editCmdModel = `{
  "specification": {
    "elements": {"system": {"notation": "System", "container": true}, "container": {"notation": "Container"}},
    "relationships": {"uses": {"notation": "uses"}}
  },
  "model": {
    "shop": {"kind": "system", "title": "Shop", "children": {"web": {"kind": "container", "title": "Web"}}},
    "payment": {"kind": "system", "title": "Payment"}
  },
  "relationships": [{"from": "shop.web", "to": "payment", "label": "charges", "kind": "uses"}],
  "views": {"context": {"title": "Context", "include": ["shop", "payment"]}}
}`

func writeEditCmdModel(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "architecture.jsonc")
	if err := os.WriteFile(p, []byte(editCmdModel), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func exitCodeOf(t *testing.T, err error) int {
	t.Helper()
	var ee *exitError
	if !errors.As(err, &ee) {
		t.Fatalf("expected an exit error, got %v", err)
	}
	return ee.code
}

func TestRenameCmd_SavesAndReports(t *testing.T) {
	path := writeEditCmdModel(t)
	out, err := executeRootCmd("rename", "--model", path, "--id", "shop", "--to", "store")
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	if !strings.Contains(out, "relationships[0].from: shop.web -> store.web") || !strings.Contains(out, "change(s) saved") {
		t.Errorf("unexpected output:\n%s", out)
	}
	m, err := model.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Model["store"]; !ok || m.Relationships[0].From != "store.web" || m.Views["context"].Include[0] != "store" {
		t.Errorf("model not rewritten: %+v", m)
	}
}

func TestRenameCmd_DryRunJSONDoesNotWrite(t *testing.T) {
	path := writeEditCmdModel(t)
	out := captureJSON(t, "rename", "--model", path, "--id", "shop", "--to", "store", "--dry-run", "--format", "json")
	var res editResult
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if !res.DryRun || len(res.Changes) == 0 {
		t.Errorf("unexpected result: %+v", res)
	}
	if data, _ := os.ReadFile(path); string(data) != editCmdModel {
		t.Error("dry run changed the file")
	}
}

func TestRenameCmd_DryRunText(t *testing.T) {
	path := writeEditCmdModel(t)
	out, err := executeRootCmd("rename", "--model", path, "--id", "shop", "--to", "store", "--dry-run")
	if err != nil || !strings.Contains(out, "Dry run:") {
		t.Errorf("err=%v output:\n%s", err, out)
	}
}

func TestEditCmds_ExitCodes(t *testing.T) {
	tests := []struct {
		name string
		args []string
		code int
	}{
		{"unknown element", []string{"rename", "--id", "nope", "--to", "x"}, 1},
		{"update without a field", []string{"update", "element", "--id", "shop"}, 1},
		{"update with an unknown kind", []string{"update", "element", "--id", "shop", "--kind", "database"}, 1},
		{"remove a referenced element", []string{"remove", "element", "--id", "shop.web"}, 1},
		{"remove a missing relationship", []string{"remove", "relationship", "--from", "payment", "--to", "shop"}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeEditCmdModel(t)
			_, err := executeRootCmd(append(tt.args, "--model", path)...)
			if err == nil || exitCodeOf(t, err) != tt.code {
				t.Fatalf("err = %v, want exit code %d", err, tt.code)
			}
			if data, _ := os.ReadFile(path); string(data) != editCmdModel {
				t.Error("file changed although the command failed")
			}
		})
	}

	t.Run("missing model file is exit 2", func(t *testing.T) {
		_, err := executeRootCmd("rename", "--model", filepath.Join(t.TempDir(), "none.jsonc"), "--id", "a", "--to", "b")
		if err == nil || exitCodeOf(t, err) != 2 {
			t.Errorf("err = %v, want exit code 2", err)
		}
	})
}

// A change that would introduce a validation error is refused and nothing is
// saved; errors that were in the model before do not block it.
func TestRunEdit_RefusesNewValidationErrors(t *testing.T) {
	path := writeEditCmdModel(t)
	out, err := executeRootCmd("remove", "element", "--model", path, "--id", "payment", "--cascade")
	if err != nil {
		t.Fatalf("a clean cascade removal must work: %v\n%s", err, out)
	}

	// Pre-existing error: the view includes an element that does not exist.
	broken := strings.Replace(editCmdModel, `"include": ["shop", "payment"]`, `"include": ["shop", "ghost"]`, 1)
	p := filepath.Join(t.TempDir(), "architecture.jsonc")
	if err := os.WriteFile(p, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := executeRootCmd("update", "element", "--model", p, "--id", "shop", "--title", "Store"); err != nil {
		t.Errorf("an unrelated, pre-existing error must not block the edit: %v\n%s", err, out)
	}
}

func TestUpdateElementCmd_SetsAndClearsFields(t *testing.T) {
	path := writeEditCmdModel(t)
	if out, err := executeRootCmd("update", "element", "--model", path, "--id", "shop.web",
		"--title", "Storefront", "--technology", "Go", "--description", "d"); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if out, err := executeRootCmd("update", "element", "--model", path, "--id", "shop.web", "--technology", ""); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	m, _ := model.Load(path)
	web := m.Model["shop"].Children["web"]
	if web.Title != "Storefront" || web.Technology != "" || web.Description != "d" {
		t.Errorf("unexpected element: %+v", web)
	}
}

func TestPrintEditResult_RemovalAndEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := printEditResult(&buf, "text", false, []model.Change{{Path: "relationships[0]", From: "a -> b"}}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "removed relationships[0]: a -> b") {
		t.Errorf("unexpected text:\n%s", buf.String())
	}
	buf.Reset()
	if err := printEditResult(&buf, "json", true, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"changes": []`) {
		t.Errorf("an empty change list must be an array:\n%s", buf.String())
	}
}
