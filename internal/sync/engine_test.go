package sync

import (
	"strings"
	"testing"

	"github.com/docToolchain/Bauteinsicht/internal/drawio"
	"github.com/docToolchain/Bauteinsicht/internal/model"
)

// --- Test 1: Empty sync (no changes) → zero counts ---

func TestRun_NoChanges(t *testing.T) {
	m := simpleModel("api", "API", "A service", "Go")
	doc := docWithElem("api", "API", "Go", "A service")
	state := stateWithElem("api", "API", "A service", "Go")
	ts := minimalTemplates(t)

	result := Run(m, doc, state, ts)

	if result.Forward.ElementsCreated != 0 {
		t.Errorf("expected 0 elements created, got %d", result.Forward.ElementsCreated)
	}
	if result.Forward.ElementsUpdated != 0 {
		t.Errorf("expected 0 elements updated, got %d", result.Forward.ElementsUpdated)
	}
	if result.Reverse.ElementsUpdated != 0 {
		t.Errorf("expected 0 elements updated in reverse, got %d", result.Reverse.ElementsUpdated)
	}
	if len(result.Conflicts) != 0 {
		t.Errorf("expected no conflicts, got %d", len(result.Conflicts))
	}
}

// --- Test 2: Model added element → forward creates it in doc ---

func TestRun_ModelAddedElement(t *testing.T) {
	m := modelWithElem("svc", "container", "Service")
	doc := emptyDoc()
	state := emptyState()
	ts := minimalTemplates(t)

	result := Run(m, doc, state, ts)

	if result.Forward.ElementsCreated != 1 {
		t.Errorf("expected 1 element created, got %d", result.Forward.ElementsCreated)
	}
	if result.Reverse.ElementsCreated != 0 {
		t.Errorf("expected no reverse creates, got %d", result.Reverse.ElementsCreated)
	}
	if len(result.Conflicts) != 0 {
		t.Errorf("expected no conflicts, got %d", len(result.Conflicts))
	}

	// Verify element exists in doc
	pages := doc.Pages()
	if len(pages) == 0 {
		t.Fatal("expected at least one page in doc")
	}
	elems := pages[0].FindAllElements()
	found := false
	for _, e := range elems {
		if e.SelectAttrValue("bausteinsicht_id", "") == "svc" {
			found = true
		}
	}
	if !found {
		t.Error("element 'svc' not found in doc after forward sync")
	}
}

// --- Test 3: draw.io modified title → reverse updates model ---

func TestRun_DrawioModifiedTitle(t *testing.T) {
	// Model and state agree; draw.io has a different title.
	m := simpleModel("api", "Old Title", "", "")
	doc := docWithElem("api", "New Title", "", "")
	state := stateWithElem("api", "Old Title", "", "")
	ts := minimalTemplates(t)

	result := Run(m, doc, state, ts)

	if result.Reverse.ElementsUpdated != 1 {
		t.Errorf("expected 1 element updated in reverse, got %d", result.Reverse.ElementsUpdated)
	}
	if result.Forward.ElementsUpdated != 0 {
		t.Errorf("expected 0 forward updates (no model change), got %d", result.Forward.ElementsUpdated)
	}
	if len(result.Conflicts) != 0 {
		t.Errorf("expected no conflicts, got %d", len(result.Conflicts))
	}

	// Model should now have the updated title.
	elem := m.Model["api"]
	if elem.Title != "New Title" {
		t.Errorf("expected model title 'New Title', got %q", elem.Title)
	}
}

// --- Test 4: Conflict → model wins, warning generated ---

func TestRun_ConflictModelWins(t *testing.T) {
	// Both model and draw.io changed title compared to last sync.
	m := simpleModel("api", "Model Title", "", "")
	doc := docWithElem("api", "DrawIO Title", "", "")
	state := stateWithElem("api", "Old Title", "", "")
	ts := minimalTemplates(t)

	result := Run(m, doc, state, ts)

	if len(result.Conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(result.Conflicts))
	}
	if result.Conflicts[0].Winner != "model" {
		t.Errorf("expected model to win, got %q", result.Conflicts[0].Winner)
	}

	// Warning must be present.
	if len(result.Warnings) == 0 {
		t.Error("expected at least one warning for conflict")
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "Conflict") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected conflict warning message, got %v", result.Warnings)
	}

	// draw.io change for that field must NOT have been applied to model.
	elem := m.Model["api"]
	if elem.Title != "Model Title" {
		t.Errorf("expected model title to remain 'Model Title', got %q", elem.Title)
	}
}

// --- Test 5: Full round-trip with views (regression test for #83) ---
//
// Verifies that after a forward sync with views, a second sync detects no
// phantom changes. Scoped cell IDs on connectors must be resolved back to
// element IDs so the state comparison works correctly.

