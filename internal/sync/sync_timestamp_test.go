package sync

import (
	"strings"
	"testing"

	"github.com/beevik/etree"
	"github.com/docToolchain/Bausteinsicht/internal/drawio"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

func pageXML(t *testing.T, page *drawio.Page) string {
	t.Helper()
	doc := etree.NewDocumentWithRoot(page.Root().Copy())
	s, err := doc.WriteToString()
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func timestampTestModel() *model.BausteinsichtModel {
	return &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"a": {Kind: "container", Title: "A"},
			"b": {Kind: "container", Title: "B"},
		},
		Relationships: []model.Relationship{},
		Views:         map[string]model.View{"v": {Title: "V", Include: []string{"a", "b"}}},
	}
}

// "Last synced" is the time of the last sync that changed the page: a sync that
// changes nothing leaves the page as it is, however much later it runs (#731).
func TestApplyForward_LastSyncedOnlyChangesWithThePage(t *testing.T) {
	m := timestampTestModel()
	doc := drawio.NewDocument()
	doc.AddPage("view-v", "V")
	ts := minimalTemplates(t)
	opts := func(at string) ForwardOptions { return ForwardOptions{ModelPath: "m.jsonc", SyncTime: at} }

	ApplyForward(&ChangeSet{ModelElementChanges: []ElementChange{{ID: "a", Type: Added}, {ID: "b", Type: Added}}}, doc, ts, m, opts("2026-01-01 10:00"))
	first := pageXML(t, requirePage(t, doc, "view-v"))
	if !strings.Contains(first, "Last synced: 2026-01-01 10:00") {
		t.Fatalf("the first sync writes its time:\n%s", first)
	}

	result := ApplyForward(&ChangeSet{}, doc, ts, m, opts("2026-01-01 10:05"))
	if again := pageXML(t, requirePage(t, doc, "view-v")); again != first {
		t.Errorf("a sync that changes nothing must leave the page byte-identical, got:\n%s", again)
	}
	if result.MetadataUpdated != 0 {
		t.Errorf("nothing was updated, got MetadataUpdated=%d", result.MetadataUpdated)
	}

	m.Model["c"] = model.Element{Kind: "container", Title: "C"}
	m.Views["v"] = model.View{Title: "V", Include: []string{"a", "b", "c"}}
	ApplyForward(&ChangeSet{ModelElementChanges: []ElementChange{{ID: "c", Type: Added}}}, doc, ts, m, opts("2026-01-01 10:10"))
	if got := pageXML(t, requirePage(t, doc, "view-v")); !strings.Contains(got, "Last synced: 2026-01-01 10:10") {
		t.Errorf("a sync that changed the page writes its time:\n%s", got)
	}
}

// The text of the box (here the title of the view) still follows the model.
func TestApplyForward_MetadataFollowsTheViewTitle(t *testing.T) {
	m := timestampTestModel()
	doc := drawio.NewDocument()
	doc.AddPage("view-v", "V")
	ts := minimalTemplates(t)
	ApplyForward(&ChangeSet{ModelElementChanges: []ElementChange{{ID: "a", Type: Added}, {ID: "b", Type: Added}}}, doc, ts, m, ForwardOptions{ModelPath: "m.jsonc", SyncTime: "2026-01-01 10:00"})

	m.Views["v"] = model.View{Title: "Renamed", Include: []string{"a", "b"}}
	ApplyForward(&ChangeSet{}, doc, ts, m, ForwardOptions{ModelPath: "m.jsonc", SyncTime: "2026-01-01 10:07"})
	got := pageXML(t, requirePage(t, doc, "view-v"))
	if !strings.Contains(got, "Renamed") || !strings.Contains(got, "Last synced: 2026-01-01 10:07") {
		t.Errorf("a changed title updates the box and its time:\n%s", got)
	}
}
