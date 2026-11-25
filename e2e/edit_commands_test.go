package e2e

// TestEditCommands covers #634 through the real CLI: rename, update element,
// remove element and remove relationship, with --dry-run and --format json, on
// the model from the issue.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const editModel = `{
  "specification": {
    "elements": {
      "system":    { "notation": "Software System", "container": true },
      "container": { "notation": "Container" }
    },
    "relationships": { "uses": { "notation": "uses" } }
  },
  "model": {
    "shop": { "kind": "system", "title": "Shop", "children": {
      "web": { "kind": "container", "title": "Web App" }
    } },
    "payment": { "kind": "system", "title": "Payment" }
  },
  "relationships": [ { "from": "shop.web", "to": "payment", "label": "charges", "kind": "uses" } ],
  "views": { "shop": { "title": "Shop", "scope": "shop", "include": ["shop.*"] } },
  "dynamicViews": [ { "key": "checkout", "title": "Checkout", "steps": [
    { "index": 1, "from": "shop.web", "to": "payment", "label": "charge" }
  ] } ]
}
`

type editJSON struct {
	DryRun  bool `json:"dry_run"`
	Changes []struct {
		Path string `json:"path"`
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"changes"`
}

func newEditDir(t *testing.T) (dir, path string) {
	t.Helper()
	dir = t.TempDir()
	path = filepath.Join(dir, "architecture.jsonc")
	if err := os.WriteFile(path, []byte(editModel), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir, path
}

func mustValidate(t *testing.T, bin, dir string) {
	t.Helper()
	if out, code := runCLIAllowFail(t, bin, dir, "validate"); code != 0 {
		t.Fatalf("model must validate (exit %d):\n%s", code, out)
	}
}

func TestRenameElement(t *testing.T) {
	bin := buildBinary(t)

	t.Run("dry run lists the changes and writes nothing", func(t *testing.T) {
		dir, path := newEditDir(t)
		stdout, stderr, code := runCLISplit(t, bin, dir, "rename", "--id", "shop", "--to", "store", "--dry-run", "--format", "json")
		if code != 0 {
			t.Fatalf("exit %d\n%s\n%s", code, stdout, stderr)
		}
		var res editJSON
		if err := json.Unmarshal([]byte(stdout), &res); err != nil {
			t.Fatalf("invalid JSON: %v\n%s", err, stdout)
		}
		if !res.DryRun || len(res.Changes) != 5 {
			t.Errorf("unexpected result: %+v", res)
		}
		after, _ := os.ReadFile(path)
		if string(after) != editModel {
			t.Errorf("dry run changed the file")
		}
	})

	t.Run("rename rewrites key, children, relationships, view and dynamic view", func(t *testing.T) {
		dir, path := newEditDir(t)
		out, code := runCLIAllowFail(t, bin, dir, "rename", "--id", "shop", "--to", "store")
		if code != 0 {
			t.Fatalf("exit %d\n%s", code, out)
		}
		data, _ := os.ReadFile(path)
		got := string(data)
		if strings.Contains(got, `"shop"`) && !strings.Contains(got, `"views"`) {
			t.Errorf("old key left:\n%s", got)
		}
		for _, want := range []string{`"store"`, `"from": "store.web"`, `"scope": "store"`, `"store.*"`} {
			if !strings.Contains(got, want) {
				t.Errorf("missing %s in:\n%s", want, got)
			}
		}
		for _, old := range []string{`"shop.web"`, `"scope": "shop"`, `"shop.*"`} {
			if strings.Contains(got, old) {
				t.Errorf("stale reference %s left:\n%s", old, got)
			}
		}
		mustValidate(t, bin, dir)
	})

	t.Run("an unknown element fails with exit 1", func(t *testing.T) {
		dir, _ := newEditDir(t)
		if out, code := runCLIAllowFail(t, bin, dir, "rename", "--id", "nope", "--to", "x"); code != 1 || !strings.Contains(out, "not found") {
			t.Errorf("exit %d, output:\n%s", code, out)
		}
	})
}

func TestUpdateElement(t *testing.T) {
	bin := buildBinary(t)
	dir, path := newEditDir(t)

	out, code := runCLIAllowFail(t, bin, dir, "update", "element", "--id", "shop.web", "--title", "Storefront", "--technology", "Go")
	if code != 0 {
		t.Fatalf("exit %d\n%s", code, out)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), `"title": "Storefront"`) || !strings.Contains(string(data), `"technology": "Go"`) {
		t.Errorf("fields not updated:\n%s", data)
	}
	mustValidate(t, bin, dir)

	if out, code := runCLIAllowFail(t, bin, dir, "update", "element", "--id", "shop.web", "--kind", "database"); code != 1 || !strings.Contains(out, "unknown element kind") {
		t.Errorf("unknown kind: exit %d\n%s", code, out)
	}
	if out, code := runCLIAllowFail(t, bin, dir, "update", "element", "--id", "shop.web"); code != 1 || !strings.Contains(out, "nothing to update") {
		t.Errorf("no fields: exit %d\n%s", code, out)
	}
}

func TestRemoveCommands(t *testing.T) {
	bin := buildBinary(t)

	t.Run("a referenced element needs --cascade", func(t *testing.T) {
		dir, path := newEditDir(t)
		out, code := runCLIAllowFail(t, bin, dir, "remove", "element", "--id", "shop.web")
		if code != 1 || !strings.Contains(out, "--cascade") || !strings.Contains(out, "relationships[0]") {
			t.Fatalf("exit %d\n%s", code, out)
		}
		if after, _ := os.ReadFile(path); string(after) != editModel {
			t.Error("file changed although the command failed")
		}

		out, code = runCLIAllowFail(t, bin, dir, "remove", "element", "--id", "shop.web", "--cascade")
		if code != 0 {
			t.Fatalf("cascade: exit %d\n%s", code, out)
		}
		data, _ := os.ReadFile(path)
		if strings.Contains(string(data), "shop.web") || strings.Contains(string(data), "charges") {
			t.Errorf("references left after --cascade:\n%s", data)
		}
	})

	t.Run("remove relationship", func(t *testing.T) {
		dir, path := newEditDir(t)
		out, code := runCLIAllowFail(t, bin, dir, "remove", "relationship", "--from", "shop.web", "--to", "payment", "--label", "charges")
		if code != 0 {
			t.Fatalf("exit %d\n%s", code, out)
		}
		if data, _ := os.ReadFile(path); strings.Contains(string(data), `"charges"`) {
			t.Errorf("relationship not removed:\n%s", data)
		}
		mustValidate(t, bin, dir)
		if _, code := runCLIAllowFail(t, bin, dir, "remove", "relationship", "--from", "shop.web", "--to", "payment"); code != 1 {
			t.Errorf("removing it twice: exit %d, want 1", code)
		}
	})
}
