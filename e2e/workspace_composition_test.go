package e2e

// TestWorkspaceComposition covers the local first phase of #227 (ADR-012)
// through the real CLI: cross-model relationships are merged and validated,
// models can be imported partially (selected elements, cut off at a depth), and
// nested elements are merged once.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const teamAModel = `{
  "specification": {
    "elements": { "system": { "notation": "System", "container": true }, "container": { "notation": "Container", "container": true } },
    "relationships": { "uses": { "notation": "uses" } }
  },
  "model": {
    "shop": { "kind": "system", "title": "Shop", "children": {
      "api": { "kind": "container", "title": "API", "children": { "handler": { "kind": "container", "title": "Handler" } } }
    } },
    "admin": { "kind": "system", "title": "Admin" }
  },
  "relationships": [],
  "views": { "overview": { "title": "Overview", "include": ["shop"] } }
}
`

const teamBModel = `{
  "specification": {
    "elements": { "system": { "notation": "System" } },
    "relationships": { "uses": { "notation": "uses" } }
  },
  "model": { "billing": { "kind": "system", "title": "Billing" } },
  "relationships": [],
  "views": { "overview": { "title": "Overview", "include": ["billing"] } }
}
`

func writeWorkspace(t *testing.T, config string) (dir, cfgPath string) {
	t.Helper()
	dir = t.TempDir()
	for name, content := range map[string]string{"a.jsonc": teamAModel, "b.jsonc": teamBModel, "workspace.json": config} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir, filepath.Join(dir, "workspace.json")
}

type mergedModel struct {
	Model         map[string]json.RawMessage `json:"model"`
	Relationships []struct {
		From, To, Label string
	} `json:"relationships"`
	Views map[string]json.RawMessage `json:"views"`
}

func readMerged(t *testing.T, path string) mergedModel {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var m mergedModel
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("merged model is not valid JSON: %v", err)
	}
	return m
}

func TestWorkspaceComposition(t *testing.T) {
	bin := buildBinary(t)

	t.Run("cross-model relationships and partial imports", func(t *testing.T) {
		dir, cfg := writeWorkspace(t, `{
  "workspace": { "name": "landscape" },
  "models": [
    { "id": "team_a", "path": "a.jsonc", "elements": ["shop"], "depth": 2 },
    { "id": "team_b", "path": "b.jsonc", "prefix": "b" }
  ],
  "crossModelRelationships": [
    { "id": "pay", "from": "team_a:shop.api", "to": "team_b:billing", "label": "charges via", "kind": "uses" }
  ]
}`)
		if out, code := runCLIAllowFail(t, bin, dir, "workspace", "validate", cfg); code != 0 {
			t.Fatalf("validate: exit %d\n%s", code, out)
		}
		out := filepath.Join(dir, "merged.json")
		runCLI(t, bin, dir, "workspace", "merge", cfg, out)
		m := readMerged(t, out)

		for id := range m.Model {
			if strings.Contains(id, ".") {
				t.Errorf("nested element %q was merged as a top-level entry", id)
			}
		}
		if _, ok := m.Model["team_a_shop"]; !ok || len(m.Model) != 2 {
			t.Errorf("want team_a_shop and b_billing only (admin is not selected), got %v", keysOfRaw(m.Model))
		}
		if len(m.Relationships) != 1 || m.Relationships[0].From != "team_a_shop.api" || m.Relationships[0].To != "b_billing" {
			t.Errorf("cross-model relationship missing or wrong: %+v", m.Relationships)
		}
		if _, ok := m.Views["team_a_overview"]; ok {
			t.Error("views of a partially imported model must not be imported")
		}
		if _, ok := m.Views["b_overview"]; !ok {
			t.Error("views of a fully imported model are imported")
		}
		if list := runCLI(t, bin, dir, "workspace", "list", cfg); !strings.Contains(list, "Elements: shop") || !strings.Contains(list, "Depth: 2") {
			t.Errorf("list does not show the selection:\n%s", list)
		}
	})

	t.Run("a dangling cross-model relationship fails validate and merge", func(t *testing.T) {
		dir, cfg := writeWorkspace(t, `{
  "workspace": { "name": "landscape" },
  "models": [ { "id": "team_a", "path": "a.jsonc" }, { "id": "team_b", "path": "b.jsonc" } ],
  "crossModelRelationships": [ { "id": "x", "from": "team_a:shop.cache", "to": "team_b:billing" } ]
}`)
		out, code := runCLIAllowFail(t, bin, dir, "workspace", "validate", cfg)
		if code == 0 || !strings.Contains(out, `element "shop.cache" not found in model "team_a"`) {
			t.Errorf("validate: exit %d\n%s", code, out)
		}
		out, code = runCLIAllowFail(t, bin, dir, "workspace", "merge", cfg, filepath.Join(dir, "out.json"))
		if code == 0 || !strings.Contains(out, "shop.cache") {
			t.Errorf("merge: exit %d\n%s", code, out)
		}
		if _, err := os.Stat(filepath.Join(dir, "out.json")); err == nil {
			t.Error("merge wrote an output file although it failed")
		}
	})

	t.Run("constraints of a team still apply to its elements after the merge", func(t *testing.T) {
		const constrained = `{
  "specification": {
    "elements": { "system": { "notation": "System", "container": true }, "container": { "notation": "Container" } },
    "relationships": { "uses": { "notation": "uses" } }
  },
  "model": { "shop": { "kind": "system", "title": "Shop", "children": {
    "web": { "kind": "container", "title": "Web" }, "db": { "kind": "container", "title": "DB" } } } },
  "relationships": [ { "from": "shop.web", "to": "shop.db", "kind": "uses" } ],
  "views": {},
  "constraints": [ { "id": "no-direct-db", "description": "the web tier must not use the db", "rule": "no-relationship", "from": "shop.web", "to": "shop.db" } ]
}
`
		dir := t.TempDir()
		for name, content := range map[string]string{
			"a.jsonc":        constrained,
			"workspace.json": `{ "workspace": { "name": "w" }, "models": [ { "id": "team_a", "path": "a.jsonc" } ] }`,
		} {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		out := filepath.Join(dir, "merged.jsonc")
		runCLI(t, bin, dir, "workspace", "merge", filepath.Join(dir, "workspace.json"), out)

		lintOut, code := runCLIAllowFail(t, bin, dir, "lint", "--model", out)
		if code == 0 || !strings.Contains(lintOut, "no-direct-db") {
			t.Errorf("the constraint must still catch the relationship after the merge: exit %d\n%s", code, lintOut)
		}
	})

	t.Run("colliding merged IDs fail validate and merge", func(t *testing.T) {
		// prefix "a" + element "b_c" and prefix "a_b" + element "c" are both "a_b_c".
		const spec = `"specification": { "elements": { "system": { "notation": "System" } } }, "relationships": [], "views": {}`
		dir := t.TempDir()
		for name, content := range map[string]string{
			"one.jsonc": `{ ` + spec + `, "model": { "b_c": { "kind": "system", "title": "One" } } }`,
			"two.jsonc": `{ ` + spec + `, "model": { "c": { "kind": "system", "title": "Two" } } }`,
			"workspace.json": `{ "workspace": { "name": "w" }, "models": [
  { "id": "m1", "path": "one.jsonc", "prefix": "a" }, { "id": "m2", "path": "two.jsonc", "prefix": "a_b" } ] }`,
		} {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		cfg := filepath.Join(dir, "workspace.json")
		for _, args := range [][]string{{"workspace", "validate", cfg}, {"workspace", "merge", cfg, filepath.Join(dir, "out.json")}} {
			out, code := runCLIAllowFail(t, bin, dir, args...)
			if code == 0 || !strings.Contains(out, `"a_b_c"`) {
				t.Errorf("%s: exit %d, want a failure that names a_b_c\n%s", args[1], code, out)
			}
		}
		if _, err := os.Stat(filepath.Join(dir, "out.json")); err == nil {
			t.Error("merge wrote a file although the IDs collide")
		}
	})

	t.Run("a selection of an unknown element is reported", func(t *testing.T) {
		dir, cfg := writeWorkspace(t, `{
  "workspace": { "name": "landscape" },
  "models": [ { "id": "team_a", "path": "a.jsonc", "elements": ["nope"] } ]
}`)
		out, code := runCLIAllowFail(t, bin, dir, "workspace", "validate", cfg)
		if code == 0 || !strings.Contains(out, `"nope" not found`) {
			t.Errorf("exit %d\n%s", code, out)
		}
	})
}

func keysOfRaw(m map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
