package model

import (
	"reflect"
	"strings"
	"testing"
)

func TestSplitChangePath(t *testing.T) {
	tests := map[string][]string{
		"model.a.children.b.title":    {"model", "a", "children", "b", "title"},
		"relationships[3].from":       {"relationships", "[3]", "from"},
		"views.x.include[1]":          {"views", "x", "include", "[1]"},
		"dynamicViews[2].steps[0].to": {"dynamicViews", "[2]", "steps", "[0]", "to"},
		"relationships[0]":            {"relationships", "[0]"},
		"matrix[1][2]":                {"matrix", "[1]", "[2]"},
		"broken[":                     {"broken", "["},
	}
	for in, want := range tests {
		if got := SplitChangePath(in); !reflect.DeepEqual(got, want) {
			t.Errorf("SplitChangePath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestReplaceValueAt(t *testing.T) {
	data := []byte("{\n  // list\n  \"a\": [ \"x\", \"y\", { \"k\": \"v\" } ],\n  \"b\": 1\n}\n")
	got, err := ReplaceValueAt(data, []string{"a", "[1]"}, `"Z"`)
	if err != nil || string(got) != strings.Replace(string(data), `"y"`, `"Z"`, 1) {
		t.Fatalf("array element: %q, %v", got, err)
	}
	got, err = ReplaceValueAt(data, []string{"a", "[2]", "k"}, `"w"`)
	if err != nil || !strings.Contains(string(got), `{ "k": "w" }`) || !strings.Contains(string(got), "// list") {
		t.Fatalf("object inside an array: %q, %v", got, err)
	}
	for _, path := range [][]string{{"a", "[3]"}, {"b", "[0]"}, {"nope"}, {"a", "k"}, nil} {
		if _, err := ReplaceValueAt(data, path, "1"); err == nil {
			t.Errorf("path %v must fail", path)
		}
	}
}

func TestRenameObjectKey(t *testing.T) {
	data := []byte("{\n  \"model\": {\n    // c\n    \"old\": { \"title\": \"T\" }\n  }\n}\n")
	got, err := RenameObjectKey(data, []string{"model", "old"}, "new")
	if err != nil || !strings.Contains(string(got), `"new": { "title": "T" }`) || !strings.Contains(string(got), "// c") {
		t.Fatalf("%q, %v", got, err)
	}
	if _, err := RenameObjectKey([]byte(`{"a": ["x"]}`), []string{"a", "[0]"}, "b"); err == nil {
		t.Error("an array element has no key")
	}
}

func TestRemoveEntry(t *testing.T) {
	tests := []struct {
		name, in string
		path     []string
		want     string
	}{
		{"middle entry with a comment above",
			"{\n  // first\n  \"a\": 1,\n  // second\n  \"b\": 2,\n  \"c\": 3\n}\n", []string{"b"},
			"{\n  // first\n  \"a\": 1,\n  \"c\": 3\n}\n"},
		{"last entry: the comma before it goes, its trailing comment too",
			"{\n  \"a\": 1, // keep\n  \"b\": 2 // gone\n}\n", []string{"b"},
			"{\n  \"a\": 1 // keep\n}\n"},
		{"only entry",
			"{\n  \"a\": {\n    \"b\": 1\n  }\n}\n", []string{"a", "b"},
			"{\n  \"a\": {\n  }\n}\n"},
		{"entries on one line",
			`{ "a": 1, "b": 2, "c": 3 }`, []string{"b"}, `{ "a": 1, "c": 3 }`},
		{"last entry on one line",
			`{ "a": 1, "b": 2 }`, []string{"b"}, `{ "a": 1 }`},
		{"array element",
			"{\n  \"r\": [\n    // one\n    { \"x\": 1 },\n    { \"x\": 2 },\n    { \"x\": 3 } // last\n  ]\n}\n", []string{"r", "[2]"},
			"{\n  \"r\": [\n    // one\n    { \"x\": 1 },\n    { \"x\": 2 }\n  ]\n}\n"},
		{"first array element takes its comment",
			"{\n  \"r\": [\n    // one\n    { \"x\": 1 },\n    { \"x\": 2 }\n  ]\n}\n", []string{"r", "[0]"},
			"{\n  \"r\": [\n    { \"x\": 2 }\n  ]\n}\n"},
		{"windows line ends",
			"{\r\n  \"a\": 1,\r\n  \"b\": 2\r\n}\r\n", []string{"a"}, "{\r\n  \"b\": 2\r\n}\r\n"},
		{"a blank line stops the comment above",
			"{\n  // about the file\n\n  \"a\": 1,\n  \"b\": 2\n}\n", []string{"a"}, "{\n  // about the file\n\n  \"b\": 2\n}\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RemoveEntry([]byte(tt.in), tt.path)
			if err != nil || string(got) != tt.want {
				t.Fatalf("err %v\n got  %q\n want %q", err, got, tt.want)
			}
			if _, err := ParseModel(got); err != nil {
				t.Errorf("the result is not valid JSONC: %v\n%s", err, got)
			}
		})
	}
	if _, err := RemoveEntry([]byte(`{"a": 1}`), []string{"missing"}); err == nil {
		t.Error("removing a missing entry must fail")
	}
}

func TestPatchChanges(t *testing.T) {
	data := []byte(`{
  // spec
  "specification": { "elements": { "k": { "notation": "K" } } },
  "model": {
    // a
    "a": { "kind": "k", "title": "A", "technology": "Go" },
    "b": { "kind": "k", "title": "B" }
  },
  "relationships": [
    { "from": "a", "to": "b", "label": "x" },
    { "from": "b", "to": "a", "label": "y" }
  ],
  "views": { "v": { "title": "V", "include": ["a", "b"], "exclude": ["a"] } },
  "dynamicViews": [
    { "key": "d", "title": "D", "steps": [ { "index": 1, "from": "a", "to": "b" }, { "index": 2, "from": "b", "to": "b" } ] },
    { "key": "e", "title": "E", "steps": [ { "index": 1, "from": "a", "to": "a" } ] }
  ]
}
`)
	t.Run("update sets, inserts and clears fields", func(t *testing.T) {
		got, err := PatchUpdateChanges(data, []Change{
			{Path: "model.a.title", From: "A", To: "A2"},
			{Path: "model.a.technology", From: "Go", To: ""},
			{Path: "model.b.description", From: "", To: "new"},
		})
		if err != nil {
			t.Fatal(err)
		}
		s := string(got)
		if !strings.Contains(s, `"title": "A2"`) || strings.Contains(s, "technology") || !strings.Contains(s, `"description": "new"`) || !strings.Contains(s, "// a") {
			t.Errorf("unexpected result:\n%s", s)
		}
		// Clearing a field that is not there is not an error.
		if _, err := PatchUpdateChanges(data, []Change{{Path: "model.b.technology", From: "x", To: ""}}); err != nil {
			t.Errorf("clearing a missing field: %v", err)
		}
		if _, err := PatchUpdateChanges(data, []Change{{Path: "title", To: "x"}}); err == nil {
			t.Error("a path with one segment must fail")
		}
	})
	t.Run("rename changes the key and every reference", func(t *testing.T) {
		got, err := PatchRenameChanges(data, []Change{
			{Path: "model.a", From: "a", To: "z"},
			{Path: "relationships[0].from", From: "a", To: "z"},
			{Path: "relationships[1].to", From: "a", To: "z"},
			{Path: "views.v.include[0]", From: "a", To: "z"},
			{Path: "dynamicViews[0].steps[0].from", From: "a", To: "z"},
		})
		if err != nil {
			t.Fatal(err)
		}
		m, err := ParseModel(got)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := m.Model["z"]; !ok || m.Relationships[0].From != "z" || m.Relationships[1].To != "z" || m.Views["v"].Include[0] != "z" || m.DynamicViews[0].Steps[0].From != "z" {
			t.Errorf("a reference was not renamed:\n%s", got)
		}
		if !strings.Contains(string(got), "// a") || !strings.Contains(string(got), "// spec") {
			t.Errorf("comments lost:\n%s", got)
		}
		if out, err := PatchRenameChanges(data, nil); err != nil || string(out) != string(data) {
			t.Errorf("no changes must leave the text alone: %v", err)
		}
	})
	t.Run("remove with a cascade drops every reference", func(t *testing.T) {
		got, err := PatchRemoveChanges(data, []Change{
			{Path: "model.a", From: "a"},
			{Path: "relationships[0].from", From: "a"},
			{Path: "relationships[1].to", From: "a"},
			{Path: "views.v.include[0]", From: "a"},
			{Path: "views.v.exclude[0]", From: "a"},
			{Path: "dynamicViews[0].steps[0].from", From: "a"},
			{Path: "dynamicViews[1]", From: "e"},
		})
		if err != nil {
			t.Fatal(err)
		}
		m, err := ParseModel(got)
		if err != nil {
			t.Fatalf("%v\n%s", err, got)
		}
		if _, ok := m.Model["a"]; ok || len(m.Relationships) != 0 || len(m.Views["v"].Include) != 1 || len(m.Views["v"].Exclude) != 0 ||
			len(m.DynamicViews) != 1 || len(m.DynamicViews[0].Steps) != 1 {
			t.Errorf("unexpected result:\n%s", got)
		}
		if !strings.Contains(string(got), "// spec") {
			t.Errorf("comments lost:\n%s", got)
		}
	})
	t.Run("remove a relationship", func(t *testing.T) {
		got, err := PatchRemoveChanges(data, []Change{{Path: "relationships[1]", From: "b -> a"}})
		if err != nil || strings.Contains(string(got), `"label": "y"`) || !strings.Contains(string(got), `"label": "x"`) {
			t.Errorf("%v\n%s", err, got)
		}
	})
	t.Run("an unexpected path is an error", func(t *testing.T) {
		if _, err := PatchRemoveChanges(data, []Change{{Path: "somewhere.else", From: "x"}}); err == nil {
			t.Error("must fail")
		}
	})
}
