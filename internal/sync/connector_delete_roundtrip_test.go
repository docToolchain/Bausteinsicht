package sync

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/drawio"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// deleteRoundtripModel has three relationships on one page; deleting the
// connector of the one in the middle shifts the array index of the last one.
func deleteRoundtripModel() *model.BausteinsichtModel {
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

// savedState builds the state a sync would save after a run.
func savedState(t *testing.T, m *model.BausteinsichtModel, doc *drawio.Document) *SyncState {
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

// TestRun_ConnectorDeletedInDrawioConverges (#719): the user deletes the
// connector of a relationship in draw.io. The sync removes the relationship
// from the model, which shifts the array index of every relationship behind it;
// the connectors in the document must be renumbered in the same run, otherwise
// the next sync reads the old and the new numbers as changes (drift, duplicate
// relationships, a model that no longer validates).
func TestRun_ConnectorDeletedInDrawioConverges(t *testing.T) {
	m := deleteRoundtripModel()
	doc := drawio.NewDocument()
	doc.AddPage("view-v", "V")
	ts := minimalTemplates(t)

	Run(m, doc, emptyState(), ts, map[string]bool{"view-v": true})
	state := savedState(t, m, doc)
	page := requirePage(t, doc, "view-v")
	if page.FindConnector("v--b", "v--c", 1) == nil {
		t.Fatal("setup: the connector of r1 is missing")
	}
	page.DeleteConnector("v--b", "v--c", 1) // the user deletes r1 in draw.io

	result := Run(m, doc, state, ts, nil)
	if result.Reverse.RelationshipsDeleted != 1 || len(m.Relationships) != 2 {
		t.Fatalf("the deletion must remove exactly r1 from the model, got %d deleted, relationships %+v",
			result.Reverse.RelationshipsDeleted, m.Relationships)
	}
	state = savedState(t, m, doc)

	// A sync without any change must now be a no-op, however often it runs.
	for i := 1; i <= 3; i++ {
		before := append([]model.Relationship(nil), m.Relationships...)
		result = Run(m, doc, state, ts, nil)
		if r := result.Reverse; r.RelationshipsCreated+r.RelationshipsDeleted+r.RelationshipsUpdated != 0 {
			t.Fatalf("sync %d after the deletion changed the model: %+v", i, r)
		}
		if len(m.Relationships) != len(before) {
			t.Fatalf("sync %d changed the relationships from %+v to %+v", i, before, m.Relationships)
		}
		if f := result.Forward; f.ConnectorsCreated+f.ConnectorsDeleted != 0 {
			t.Errorf("sync %d after the deletion created %d and deleted %d connectors, want a no-op", i, f.ConnectorsCreated, f.ConnectorsDeleted)
		}
		state = savedState(t, m, doc)
	}

	// The connector ids follow the model: r0 at 0, r2 now at 1.
	if page.FindConnector("v--c", "v--d", 1) == nil || page.FindConnector("v--c", "v--d", 2) != nil {
		t.Error("the connector of r2 was not renumbered to its new index 1")
	}
}
