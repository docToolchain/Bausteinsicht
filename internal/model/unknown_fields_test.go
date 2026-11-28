package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func loadText(t *testing.T, text string) *BausteinsichtModel {
	t.Helper()
	path := filepath.Join(t.TempDir(), "m.jsonc")
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func warningFor(ws []ValidationWarning, path string) (ValidationWarning, bool) {
	for _, w := range ws {
		if w.Path == path {
			return w, true
		}
	}
	return ValidationWarning{}, false
}

// #725: a misspelt key was dropped without a word, and gone after the next
// rewrite of the file.
func TestValidate_WarnsAboutUnknownFields(t *testing.T) {
	m := loadText(t, `{"specification":{"elements":{"a":{"notation":"A"}}},"modle":{},
 "model":{"x":{"kind":"a","title":"X","titel":"typo","tecnology":"Go","children":{"y":{"kind":"a","title":"Y","decisons":["d"]}}}},
 "relationships":[{"from":"x","to":"x","lable":"calls"}],
 "views":{"v":{"title":"V","include":["*"],"layuot":"layered"}}}`)

	ws := ValidateWithWarnings(m).Warnings

	for _, want := range []ValidationWarning{
		{Path: "(top level)", Message: `unknown top-level key "modle" (did you mean "model"?)`},
		{Path: "model.x", Message: `unknown field "titel" (did you mean "title"?)`},
		{Path: "model.x", Message: `unknown field "tecnology" (did you mean "technology"?)`},
		{Path: "model.x.children.y", Message: `unknown field "decisons" (did you mean "decisions"?)`},
		{Path: "relationships[0]", Message: `unknown field "lable" (did you mean "label"?)`},
		{Path: "views.v", Message: `unknown field "layuot" (did you mean "layout"?)`},
	} {
		found := false
		for _, w := range ws {
			if w == want {
				found = true
			}
		}
		if !found {
			t.Errorf("no warning %+v, warnings: %v", want, ws)
		}
	}
}

func TestValidate_UnknownFieldWithoutNeighbourHasNoSuggestion(t *testing.T) {
	m := loadText(t, `{"specification":{"elements":{"a":{"notation":"A"}}},"model":{"x":{"kind":"a","title":"X","zzzzzz":1}}}`)
	w, ok := warningFor(ValidateWithWarnings(m).Warnings, "model.x")
	if !ok || w.Message != `unknown field "zzzzzz"` {
		t.Errorf("warning: %+v (found %v)", w, ok)
	}
}

func TestValidate_FreeFormDataIsNotReported(t *testing.T) {
	m := loadText(t, `{"$schema":"x.json","meta":{"anything":{"goes":1}},
 "specification":{"elements":{"a":{"notation":"A"}}},"model":{"x":{"kind":"a","title":"X","metadata":{"owner":"me"}}}}`)
	for _, w := range ValidateWithWarnings(m).Warnings {
		if strings.Contains(w.Message, "unknown") {
			t.Errorf("unexpected warning: %+v", w)
		}
	}
}
