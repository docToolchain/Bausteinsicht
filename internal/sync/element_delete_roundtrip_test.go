package sync

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/drawio"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

func elementDeleteModel() *model.BausteinsichtModel {
	return &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"a": {Kind: "container", Title: "A"},
			"b": {Kind: "container", Title: "B"},
			"c": {Kind: "container", Title: "C"},
			"d": {Kind: "container", Title: "D"},
		},
		Relationships: []model.Relationship{
			{From: "a", To: "b", Label: "r0"},
			{From: "b", To: "c", Label: "r1"},
			{From: "c", To: "d", Label: "r2"},
		},
		Views: map[string]model.View{"v": {Title: "V", Include: []string{"*"}}},
	}
}

func elementDeleteState(t *testing.T, m *model.BausteinsichtModel, doc *drawio.Document) *SyncState {
	t.Helper()
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
	return state
}

// TestRun_ElementDeletedInDrawioIsNotDrawnAgain (#720): the user deletes an
// element in draw.io. Forward sync runs before reverse sync, and the model
// still has the element at that point, so it drew the element again; reverse
// sync then removed it from the model, and the next sync imported the redrawn
// shape as a new element ("New element ... added from draw.io (kind=actor)").
func TestRun_ElementDeletedInDrawioIsNotDrawnAgain(t *testing.T) {
	m := elementDeleteModel()
	doc := drawio.NewDocument()
	doc.AddPage("view-v", "V")
	ts := minimalTemplates(t)

	Run(m, doc, emptyState(), ts, map[string]bool{"view-v": true})
	state := elementDeleteState(t, m, doc)
	page := requirePage(t, doc, "view-v")
	if page.FindElement("c") == nil {
		t.Fatal("setup: element c is not on the page")
	}
	page.DeleteConnectorsFor("v--c")
	page.DeleteElement("c") // the user deletes c in draw.io

	result := Run(m, doc, state, ts, nil)
	if result.Reverse.ElementsDeleted != 1 {
		t.Fatalf("the deletion must remove c from the model: %+v", result.Reverse)
	}
	if _, ok := m.Model["c"]; ok {
		t.Fatal("c is still in the model")
	}
	if page.FindElement("c") != nil {
		t.Fatal("c was drawn again in the same run, so the next sync would import it as a new element")
	}
	state = elementDeleteState(t, m, doc)

	for i := 1; i <= 3; i++ {
		result = Run(m, doc, state, ts, nil)
		if r := result.Reverse; r.ElementsCreated+r.ElementsDeleted+r.ElementsUpdated+r.RelationshipsCreated+r.RelationshipsDeleted != 0 {
			t.Fatalf("sync %d after the deletion changed the model: %+v", i, r)
		}
		if _, ok := m.Model["c"]; ok || len(m.Model) != 3 {
			t.Fatalf("sync %d changed the elements: %v", i, m.Model)
		}
		state = elementDeleteState(t, m, doc)
	}
}

// An element that a constraint names cannot be removed from the model, so it is
// kept (the model wins). It must then be drawn again in the same run, not one
// sync later.
func TestRun_ElementDeletedInDrawioButKeptIsDrawnAgain(t *testing.T) {
	m := elementDeleteModel()
	m.Constraints = []model.Constraint{{ID: "c1", Rule: "no-relationship", From: "c", To: "a"}}
	doc := drawio.NewDocument()
	doc.AddPage("view-v", "V")
	ts := minimalTemplates(t)

	Run(m, doc, emptyState(), ts, map[string]bool{"view-v": true})
	state := elementDeleteState(t, m, doc)
	page := requirePage(t, doc, "view-v")
	page.DeleteConnectorsFor("v--c")
	page.DeleteElement("c")

	result := Run(m, doc, state, ts, nil)
	if _, ok := m.Model["c"]; !ok || result.Reverse.ElementsDeleted != 0 {
		t.Fatalf("the element must stay in the model: %+v", result.Reverse)
	}
	if page.FindElement("c") == nil {
		t.Fatal("the element that was kept must be drawn again in the same run")
	}
	state = elementDeleteState(t, m, doc)
	result = Run(m, doc, state, ts, nil)
	if r := result.Reverse; r.ElementsCreated+r.ElementsDeleted+r.RelationshipsCreated+r.RelationshipsDeleted != 0 {
		t.Errorf("the next sync is not a no-op: %+v", r)
	}
}
