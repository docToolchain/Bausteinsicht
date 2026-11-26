package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validatePatternModel = `{
  "specification": {
    "elements": { "container": { "notation": "Container", "container": true } },
    "relationships": { "uses": { "notation": "uses" } },
    "patterns": {
      "ok": { "description": "one container",
        "elements": [ { "id": "{base}", "kind": "container", "title": "{Title}" } ], "relationships": [] },
      "badkind": { "description": "unknown kind",
        "elements": [ { "id": "{base}", "kind": "nokind", "title": "t" } ], "relationships": [] },
      "badrel": { "description": "relationship to an element that does not exist",
        "elements": [ { "id": "{base}", "kind": "container", "title": "t" } ],
        "relationships": [ { "id": "{base}-r", "from": "{base}", "to": "zzz", "kind": "uses" } ] }
    }
  },
  "model": { "existing": { "kind": "container", "title": "Existing" } },
  "relationships": [],
  "views": {}
}`

// runPatternAgainst writes model text to a file, runs "add add-from-pattern" and
// returns the file path, the error and the text of the file afterwards.
func runPatternAgainst(t *testing.T, modelText string, args ...string) (err error, after string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "architecture.jsonc")
	if werr := os.WriteFile(path, []byte(modelText), 0o600); werr != nil {
		t.Fatal(werr)
	}
	cmd := NewRootCmd()
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(append([]string{"add", "add-from-pattern", "--model", path}, args...))
	err = cmd.Execute()
	data, _ := os.ReadFile(path)
	return err, string(data) + "\n--output--\n" + out.String()
}

// TestAddFromPatternRefusesAnInvalidID (#703): --id was not checked against the
// ID rules, so "bad id" was saved as an element key; add element refuses it.
func TestAddFromPatternRefusesAnInvalidID(t *testing.T) {
	for name, args := range map[string][]string{
		"id with a space":    {"ok", "--id", "bad id"},
		"id with a dot":      {"ok", "--id", "a.b"},
		"id starting with 1": {"ok", "--id", "1abc"},
		"prefix with a dot":  {"ok", "--id", "x", "--prefix", "p.q"},
	} {
		t.Run(name, func(t *testing.T) {
			err, after := runPatternAgainst(t, validatePatternModel, args...)
			if err == nil || (err != nil && exitCodeOf(t, err) != 1) || !strings.Contains(err.Error(), "invalid") {
				t.Fatalf("the ID must be refused with exit code 1 and a message, got %v", err)
			}
			if !strings.HasPrefix(after, validatePatternModel) {
				t.Errorf("the model must not change:\n%s", after)
			}
		})
	}
}

// A pattern that expands to an invalid model (an element of a kind that does not
// exist, a relationship to an element that does not exist) was saved and left the
// model invalid; add element and add relationship refuse such input.
func TestAddFromPatternRefusesAnInvalidExpansion(t *testing.T) {
	for name, tt := range map[string]struct{ pattern, want string }{
		"unknown kind":          {"badkind", "nokind"},
		"dangling relationship": {"badrel", "zzz"},
	} {
		t.Run(name, func(t *testing.T) {
			err, after := runPatternAgainst(t, validatePatternModel, tt.pattern, "--id", "pay")
			if err == nil || (err != nil && exitCodeOf(t, err) != 1) {
				t.Fatalf("the pattern must be refused with exit code 1, got %v", err)
			}
			if !strings.Contains(err.Error()+after, tt.want) || !strings.Contains(err.Error(), "nothing was saved") {
				t.Errorf("the message must name the problem and say that nothing was saved: %v", err)
			}
			if !strings.HasPrefix(after, validatePatternModel) {
				t.Errorf("the model must not change:\n%s", after)
			}
		})
	}
}

// Errors that the model had before do not block a pattern that adds nothing
// wrong (the same rule as "update" and "remove").
func TestAddFromPatternIgnoresErrorsThatWereThereBefore(t *testing.T) {
	broken := strings.Replace(validatePatternModel, `"relationships": [],
  "views": {}`, `"relationships": [ { "from": "existing", "to": "gone", "kind": "uses" } ],
  "views": {}`, 1)
	err, after := runPatternAgainst(t, broken, "ok", "--id", "pay")
	if err != nil {
		t.Fatalf("a valid pattern must not be blocked by an error that was already there: %v", err)
	}
	if !strings.Contains(after, `"pay"`) {
		t.Errorf("the element must be added:\n%s", after)
	}
}
