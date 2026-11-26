package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// commentedModel is a model like the one "init" writes: comments above entries,
// a trailing comment, entries on one line and entries over several lines.
const commentedModel = `{
  "$schema": "https://raw.githubusercontent.com/docToolchain/Bausteinsicht/main/schemas/bausteinsicht.schema.json",

  // Specification: the kinds used in this model.
  "specification": {
    "elements": {
      "actor": { "notation": "Actor" },
      "system": { "notation": "System", "container": true },
      "container": { "notation": "Container", "container": true }
    },
    "relationships": { "uses": { "notation": "uses" } }
  },

  // Architecture model
  "model": {
    // The user
    "customer": {
      "kind": "actor",
      "title": "Customer", // shown on the diagram
      "description": "Buys things"
    },
    // The shop
    "shop": {
      "kind": "system",
      "title": "Shop",
      "technology": "Go",
      "children": {
        // The web tier
        "web": { "kind": "container", "title": "Web" },
        "db": { "kind": "container", "title": "Database" } // last entry
      }
    }
  },

  // Who talks to whom
  "relationships": [
    // the customer browses
    { "from": "customer", "to": "shop.web", "kind": "uses", "label": "browses" },
    { "from": "shop.web", "to": "shop.db", "kind": "uses", "label": "reads" },
    { "from": "customer", "to": "shop", "kind": "uses", "label": "pays" } // last
  ],

  "views": {
    // overview
    "ctx": { "title": "Context", "include": ["customer", "shop", "shop.web"], "exclude": ["shop.db"] },
    "inner": { "title": "Inner", "scope": "shop", "include": ["shop.*"] }
  },

  "dynamicViews": [
    // checkout
    { "key": "buy", "title": "Buy", "steps": [
      { "index": 1, "from": "customer", "to": "shop.web", "label": "go" },
      { "index": 2, "from": "shop.web", "to": "shop.db", "label": "read" }
    ] },
    { "key": "db_only", "title": "DB", "steps": [
      { "index": 1, "from": "shop.web", "to": "shop.db", "label": "q" }
    ] }
  ]
}
`

