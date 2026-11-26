package sync

import (
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/drawio"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// docWithElemOnPages returns a document with one page per entry of titles; every
// page shows the element "shop" with that page's title.
func docWithElemOnPages(titles ...string) *drawio.Document {
	doc := drawio.NewDocument()
	for i, title := range titles {
		viewID := string(rune('a' + i))
		page := doc.AddPage("view-"+viewID, "View "+viewID)
		_ = page.CreateElement(drawio.ElementData{ID: "shop", CellID: viewID + "--shop", Kind: "container", Title: title}, "")
	}
	return doc
}

func shopModelAndState(title string) (*model.BausteinsichtModel, *SyncState) {
	m := simpleModel("shop", title, "", "")
	m.Views = map[string]model.View{
		"a": {Title: "A", Include: []string{"*"}},
		"b": {Title: "B", Include: []string{"*"}},
		"c": {Title: "C", Include: []string{"*"}},
	}
	return m, stateWithElem("shop", title, "", "")
}

func titleChanges(cs *ChangeSet) []ElementChange {
	var out []ElementChange
	for _, ch := range cs.DrawioElementChanges {
		if ch.Field == fieldTitle {
			out = append(out, ch)
		}
	}
	return out
}

// TestDetectChanges_TitleEditedOnAnyPageIsDetected (#684): an element is drawn
// on several pages. The title edited on one of them must be reverse-synced,
// whichever page it is. Only the first page used to count (duplicate
// bausteinsicht_id: first occurrence wins), so an edit on a later page was
// ignored and then overwritten by forward sync; which page came first changed
// from run to run (#683).
func TestDetectChanges_TitleEditedOnAnyPageIsDetected(t *testing.T) {
	titles := map[string][]string{
		"edit on the first page":  {"Shop EDIT", "Shop", "Shop"},
		"edit on the second page": {"Shop", "Shop EDIT", "Shop"},
		"edit on the last page":   {"Shop", "Shop", "Shop EDIT"},
	}
	for name, pageTitles := range titles {
		t.Run(name, func(t *testing.T) {
			m, state := shopModelAndState("Shop")
			cs := DetectChanges(m, docWithElemOnPages(pageTitles...), state, nil)

			got := titleChanges(cs)
			if len(got) != 1 || got[0].ID != "shop" || got[0].NewValue != "Shop EDIT" {
				t.Fatalf("want one title change to %q, got %+v", "Shop EDIT", got)
			}
			for _, ch := range cs.ModelElementChanges {
				if ch.Field == fieldTitle {
					t.Errorf("forward sync must not overwrite the edit with the model's title: %+v", ch)
				}
			}
		})
	}
}

func TestRun_TitleEditedOnALaterPageReachesTheModel(t *testing.T) {
	m, state := shopModelAndState("Shop")
	doc := docWithElemOnPages("Shop", "Shop", "Shop EDIT")

	Run(m, doc, state, minimalTemplates(t), nil)

	if got := m.Model["shop"].Title; got != "Shop EDIT" {
		t.Fatalf("the model title is %q, want the edited %q", got, "Shop EDIT")
	}
}

// Two pages edited differently cannot both win: one value is taken (the first
// edited page) and the conflict is reported, not dropped silently.
func TestRun_DifferentEditsOnTwoPagesAreReported(t *testing.T) {
	m, state := shopModelAndState("Shop")

	result := Run(m, docWithElemOnPages("Shop", "Shop ONE", "Shop TWO"), state, minimalTemplates(t), nil)

	if got := m.Model["shop"].Title; got != "Shop ONE" {
		t.Fatalf("the first edited page must win, the model title is %q", got)
	}
	var found bool
	for _, w := range result.Warnings {
		if strings.Contains(w, `"shop"`) && strings.Contains(w, "Shop TWO") {
			found = true
		}
	}
	if !found {
		t.Errorf("the conflict must be reported with the ignored value: %v", result.Warnings)
	}
}

func TestDetectChanges_UneditedPagesProduceNoChange(t *testing.T) {
	m, state := shopModelAndState("Shop")
	cs := DetectChanges(m, docWithElemOnPages("Shop", "Shop", "Shop"), state, nil)
	if len(cs.DrawioElementChanges) != 0 {
		t.Errorf("no change expected: %+v", cs.DrawioElementChanges)
	}
}

// docWithConnectorOnPages draws the connector a->b with the given label on one
// page per label.
func docWithConnectorOnPages(labels ...string) *drawio.Document {
	doc := drawio.NewDocument()
	for i, label := range labels {
		viewID := string(rune('a' + i))
		page := doc.AddPage("view-"+viewID, "View "+viewID)
		for _, id := range []string{"a", "b"} {
			_ = page.CreateElement(drawio.ElementData{ID: id, CellID: viewID + "--" + id, Kind: "container", Title: id}, "")
		}
		page.CreateConnector(drawio.ConnectorData{From: "a", To: "b", Label: label, SourceRef: viewID + "--a", TargetRef: viewID + "--b"}, "")
	}
	return doc
}

// TestDetectChanges_ConnectorLabelEditedOnAnyPageIsDetected: the same as for
// titles, for the label of a connector that is drawn on several pages.
func TestDetectChanges_ConnectorLabelEditedOnAnyPageIsDetected(t *testing.T) {
	for name, labels := range map[string][]string{
		"first page":  {"edited", "calls"},
		"second page": {"calls", "edited"},
	} {
		t.Run(name, func(t *testing.T) {
			m := &model.BausteinsichtModel{
				Model: map[string]model.Element{
					"a": {Kind: "container", Title: "a"}, "b": {Kind: "container", Title: "b"},
				},
				Relationships: []model.Relationship{{From: "a", To: "b", Label: "calls"}},
				Views:         map[string]model.View{"a": {Title: "A", Include: []string{"*"}}, "b": {Title: "B", Include: []string{"*"}}},
			}
			state := emptyState()
			state.Elements["a"] = ElementState{Title: "a", Kind: "container"}
			state.Elements["b"] = ElementState{Title: "b", Kind: "container"}
			state.Relationships = []RelationshipState{{From: "a", To: "b", Index: 0, Label: "calls"}}

			cs := DetectChanges(m, docWithConnectorOnPages(labels...), state, nil)

			var got []RelationshipChange
			for _, ch := range cs.DrawioRelationshipChanges {
				if ch.Type == Modified && ch.Field == fieldLabel {
					got = append(got, ch)
				}
			}
			if len(got) != 1 || got[0].NewValue != "edited" {
				t.Fatalf("want one label change to %q, got %+v (all: %+v)", "edited", got, cs.DrawioRelationshipChanges)
			}
		})
	}
}

// TestRun_ReverseEditIsWrittenToEveryPage: reverse sync took the edit of one
// page over into the model; the other pages must show it at the end of the same
// run. Otherwise they would be stale, and the next sync would read their old
// value as an edit of its own and change the model back (ping-pong).
func TestRun_ReverseEditIsWrittenToEveryPage(t *testing.T) {
	m, state := shopModelAndState("Shop")
	doc := docWithElemOnPages("Shop", "Shop EDIT", "Shop")
	ts := minimalTemplates(t)

	Run(m, doc, state, ts, nil)

	for _, name := range []string{"view-a", "view-b", "view-c"} {
		page := doc.GetPage(name)
		obj := page.FindElement("shop")
		if obj == nil {
			t.Fatalf("%s lost the element", name)
		}
		if title, _, _ := page.ReadElementFields(obj); title != "Shop EDIT" {
			t.Errorf("%s still shows %q after the run, want %q", name, title, "Shop EDIT")
		}
	}

	// The next sync, with the state of the run, is a no-op: nothing flips back.
	state = elementDeleteState(t, m, doc)
	result := Run(m, doc, state, ts, nil)
	if r := result.Reverse; r.ElementsUpdated+r.ElementsCreated+r.ElementsDeleted != 0 || m.Model["shop"].Title != "Shop EDIT" {
		t.Errorf("the next sync changed the model: %+v, title %q", r, m.Model["shop"].Title)
	}
}

// Same for the label of a connector drawn on several pages.
func TestRun_ReverseConnectorLabelEditIsWrittenToEveryPage(t *testing.T) {
	m := &model.BausteinsichtModel{
		Model:         map[string]model.Element{"a": {Kind: "container", Title: "a"}, "b": {Kind: "container", Title: "b"}},
		Relationships: []model.Relationship{{From: "a", To: "b", Label: "calls"}},
		Views:         map[string]model.View{"a": {Title: "A", Include: []string{"*"}}, "b": {Title: "B", Include: []string{"*"}}},
	}
	state := emptyState()
	state.Elements["a"] = ElementState{Title: "a", Kind: "container"}
	state.Elements["b"] = ElementState{Title: "b", Kind: "container"}
	state.Relationships = []RelationshipState{{From: "a", To: "b", Index: 0, Label: "calls"}}
	doc := docWithConnectorOnPages("calls", "edited")

	Run(m, doc, state, minimalTemplates(t), nil)

	if got := m.Relationships[0].Label; got != "edited" {
		t.Fatalf("the model label is %q, want %q", got, "edited")
	}
	for _, name := range []string{"view-a", "view-b"} {
		c := doc.GetPage(name).FindConnector(name[len("view-"):]+"--a", name[len("view-"):]+"--b", 0)
		if c == nil || c.SelectAttrValue("value", "") != "edited" {
			t.Errorf("%s still shows the old label", name)
		}
	}
}
