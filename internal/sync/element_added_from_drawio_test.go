package sync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/drawio"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// addedElementModel has two element kinds; "actor" sorts first, so it is the
// kind that was used for every element that came from draw.io (#721).
func addedElementModel() *model.BausteinsichtModel {
	return &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"actor":     {Notation: "Actor"},
				"container": {Notation: "Container"},
			},
		},
		Model: map[string]model.Element{"user": {Kind: "actor", Title: "User"}},
		Views: map[string]model.View{"v": {Title: "V", Include: []string{"*"}}},
	}
}

// addObject adds an <object> to the first page, the way a hand edit or a pasted
// shape without sub-cells looks.
func addObject(t *testing.T, doc *drawio.Document, id, kind, label string) {
	t.Helper()
	root := doc.Pages()[0].Root()
	obj := root.CreateElement("object")
	obj.CreateAttr("id", "x1")
	obj.CreateAttr("label", label)
	obj.CreateAttr("bausteinsicht_id", id)
	if kind != "" {
		obj.CreateAttr("bausteinsicht_kind", kind)
	}
	cell := obj.CreateElement("mxCell")
	cell.CreateAttr("style", "rounded=1;")
	cell.CreateAttr("vertex", "1")
	cell.CreateAttr("parent", "1")
	geo := cell.CreateElement("mxGeometry")
	geo.CreateAttr("x", "900")
	geo.CreateAttr("y", "20")
	geo.CreateAttr("width", "120")
	geo.CreateAttr("height", "60")
	geo.CreateAttr("as", "geometry")
}

// syncAfterAdding returns the model after one sync cycle in which the document
// has a new shape that the model and the saved state do not know.
func syncAfterAdding(t *testing.T, add func(*drawio.Document)) (*model.BausteinsichtModel, *SyncResult) {
	t.Helper()
	m := addedElementModel()
	doc := drawio.NewDocument()
	doc.AddPage("view-v", "V")
	ts := minimalTemplates(t)
	Run(m, doc, emptyState(), ts, map[string]bool{"view-v": true})

	dir := t.TempDir()
	modelPath, drawioPath := filepath.Join(dir, "m.jsonc"), filepath.Join(dir, "a.drawio")
	for _, p := range []string{modelPath, drawioPath} {
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	state, err := BuildState(m, doc, modelPath, drawioPath)
	if err != nil {
		t.Fatal(err)
	}
	add(doc)
	result := Run(m, doc, state, ts, nil)
	return m, result
}

// TestRun_ElementAddedInDrawioKeepsItsKindAndTitle (#721): a shape with an
// unknown bausteinsicht_id was imported with the first kind of the
// specification (the bausteinsicht_kind attribute was ignored) and an empty
// title (the label was not used), so the model no longer validated and every
// following sync refused to run.
func TestRun_ElementAddedInDrawioKeepsItsKindAndTitle(t *testing.T) {
	tests := map[string]struct {
		add       func(*drawio.Document)
		wantKind  string
		wantTitle string
	}{
		"plain label and kind attribute": {
			add:       func(d *drawio.Document) { addObject(t, d, "brandnew", "container", "Brand New") },
			wantKind:  "container",
			wantTitle: "Brand New",
		},
		"html label (legacy format)": {
			add:       func(d *drawio.Document) { addObject(t, d, "brandnew", "container", "<b>Brand New</b>") },
			wantKind:  "container",
			wantTitle: "Brand New",
		},
		"sub-cell format": {
			add: func(d *drawio.Document) {
				if err := d.Pages()[0].CreateElement(drawio.ElementData{ID: "brandnew", Kind: "container", Title: "Brand New"}, ""); err != nil {
					t.Fatal(err)
				}
			},
			wantKind:  "container",
			wantTitle: "Brand New",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			m, result := syncAfterAdding(t, tt.add)

			el, ok := m.Model["brandnew"]
			if !ok || result.Reverse.ElementsCreated != 1 {
				t.Fatalf("the element must be imported once: %+v", result.Reverse)
			}
			if el.Kind != tt.wantKind || el.Title != tt.wantTitle {
				t.Errorf("imported as kind %q, title %q; want kind %q, title %q", el.Kind, el.Title, tt.wantKind, tt.wantTitle)
			}
			if errs := model.Validate(m); len(errs) > 0 {
				t.Errorf("the model is invalid after the import: %v", errs)
			}
		})
	}
}

// If the shape has no usable title or its kind is not in the specification, the
// import must still leave a valid model: the ID is the title, the first kind of
// the specification the kind, and the warning says what to review.
func TestRun_ElementAddedInDrawioWithoutTitleOrKnownKindStaysValid(t *testing.T) {
	m, result := syncAfterAdding(t, func(d *drawio.Document) { addObject(t, d, "brandnew", "nokind", "") })

	el, ok := m.Model["brandnew"]
	if !ok {
		t.Fatalf("the element must be imported: %+v", result.Reverse)
	}
	if el.Title != "brandnew" || el.Kind != "actor" {
		t.Errorf("imported as kind %q, title %q; want the ID as title and the first kind of the specification", el.Kind, el.Title)
	}
	if errs := model.Validate(m); len(errs) > 0 {
		t.Errorf("the model is invalid after the import: %v", errs)
	}
	if got := strings.Join(result.Warnings, "\n"); !strings.Contains(got, `"brandnew"`) {
		t.Errorf("the warning must name the element:\n%s", got)
	}
}
