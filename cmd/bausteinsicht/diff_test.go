package main

import (
	"os"
	"strings"
	"testing"
)

func createTempFile(t *testing.T, content string) string {
	tmpFile, err := os.CreateTemp("", "test-*.jsonc")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() {
		if err := tmpFile.Close(); err != nil {
			t.Logf("Failed to close temp file: %v", err)
		}
	}()

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	return tmpFile.Name()
}

func removeTempFile(t *testing.T, path string) {
	if err := os.Remove(path); err != nil {
		t.Logf("Failed to remove temp file: %v", err)
	}
}

func TestDiff_NoAsIsToBe(t *testing.T) {
	// Create temp model without asIs/toBe
	modelContent := `{
    "specification": {
      "elements": {
        "system": { "notation": "Software System" }
      }
    },
    "model": {
      "mysystem": { "kind": "system", "title": "My System" }
    },
    "views": {}
  }`

	tmpFile := createTempFile(t, modelContent)
	defer removeTempFile(t, tmpFile)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"diff", "--model", tmpFile, "--format", "text"})
	err := cmd.Execute()

	if err == nil {
		t.Error("Expected error for model without asIs/toBe")
	}
}

func TestDiff_TextFormat(t *testing.T) {
	// Create model with asIs/toBe
	modelContent := `{
    "specification": {
      "elements": {
        "system": { "notation": "Software System" }
      }
    },
    "model": {
      "mysystem": { "kind": "system", "title": "My System" }
    },
    "asIs": {
      "elements": {
        "legacy": { "kind": "system", "title": "Legacy System" }
      },
      "relationships": []
    },
    "toBe": {
      "elements": {
        "modern": { "kind": "system", "title": "Modern System" }
      },
      "relationships": []
    },
    "views": {}
  }`

	tmpFile := createTempFile(t, modelContent)
	defer removeTempFile(t, tmpFile)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"diff", "--model", tmpFile, "--format", "text"})
	err := cmd.Execute()

	if err != nil {
		t.Errorf("diff command failed: %v", err)
	}
}

func TestDiff_JSONFormat(t *testing.T) {
	modelContent := `{
    "specification": {
      "elements": {
        "system": { "notation": "Software System" }
      }
    },
    "model": {
      "mysystem": { "kind": "system", "title": "My System" }
    },
    "asIs": {
      "elements": {
        "api": { "kind": "container", "title": "API v1" }
      },
      "relationships": []
    },
    "toBe": {
      "elements": {
        "api": { "kind": "container", "title": "API v2" }
      },
      "relationships": []
    },
    "views": {}
  }`

	tmpFile := createTempFile(t, modelContent)
	defer removeTempFile(t, tmpFile)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"diff", "--model", tmpFile, "--format", "json"})
	err := cmd.Execute()

	if err != nil {
		t.Errorf("diff command failed: %v", err)
	}
}

func TestDiff_InvalidFormat(t *testing.T) {
	modelContent := `{
    "specification": {
      "elements": {
        "system": { "notation": "Software System" }
      }
    },
    "asIs": {
      "elements": {},
      "relationships": []
    },
    "toBe": {
      "elements": {},
      "relationships": []
    },
    "views": {}
  }`

	tmpFile := createTempFile(t, modelContent)
	defer removeTempFile(t, tmpFile)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"diff", "--model", tmpFile, "--format", "invalid"})
	err := cmd.Execute()

	if err == nil {
		t.Error("Expected error for invalid format")
	}
}

const diffViewModel = `{"specification":{"elements":{"actor":{"notation":"A"},"db":{"notation":"D"}},"relationships":{"uses":{"notation":"uses"}}},
 "model":{"u":{"kind":"actor","title":"U"}},
 "asIs":{"elements":{"u":{"kind":"actor","title":"U"},"old":{"kind":"db","title":"Old"}},"relationships":[{"from":"u","to":"old","kind":"uses"}]},
 "toBe":{"elements":{"u":{"kind":"actor","title":"U"},"new":{"kind":"db","title":"New"}},"relationships":[{"from":"u","to":"new","kind":"uses"}]},
 "views":{"v":{"title":"V","include":["u"]},"w":{"title":"W","include":["u","new"]}}}`

func runDiffOn(t *testing.T, args ...string) (string, error) {
	t.Helper()
	path := createTempFile(t, diffViewModel)
	t.Cleanup(func() { removeTempFile(t, path) })
	return executeImportCmd(append([]string{"diff", "--model", path}, args...)...)
}

// diff --view keeps the changes of the elements of that view only (#709).
func TestDiff_ViewFiltersTheChanges(t *testing.T) {
	full, err := runDiffOn(t)
	if err != nil || !strings.Contains(full, "new") || !strings.Contains(full, "old") {
		t.Fatalf("without --view: want the whole diff, got %q (%v)", full, err)
	}
	out, err := runDiffOn(t, "--view", "v")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "new") || strings.Contains(out, "old") || !strings.Contains(out, "No changes found") {
		t.Errorf("view v shows only u, which did not change; want no changes, got:\n%s", out)
	}
	out, err = runDiffOn(t, "--view", "w", "--format", "json")
	if err != nil || !strings.Contains(out, `"new"`) || strings.Contains(out, `"old"`) || !strings.Contains(out, `"addedElements": 1`) {
		t.Errorf("view w shows u and new: want the added new, not old, with a fitting summary, got:\n%s (%v)", out, err)
	}
}

func TestDiff_UnknownViewFails(t *testing.T) {
	_, err := runDiffOn(t, "--view", "nope")
	if err == nil || exitCodeOf(t, err) != 1 || !strings.Contains(err.Error(), `"nope"`) || !strings.Contains(err.Error(), "v, w") {
		t.Errorf("want exit code 1 naming nope and the known views, got %v", err)
	}
}

// A relationship without a label is printed without "()" (#709).
func TestDiff_EmptyLabelHasNoParentheses(t *testing.T) {
	out, err := runDiffOn(t)
	if err != nil || strings.Contains(out, "()") || !strings.Contains(out, "u → new") {
		t.Errorf("want \"u → new\" without (), got %q (%v)", out, err)
	}
}
