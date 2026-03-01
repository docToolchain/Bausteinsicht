package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPatchValue_TopLevelField(t *testing.T) {
	input := `{
  "name": "old",
  "age": 42
}`
	got, err := PatchValue([]byte(input), []string{"name"}, `"new"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{
  "name": "new",
  "age": 42
}`
	if string(got) != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestPatchValue_NestedField(t *testing.T) {
	input := `{
  "model": {
    "api": {
      "title": "API",
      "technology": "Go"
    }
  }
}`
	got, err := PatchValue([]byte(input), []string{"model", "api", "technology"}, `"Go 1.24"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{
  "model": {
    "api": {
      "title": "API",
      "technology": "Go 1.24"
    }
  }
}`
	if string(got) != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestPatchValue_PreservesComments(t *testing.T) {
	input := `{
  // Model elements
  "model": {
    "api": {
      "title": "API", // the API service
      "technology": "Go"
    }
  }
}`
	got, err := PatchValue([]byte(input), []string{"model", "api", "technology"}, `"Rust"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Comments must be preserved.
	if !strings.Contains(string(got), "// Model elements") {
		t.Error("top-level comment was stripped")
	}
	if !strings.Contains(string(got), "// the API service") {
		t.Error("inline comment was stripped")
	}
	if !strings.Contains(string(got), `"technology": "Rust"`) {
		t.Error("value not patched")
	}
}

func TestPatchValue_PreservesKeyOrder(t *testing.T) {
	input := `{
  "z_last": "1",
  "a_first": "2"
}`
	got, err := PatchValue([]byte(input), []string{"z_last"}, `"changed"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Key order must be preserved (z_last before a_first).
	zIdx := strings.Index(string(got), "z_last")
	aIdx := strings.Index(string(got), "a_first")
	if zIdx > aIdx {
		t.Error("key order not preserved: z_last should come before a_first")
	}
}

func TestPatchValue_ErrorOnMissingPath(t *testing.T) {
	input := `{"name": "test"}`
	_, err := PatchValue([]byte(input), []string{"nonexistent"}, `"val"`)
	if err == nil {
		t.Error("expected error for missing path")
	}
}

func TestPatchSave_PreservesCommentsAndOrder(t *testing.T) {
	// Create a JSONC file with comments.
	original := `{
  // Architecture model
  "specification": {
    "elements": {
      "system": {
        "notation": "System"
      }
    }
  },
  "model": {
    "mySystem": {
      "kind": "system",
      "title": "My System",
      "technology": "Go", // programming language
      "description": "Main system"
    }
  },
  "relationships": [],
  "views": {}
}`
	dir := t.TempDir()
	path := filepath.Join(dir, "model.jsonc")
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	ops := []PatchOp{
		{Path: []string{"model", "mySystem", "technology"}, Value: `"Rust"`},
	}
	if err := PatchSave(path, ops); err != nil {
		t.Fatalf("PatchSave failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	result := string(data)

	// Comments preserved.
	if !strings.Contains(result, "// Architecture model") {
		t.Error("comment stripped")
	}
	if !strings.Contains(result, "// programming language") {
		t.Error("inline comment stripped")
	}
	// Value updated.
	if !strings.Contains(result, `"technology": "Rust"`) {
		t.Error("technology not updated")
	}
	// Model still parseable.
	m, err := Load(path)
	if err != nil {
		t.Fatalf("patched file not parseable: %v", err)
	}
	if m.Model["mySystem"].Technology != "Rust" {
		t.Errorf("expected technology Rust, got %s", m.Model["mySystem"].Technology)
	}
}