func writeCommentedModel(t *testing.T) (dir, path string) {
	t.Helper()
	dir = t.TempDir()
	path = filepath.Join(dir, "architecture.jsonc")
	if err := os.WriteFile(path, []byte(commentedModel), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir, path
}

func commentLines(text string) []string {
	var out []string
	for _, l := range strings.Split(text, "\n") {
		if i := strings.Index(l, "//"); i >= 0 && !strings.Contains(l[:i], "http") {
			out = append(out, strings.TrimSpace(l[i:]))
		}
	}
	return out
}

func asJSON(t *testing.T, m *model.BausteinsichtModel) string {
	t.Helper()
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// runEditCommand runs a model-changing command and returns what it wrote to
// stderr (the "model rewritten" note of the full save is announced there).
func runEditCommand(t *testing.T, args ...string) string {
	t.Helper()
	cmd := NewRootCmd()
	var out, errOut strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("%v: %v\n%s", args, err, errOut.String())
	}
	return errOut.String()
}

// editCase runs args against the commented model and checks the three things
// that matter: the result is the same model the full rewrite produces, the file
// keeps its comments except the ones that belonged to what was removed, and no
// "rewritten" note is printed (#685).
func editCase(t *testing.T, args []string, apply func(m *model.BausteinsichtModel) error, lostComments []string) string {
	t.Helper()
	_, path := writeCommentedModel(t)
	want, err := model.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := apply(want); err != nil {
		t.Fatal(err)
	}

	stderr := runEditCommand(t, append(args, "--model", path)...)

	if strings.Contains(stderr, "rewritten") {
		t.Errorf("the file was rewritten as a whole: %s", stderr)
	}
	got, err := model.Load(path)
	if err != nil {
		t.Fatalf("the edited file does not load: %v", err)
	}
	if asJSON(t, got) != asJSON(t, want) {
		t.Errorf("the file holds another model than the full rewrite would write:\n got  %s\n want %s", asJSON(t, got), asJSON(t, want))
	}
	data, _ := os.ReadFile(path)
	gone := map[string]bool{}
	for _, c := range lostComments {
		gone[c] = true
	}
	have := commentLines(string(data))
	for _, c := range commentLines(commentedModel) {
		if gone[c] {
			continue
		}
		found := false
		for _, h := range have {
			if h == c {
				found = true
			}
		}
		if !found {
			t.Errorf("the comment %q was lost", c)
		}
	}
	return string(data)
}

func TestUpdateElementKeepsComments(t *testing.T) {
	text := editCase(t, []string{"update", "element", "--id", "customer", "--title", "Client"},
		func(m *model.BausteinsichtModel) error {
			_, err := m.UpdateElement("customer", model.ElementUpdate{Title: ptr("Client")})
			return err
		}, nil)
	if !strings.Contains(text, `"title": "Client", // shown on the diagram`) {
		t.Errorf("the title must change in place, with its comment:\n%s", text)
	}
	if changed := changedLines(commentedModel, text); changed != 1 {
		t.Errorf("exactly one line may change, %d did", changed)
	}
}

func TestUpdateElementAddsAndRemovesFieldsKeepingComments(t *testing.T) {
	editCase(t, []string{"update", "element", "--id", "shop.web", "--technology", "React", "--description", "The web tier"},
		func(m *model.BausteinsichtModel) error {
			_, err := m.UpdateElement("shop.web", model.ElementUpdate{Technology: ptr("React"), Description: ptr("The web tier")})
			return err
		}, nil)
	editCase(t, []string{"update", "element", "--id", "shop", "--technology", ""},
		func(m *model.BausteinsichtModel) error {
			_, err := m.UpdateElement("shop", model.ElementUpdate{Technology: ptr("")})
			return err
		}, nil)
	editCase(t, []string{"update", "element", "--id", "customer", "--kind", "system"},
		func(m *model.BausteinsichtModel) error {
			_, err := m.UpdateElement("customer", model.ElementUpdate{Kind: ptr("system")})
			return err
		}, nil)
}

func TestRemoveRelationshipKeepsComments(t *testing.T) {
	editCase(t, []string{"remove", "relationship", "--from", "shop.web", "--to", "shop.db"},
		func(m *model.BausteinsichtModel) error {
			_, err := m.RemoveRelationship("shop.web", "shop.db", "")
			return err
		}, nil)
	// The first and the last entry have a comment of their own (above, and after the comma).
	editCase(t, []string{"remove", "relationship", "--from", "customer", "--to", "shop.web"},
		func(m *model.BausteinsichtModel) error {
			_, err := m.RemoveRelationship("customer", "shop.web", "")
			return err
		}, []string{"// the customer browses"})
	editCase(t, []string{"remove", "relationship", "--from", "customer", "--to", "shop"},
		func(m *model.BausteinsichtModel) error {
			_, err := m.RemoveRelationship("customer", "shop", "")
			return err
		}, []string{"// last"})
}

func TestRemoveElementKeepsOtherComments(t *testing.T) {
	// A leaf without references would be removed as it is; db is referenced, so
	// use --cascade, which drops relationships, view entries and steps as well.
	editCase(t, []string{"remove", "element", "--id", "shop.db", "--cascade"},
		func(m *model.BausteinsichtModel) error {
			_, err := m.RemoveElement("shop.db", true)
			return err
		}, []string{"// last entry"})
	editCase(t, []string{"remove", "element", "--id", "customer", "--cascade"},
		func(m *model.BausteinsichtModel) error {
			_, err := m.RemoveElement("customer", true)
			return err
		}, []string{"// The user", "// the customer browses", "// last"})
}

func TestRenameKeepsComments(t *testing.T) {
	text := editCase(t, []string{"rename", "--id", "shop.web", "--to", "frontend"},
		func(m *model.BausteinsichtModel) error {
			_, err := m.RenameElement("shop.web", "frontend")
			return err
		}, nil)
	if strings.Contains(text, `"web"`) || strings.Contains(text, "shop.web") {
		t.Errorf("a reference to the old key is left:\n%s", text)
	}
	editCase(t, []string{"rename", "--id", "shop", "--to", "store"},
		func(m *model.BausteinsichtModel) error {
			_, err := m.RenameElement("shop", "store")
			return err
		}, nil)
}

func ptr(s string) *string { return &s }

// changedLines counts the lines of after that are not in before (a one-line
// edit changes exactly one).
func changedLines(before, after string) int {
	seen := map[string]int{}
	for _, l := range strings.Split(before, "\n") {
		seen[l]++
	}
	n := 0
	for _, l := range strings.Split(after, "\n") {
		if seen[l] > 0 {
			seen[l]--
		} else {
			n++
		}
	}
	return n
}
