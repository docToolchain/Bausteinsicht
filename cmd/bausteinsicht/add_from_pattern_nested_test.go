package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

const patternModel = `{
  "specification": {
    "elements": {
      "container": { "notation": "Container", "container": true },
      "datastore": { "notation": "Data Store" }
    },
    "relationships": { "uses": { "notation": "uses" } },
    "patterns": {
      "svc": {
        "description": "service with a database",
        "elements": [ { "id": "{base}", "kind": "container", "title": "{Title}",
          "children": [ { "id": "db", "kind": "datastore", "title": "{Title} DB" } ] } ],
        "relationships": [ { "id": "{base}-reads", "from": "{base}", "to": "{base}.db", "kind": "uses", "label": "reads" } ]
      },
      "pair": {
        "description": "two services, each with a child",
        "elements": [
          { "id": "{base}-api", "kind": "container", "title": "{Title} API",
            "children": [ { "id": "cache", "kind": "datastore", "title": "Cache" } ] },
          { "id": "{base}-worker", "kind": "container", "title": "{Title} Worker",
            "children": [ { "id": "queue", "kind": "datastore", "title": "Queue" }, { "id": "log", "kind": "datastore", "title": "Log" } ] },
          { "id": "{base}-plain", "kind": "container", "title": "{Title} Plain" }
        ],
        "relationships": []
      }
    }
  },
  "model": {},
  "relationships": [],
  "views": {}
}`

// applyPattern runs "add add-from-pattern" and turns a panic into a test failure
// (a crash must not take the other tests down with it).
func applyPattern(t *testing.T, args ...string) (path, stdout string, err error) {
	t.Helper()
	dir := t.TempDir()
	path = filepath.Join(dir, "architecture.jsonc")
	if werr := os.WriteFile(path, []byte(patternModel), 0o600); werr != nil {
		t.Fatal(werr)
	}
	cmd := NewRootCmd()
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(append([]string{"add", "add-from-pattern", "--model", path}, args...))
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic: %v", r)
			}
		}()
		err = cmd.Execute()
	}()
	return path, out.String(), err
}

// TestAddFromPatternWithChildren (#702): a pattern whose elements have children
// panicked with "index out of range" after the model had been written: the list
// of expanded IDs contains the children, the list of expanded elements does not.
func TestAddFromPatternWithChildren(t *testing.T) {
	path, out, err := applyPattern(t, "svc", "--id", "pay", "--title", "payment")
	if err != nil {
		t.Fatalf("the command failed: %v", err)
	}
	m, err := model.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	parent, ok := m.Model["pay"]
	if !ok || parent.Title != "Payment" {
		t.Fatalf("the service is missing or wrong: %+v", m.Model)
	}
	if db, ok := parent.Children["db"]; !ok || db.Title != "Payment DB" {
		t.Errorf("the child must be written below its parent: %+v", parent.Children)
	}
	if len(m.Relationships) != 1 || m.Relationships[0].To != "pay.db" {
		t.Errorf("relationships: %+v", m.Relationships)
	}
	for _, want := range []string{"pay ", "pay.db"} {
		if !strings.Contains(out, want) {
			t.Errorf("the summary must list %q:\n%s", want, out)
		}
	}
}

// With several top-level elements that have children, the IDs of the children
// were taken for the next top-level element (elements[1] was saved as the ID of
// the first child): wrong keys instead of a crash.
func TestAddFromPatternKeepsEveryTopLevelElementUnderItsOwnID(t *testing.T) {
	path, out, err := applyPattern(t, "pair", "--id", "pay")
	if err != nil {
		t.Fatalf("the command failed: %v", err)
	}
	m, err := model.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	for id, children := range map[string][]string{"pay-api": {"cache"}, "pay-worker": {"queue", "log"}, "pay-plain": nil} {
		el, ok := m.Model[id]
		if !ok {
			t.Errorf("top-level element %q is missing; the model has %v", id, keys(m.Model))
			continue
		}
		if len(el.Children) != len(children) {
			t.Errorf("%s has children %v, want %v", id, keys(el.Children), children)
		}
		for _, c := range children {
			if _, ok := el.Children[c]; !ok {
				t.Errorf("%s lacks the child %q", id, c)
			}
		}
	}
	if len(m.Model) != 3 {
		t.Errorf("exactly the three top-level elements are expected, got %v", keys(m.Model))
	}
	for _, want := range []string{"pay-api.cache", "pay-worker.queue", "pay-worker.log", "pay-plain"} {
		if !strings.Contains(out, want) {
			t.Errorf("the summary must list %q:\n%s", want, out)
		}
	}
}

func keys[V any](m map[string]V) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}