func TestRun_ViewBasedSyncNoPhantomChanges(t *testing.T) {
	ts := minimalTemplates(t)

	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"customer": {Kind: "container", Title: "Customer"},
			"shop":     {Kind: "container", Title: "Shop"},
		},
		Relationships: []model.Relationship{
			{From: "customer", To: "shop", Label: "uses"},
		},
		Views: map[string]model.View{
			"ctx": {Title: "Context", Include: []string{"customer", "shop"}},
		},
	}

	// Build empty doc with the view page.
	doc := drawio.NewDocument()
	doc.AddPage("view-ctx", "Context")

	state := emptyState()

	// Round 1: forward sync populates the doc.
	r1 := Run(m, doc, state, ts)
	if r1.Forward.ElementsCreated != 2 {
		t.Fatalf("round 1: expected 2 elements created, got %d", r1.Forward.ElementsCreated)
	}
	if r1.Forward.ConnectorsCreated != 1 {
		t.Fatalf("round 1: expected 1 connector created, got %d", r1.Forward.ConnectorsCreated)
	}

	// Build state after round 1 (mirrors what BuildState does).
	state1 := &SyncState{
		Elements: map[string]ElementState{
			"customer": {Title: "Customer", Kind: "container"},
			"shop":     {Title: "Shop", Kind: "container"},
		},
		Relationships: []RelationshipState{
			{From: "customer", To: "shop", Label: "uses"},
		},
	}

	// Round 2: nothing changed — sync should be a complete no-op.
	r2 := Run(m, doc, state1, ts)

	if r2.Forward.ElementsCreated != 0 || r2.Forward.ElementsUpdated != 0 || r2.Forward.ElementsDeleted != 0 {
		t.Errorf("round 2: expected no forward element changes, got created=%d updated=%d deleted=%d",
			r2.Forward.ElementsCreated, r2.Forward.ElementsUpdated, r2.Forward.ElementsDeleted)
	}
	if r2.Forward.ConnectorsCreated != 0 || r2.Forward.ConnectorsUpdated != 0 || r2.Forward.ConnectorsDeleted != 0 {
		t.Errorf("round 2: expected no forward connector changes, got created=%d updated=%d deleted=%d",
			r2.Forward.ConnectorsCreated, r2.Forward.ConnectorsUpdated, r2.Forward.ConnectorsDeleted)
	}
	if r2.Reverse.ElementsCreated != 0 || r2.Reverse.ElementsUpdated != 0 || r2.Reverse.ElementsDeleted != 0 {
		t.Errorf("round 2: expected no reverse element changes, got created=%d updated=%d deleted=%d",
			r2.Reverse.ElementsCreated, r2.Reverse.ElementsUpdated, r2.Reverse.ElementsDeleted)
	}
	if r2.Reverse.RelationshipsCreated != 0 || r2.Reverse.RelationshipsDeleted != 0 {
		t.Errorf("round 2: expected no reverse relationship changes, got created=%d deleted=%d",
			r2.Reverse.RelationshipsCreated, r2.Reverse.RelationshipsDeleted)
	}
	if len(r2.Conflicts) != 0 {
		t.Errorf("round 2: expected no conflicts, got %d", len(r2.Conflicts))
	}
}

// --- Test 6: Full round-trip ---
//
// Round 1: model has one element, doc is empty, state is empty → forward populates doc.
// Build new state manually.
// Round 2: modify model title → second sync detects change and applies it forward.

func TestRun_FullRoundTrip(t *testing.T) {
	ts := minimalTemplates(t)

	// Round 1: first sync populates an empty doc.
	m := modelWithElem("svc", "container", "Service v1")
	doc := emptyDoc()
	state := emptyState()

	r1 := Run(m, doc, state, ts)
	if r1.Forward.ElementsCreated != 1 {
		t.Fatalf("round 1: expected 1 element created, got %d", r1.Forward.ElementsCreated)
	}

	// Build state after round 1 (in-memory, without file I/O).
	state1 := &SyncState{
		Elements: map[string]ElementState{
			"svc": {Title: "Service v1", Kind: "container"},
		},
		Relationships: []RelationshipState{},
	}

	// Round 2: modify model title → forward update expected.
	updated := m.Model["svc"]
	updated.Title = "Service v2"
	m.Model["svc"] = updated

	r2 := Run(m, doc, state1, ts)
	if r2.Forward.ElementsUpdated != 1 {
		t.Errorf("round 2: expected 1 element updated, got %d", r2.Forward.ElementsUpdated)
	}
	if r2.Forward.ElementsCreated != 0 {
		t.Errorf("round 2: expected no new elements, got %d", r2.Forward.ElementsCreated)
	}
	if len(r2.Conflicts) != 0 {
		t.Errorf("round 2: expected no conflicts, got %d", len(r2.Conflicts))
	}
}
