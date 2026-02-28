package sync

import (
	"strings"
	"testing"
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

// --- Test 5: Full round-trip ---
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
	m.Model["svc"] = m.Model["svc"] // ensure map has the element
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
