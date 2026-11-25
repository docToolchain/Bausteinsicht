package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

const patchTestModel = `// keep this comment
{
  "specification": { "elements": { "system": { "notation": "System" } } },
  "model": {
    // inner comment
    "shop": { "kind": "system", "title": "Shop" }
  },
  "views": { "context": { "title": "Context" } }
}
`

func writePatchTestModel(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "architecture.jsonc")
	if err := os.WriteFile(p, []byte(patchTestModel), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestMarshalEntryJSON(t *testing.T) {
	got, err := marshalEntryJSON(map[string]string{"k": "a<b&c"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n    \"k\": \"a<b&c\"\n  }"
	if got != want {
		t.Errorf("got %q, want %q (no HTML escaping, indented for depth 1)", got, want)
	}
}

func TestSavePatched_PatchesWithoutNote(t *testing.T) {
	path := writePatchTestModel(t)
	m, err := model.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	err = savePatched(&stderr, path, m, func(data []byte) ([]byte, error) {
		return model.UpsertObjectField(data, []string{"views", "context"}, "scope", `"shop"`)
	})
	if err != nil {
		t.Fatal(err)
	}
	if stderr.Len() != 0 {
		t.Errorf("unexpected note: %q", stderr.String())
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "// inner comment") || !strings.Contains(string(data), `"scope": "shop"`) {
		t.Errorf("file not patched in place:\n%s", data)
	}
}

func TestSavePatched_FallsBackWithNote(t *testing.T) {
	path := writePatchTestModel(t)
	m, err := model.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	err = savePatched(&stderr, path, m, func([]byte) ([]byte, error) {
		return nil, os.ErrInvalid
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "note: model rewritten") {
		t.Errorf("fallback not announced, stderr = %q", stderr.String())
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "inner comment") {
		t.Errorf("expected the full save to rewrite the file:\n%s", data)
	}
}

func TestPatchView(t *testing.T) {
	data := []byte(patchTestModel)

	t.Run("new view", func(t *testing.T) {
		got, err := patchView(data, "shop", false, model.View{Title: "Shop", Include: []string{"shop"}}, model.View{}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(got), `"shop": {`) || !strings.Contains(string(got), "// inner comment") {
			t.Errorf("view not inserted:\n%s", got)
		}
	})

	t.Run("existing view without include gets one", func(t *testing.T) {
		got, err := patchView(data, "context", true, model.View{}, model.View{Title: "New title", Scope: "shop", Description: "d"}, []string{"shop"})
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{`"title": "New title"`, `"scope": "shop"`, `"description": "d"`, `"include": [`} {
			if !strings.Contains(string(got), want) {
				t.Errorf("missing %q:\n%s", want, got)
			}
		}
	})

	t.Run("unknown view fails so that the caller can fall back", func(t *testing.T) {
		if _, err := patchView(data, "missing", true, model.View{}, model.View{Title: "x"}, nil); err == nil {
			t.Error("expected an error for a view that is not in the file")
		}
	})
}

func TestInsertSpecificationKind(t *testing.T) {
	data := []byte(patchTestModel)

	got, err := insertSpecificationKind(data, "elements", "container", model.ElementKind{Notation: "Container"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), `"container": {`) {
		t.Errorf("kind not inserted into the existing section:\n%s", got)
	}

	got, err = insertSpecificationKind(data, "relationships", "uses", model.RelationshipKind{Notation: "uses"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), `"relationships": {`) || !strings.Contains(string(got), `"uses": {`) {
		t.Errorf("section not created:\n%s", got)
	}

	if _, err := insertSpecificationKind([]byte(`{}`), "elements", "x", model.ElementKind{Notation: "X"}); err == nil {
		t.Error("expected an error when there is no specification to patch")
	}
}
