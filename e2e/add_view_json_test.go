package e2e

// TestAddViewJSONReportsResultingView covers #641: `add view <key> --format
// json` on an existing view used to echo only this call's arguments, so title
// and scope showed up as empty strings and include listed only the new entries,
// although the model file itself was correct. The JSON must describe the view
// after the change, plus what this call changed.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const addViewJSONModel = `{
  "specification": {
    "elements": { "system": { "notation": "System" } }
  },
  "model": {
    "shop":    { "kind": "system", "title": "Shop" },
    "payment": { "kind": "system", "title": "Payment" }
  },
  "relationships": [],
  "views": { "context": { "title": "Context", "include": ["*"] } }
}
`

type addViewResult struct {
	ViewKey string   `json:"view_key"`
	Created bool     `json:"created"`
	Title   string   `json:"title"`
	Scope   string   `json:"scope"`
	Include []string `json:"include"`
	Added   []string `json:"added"`
}

func runAddViewJSON(t *testing.T, bin, dir string, args ...string) addViewResult {
	t.Helper()
	full := append([]string{"add", "view"}, args...)
	full = append(full, "--format", "json")
	stdout, stderr, code := runCLISplit(t, bin, dir, full...)
	if code != 0 {
		t.Fatalf("add view %v: exit %d\nstdout: %s\nstderr: %s", args, code, stdout, stderr)
	}
	var res addViewResult
	if err := json.Unmarshal([]byte(stdout), &res); err != nil {
		t.Fatalf("add view output is not valid JSON: %v\n%s", err, stdout)
	}
	return res
}

func TestAddViewJSONReportsResultingView(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "architecture.jsonc"), []byte(addViewJSONModel), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("extending an existing view reports the full view", func(t *testing.T) {
		got := runAddViewJSON(t, bin, dir, "context", "--include", "payment")
		want := addViewResult{
			ViewKey: "context",
			Created: false,
			Title:   "Context",
			Include: []string{"*", "payment"},
			Added:   []string{"payment"},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("an include that is already present is not reported as added", func(t *testing.T) {
		got := runAddViewJSON(t, bin, dir, "context", "--include", "payment")
		if len(got.Added) != 0 {
			t.Errorf("added = %v, want none", got.Added)
		}
		if !reflect.DeepEqual(got.Include, []string{"*", "payment"}) {
			t.Errorf("include = %v, want [* payment]", got.Include)
		}
	})

	t.Run("creating a view reports created", func(t *testing.T) {
		got := runAddViewJSON(t, bin, dir, "pay", "--title", "Payment view", "--include", "payment")
		want := addViewResult{
			ViewKey: "pay",
			Created: true,
			Title:   "Payment view",
			Include: []string{"payment"},
			Added:   []string{"payment"},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})
}
