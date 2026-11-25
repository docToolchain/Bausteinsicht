package e2e

// TestAddViewAndSpecificationPreserveFormatting covers #635: `add view` and
// `add specification` used to rewrite the whole model (comments lost, keys
// sorted, one-line objects expanded, "config": {} added, trailing newline
// removed), so a single command touched nearly every line. Like `add element`,
// they must only insert the new entry.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const patchSaveModel = `// Shop architecture
{
  "specification": {
    "elements": { "person": { "notation": "Person" }, "system": { "notation": "Software System" } }
  },
  "model": {
    // the buyer
    "customer": { "kind": "person", "title": "Customer" },
    "shop":     { "kind": "system", "title": "Shop" }
  },
  "views": { "context": { "title": "Context", "include": ["*"] } }
}
`

// setupPatchSaveModel writes the model into a fresh directory and returns the
// directory and the model path.
func setupPatchSaveModel(t *testing.T) (dir, path string) {
	t.Helper()
	dir = t.TempDir()
	path = filepath.Join(dir, "architecture.jsonc")
	if err := os.WriteFile(path, []byte(patchSaveModel), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir, path
}

// assertOnlyAdded checks that everything of the original model is still in the
// file byte for byte except the lines the command had to touch, and that none
// of the symptoms from #635 appear.
func assertOnlyAdded(t *testing.T, path string, wantSnippets ...string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	for _, keep := range []string{
		"// Shop architecture\n",
		"// the buyer\n",
		`"customer": { "kind": "person", "title": "Customer" },`,
		`"shop":     { "kind": "system", "title": "Shop" }`,
	} {
		if !strings.Contains(got, keep) {
			t.Errorf("original text %q is gone:\n%s", keep, got)
		}
	}
	if strings.Contains(got, `"config"`) {
		t.Errorf(`unexpected "config" entry:\n%s`, got)
	}
	if !strings.HasSuffix(got, "}\n") {
		t.Errorf("trailing newline was lost:\n%q", got[max(0, len(got)-20):])
	}
	if strings.Index(got, `"specification"`) > strings.Index(got, `"model"`) ||
		strings.Index(got, `"model"`) > strings.Index(got, `"views"`) {
		t.Errorf("top-level key order changed:\n%s", got)
	}
	for _, want := range wantSnippets {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in the file:\n%s", want, got)
		}
	}
}

func TestAddViewPreservesFormatting(t *testing.T) {
	bin := buildBinary(t)

	t.Run("new view is inserted", func(t *testing.T) {
		dir, path := setupPatchSaveModel(t)
		out, code := runCLIAllowFail(t, bin, dir, "add", "view", "shop", "--title", "Shop", "--include", "shop")
		if code != 0 {
			t.Fatalf("exit %d\n%s", code, out)
		}
		if strings.Contains(out, "note: model rewritten") {
			t.Errorf("unexpected rewrite note:\n%s", out)
		}
		assertOnlyAdded(t, path, `"shop": {`, `"context": { "title": "Context", "include": ["*"] }`) // existing view untouched
	})

	t.Run("existing view is extended and updated in place", func(t *testing.T) {
		dir, path := setupPatchSaveModel(t)
		out, code := runCLIAllowFail(t, bin, dir, "add", "view", "context", "--title", "System Context", "--include", "shop", "--include", "customer")
		if code != 0 {
			t.Fatalf("exit %d\n%s", code, out)
		}
		assertOnlyAdded(t, path, `"title": "System Context"`, `"shop"`, `"customer"`)
		// The file must still load and keep the order the entries were added in.
		stdout, stderr, c := runCLISplit(t, bin, dir, "add", "view", "context", "--include", "shop", "--format", "json")
		if c != 0 {
			t.Fatalf("second add view: exit %d\n%s\n%s", c, stdout, stderr)
		}
		if !strings.Contains(stdout, `"*"`) || !strings.Contains(stdout, `"added": []`) {
			t.Errorf("unexpected JSON result:\n%s", stdout)
		}
	})
}

func TestAddSpecificationPreservesFormatting(t *testing.T) {
	bin := buildBinary(t)

	t.Run("element kind", func(t *testing.T) {
		dir, path := setupPatchSaveModel(t)
		out, code := runCLIAllowFail(t, bin, dir, "add", "specification", "element", "container", "--notation", "Container")
		if code != 0 {
			t.Fatalf("exit %d\n%s", code, out)
		}
		assertOnlyAdded(t, path, `"container": {`, `"notation": "Container"`)
	})

	t.Run("relationship kind in a spec without relationships", func(t *testing.T) {
		dir, path := setupPatchSaveModel(t)
		out, code := runCLIAllowFail(t, bin, dir, "add", "specification", "relationship", "uses", "--notation", "uses")
		if code != 0 {
			t.Fatalf("exit %d\n%s", code, out)
		}
		assertOnlyAdded(t, path, `"relationships": {`, `"uses": {`)
		if out, code := runCLIAllowFail(t, bin, dir, "validate"); code != 0 {
			t.Fatalf("model no longer validates after the insert (exit %d):\n%s", code, out)
		}
	})
}
