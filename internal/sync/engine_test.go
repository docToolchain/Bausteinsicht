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

	result := Run(m, doc, state, ts, nil)

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

	result := Run(m, doc, state, ts, nil)

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

	result := Run(m, doc, state, ts, nil)

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

	result := Run(m, doc, state, ts, nil)

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
	r1 := Run(m, doc, state, ts, nil)
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
	r2 := Run(m, doc, state1, ts, nil)

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

// --- Test 5b: Orphaned view pages removed when view is deleted (#143) ---
//
// Round 1: model has 2 views → creates 2 pages.
// Round 2: model has 1 view → orphaned page should be removed.

func TestRun_OrphanedViewPageRemoved(t *testing.T) {
	ts := minimalTemplates(t)

	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"customer": {Kind: "container", Title: "Customer"},
			"shop":     {Kind: "container", Title: "Shop"},
		},
		Relationships: []model.Relationship{},
		Views: map[string]model.View{
			"ctx":    {Title: "Context", Include: []string{"customer", "shop"}},
			"detail": {Title: "Detail", Include: []string{"shop"}},
		},
	}

	// Build doc with pages for both views.
	doc := drawio.NewDocument()
	doc.AddPage("view-ctx", "Context")
	doc.AddPage("view-detail", "Detail")

	state := emptyState()

	// Round 1: forward sync populates both pages.
	r1 := Run(m, doc, state, ts, nil)
	if r1.Forward.ElementsCreated < 2 {
		t.Fatalf("round 1: expected at least 2 elements created, got %d", r1.Forward.ElementsCreated)
	}

	// Verify both pages exist.
	if len(doc.Pages()) != 2 {
		t.Fatalf("round 1: expected 2 pages, got %d", len(doc.Pages()))
	}

	// Round 2: remove the "detail" view from the model.
	delete(m.Views, "detail")

	// Remove the orphaned page — this is what the sync command should do.
	// We call RemoveOrphanedViewPages to simulate the sync command behavior.
	RemoveOrphanedViewPages(doc, m)

	// The doc should now have only 1 page.
	if len(doc.Pages()) != 1 {
		t.Errorf("after removing orphaned pages: expected 1 page, got %d", len(doc.Pages()))
	}

	// The remaining page should be the "Context" page.
	if doc.GetPage("view-ctx") == nil {
		t.Error("context page should still exist")
	}
	if doc.GetPage("view-detail") != nil {
		t.Error("detail page should be removed (orphaned)")
	}
}

// --- Test 5c: Non-view pages preserved when removing orphans (#143) ---
//
// Verifies that pages not managed by bausteinsicht (e.g., default template
// pages) are NOT removed when cleaning up orphaned view pages.

func TestRun_NonViewPagesPreserved(t *testing.T) {
	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"customer": {Kind: "container", Title: "Customer"},
		},
		Relationships: []model.Relationship{},
		Views: map[string]model.View{
			"ctx": {Title: "Context", Include: []string{"customer"}},
		},
	}

	// Build doc with a default page (not a view page) and a view page.
	doc := drawio.NewDocument()
	doc.AddPage("default-page", "Welcome")
	doc.AddPage("view-ctx", "Context")

	RemoveOrphanedViewPages(doc, m)

	// Both pages should still exist — the default page is not a view page.
	if len(doc.Pages()) != 2 {
		t.Errorf("expected 2 pages (default + view), got %d", len(doc.Pages()))
	}
	if doc.GetPage("default-page") == nil {
		t.Error("default page should be preserved (not a view page)")
	}
	if doc.GetPage("view-ctx") == nil {
		t.Error("context view page should be preserved")
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

	r1 := Run(m, doc, state, ts, nil)
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

	r2 := Run(m, doc, state1, ts, nil)
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

// --- Test 6b: View filter change must NOT delete model elements (#167) ---
//
// Scenario:
// Round 1: model has customer, webshop, webshop.api. View includes ["**"].
//          Sync populates draw.io with all elements.
// Round 2: view filter changes to ["webshop.*"]. Forward sync removes customer
//          from the draw.io page (reconcileViewPage). Reverse sync must NOT
//          delete customer from the model — it was removed because of the filter
//          change, not because the user deleted it in draw.io.

func TestRun_ViewFilterChangeDoesNotDeleteModelElements(t *testing.T) {
	ts := minimalTemplates(t)

	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"customer": {Kind: "container", Title: "Customer"},
			"webshop": {Kind: "container", Title: "Webshop", Children: map[string]model.Element{
				"api": {Kind: "container", Title: "API"},
			}},
		},
		Relationships: []model.Relationship{
			{From: "customer", To: "webshop", Label: "uses"},
		},
		Views: map[string]model.View{
			"main": {Title: "Main", Include: []string{"**"}},
		},
	}

	// Build doc with view page.
	doc := drawio.NewDocument()
	doc.AddPage("view-main", "Main")

	state := emptyState()

	// --- Round 1: initial sync with include ["**"] ---
	r1 := Run(m, doc, state, ts, nil)
	if r1.Forward.ElementsCreated < 3 {
		t.Fatalf("round 1: expected at least 3 elements created, got %d", r1.Forward.ElementsCreated)
	}

	// Build state after round 1 (simulating BuildState which records ALL model elements).
	state1 := &SyncState{
		Elements: map[string]ElementState{
			"customer":    {Title: "Customer", Kind: "container"},
			"webshop":     {Title: "Webshop", Kind: "container"},
			"webshop.api": {Title: "API", Kind: "container"},
		},
		Relationships: []RelationshipState{
			{From: "customer", To: "webshop", Label: "uses"},
		},
	}

	// --- Round 2: narrow the view filter to ["webshop.*"] ---
	m.Views["main"] = model.View{Title: "Main", Include: []string{"webshop.*"}}

	r2 := Run(m, doc, state1, ts, nil)

	// Forward sync should remove customer from the page (reconcileViewPage).
	// That's expected and correct.

	// CRITICAL: customer must still be in the model after reverse sync.
	// The reverse sync must NOT interpret the absence of customer from draw.io
	// as a user deletion.
	if _, ok := m.Model["customer"]; !ok {
		t.Error("CRITICAL: customer was deleted from model after view filter change — this is the #167 bug")
	}

	// The relationship should also still exist.
	if len(m.Relationships) == 0 {
		t.Error("CRITICAL: relationship was deleted from model after view filter change")
	}

	// webshop should still be in the model.
	if _, ok := m.Model["webshop"]; !ok {
		t.Error("webshop was deleted from model")
	}

	// Reverse sync should report 0 element deletions.
	if r2.Reverse.ElementsDeleted != 0 {
		t.Errorf("expected 0 reverse element deletions, got %d", r2.Reverse.ElementsDeleted)
	}
}

// --- Test 6b2: View filter change must NOT delete relationships (#167) ---
//
// This is the relationship-side version of the #167 bug. When a view filter
// narrows and forward sync removes a connector (because an endpoint is no longer
// visible), the reverse sync must NOT delete the relationship from the model.
// The relationship is still valid — it's just not visible in the current view.

func TestRun_ViewFilterChangeDoesNotDeleteRelationships(t *testing.T) {
	ts := minimalTemplates(t)

	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"customer": {Kind: "container", Title: "Customer"},
			"webshop": {Kind: "container", Title: "Webshop", Children: map[string]model.Element{
				"api": {Kind: "container", Title: "API"},
			}},
		},
		Relationships: []model.Relationship{
			{From: "customer", To: "webshop", Label: "uses"},
		},
		Views: map[string]model.View{
			"main": {Title: "Main", Include: []string{"**"}},
		},
	}

	// Build doc with view page.
	doc := drawio.NewDocument()
	doc.AddPage("view-main", "Main")

	state := emptyState()

	// --- Round 1: initial sync with include ["**"] ---
	r1 := Run(m, doc, state, ts, nil)
	if r1.Forward.ElementsCreated < 3 {
		t.Fatalf("round 1: expected at least 3 elements created, got %d", r1.Forward.ElementsCreated)
	}
	if r1.Forward.ConnectorsCreated != 1 {
		t.Fatalf("round 1: expected 1 connector created, got %d", r1.Forward.ConnectorsCreated)
	}

	// Build state after round 1.
	state1 := &SyncState{
		Elements: map[string]ElementState{
			"customer":    {Title: "Customer", Kind: "container"},
			"webshop":     {Title: "Webshop", Kind: "container"},
			"webshop.api": {Title: "API", Kind: "container"},
		},
		Relationships: []RelationshipState{
			{From: "customer", To: "webshop", Label: "uses"},
		},
	}

	// --- Round 2: narrow the view filter to ["webshop.*"] ---
	// This removes customer from the page AND its connector customer→webshop.
	m.Views["main"] = model.View{Title: "Main", Include: []string{"webshop.*"}}

	r2 := Run(m, doc, state1, ts, nil)

	// Forward sync should remove customer and the connector from the page.

	// Build state after round 2.
	state2 := &SyncState{
		Elements: map[string]ElementState{
			"customer":    {Title: "Customer", Kind: "container"},
			"webshop":     {Title: "Webshop", Kind: "container"},
			"webshop.api": {Title: "API", Kind: "container"},
		},
		Relationships: []RelationshipState{
			{From: "customer", To: "webshop", Label: "uses"},
		},
	}

	// --- Round 3: sync again — the connector is gone from draw.io ---
	r3 := Run(m, doc, state2, ts, nil)

	// CRITICAL: customer must still be in the model.
	if _, ok := m.Model["customer"]; !ok {
		t.Error("CRITICAL: customer was deleted from model after view filter change — #167 bug (element)")
	}

	// CRITICAL: the relationship must still be in the model.
	if len(m.Relationships) == 0 {
		t.Error("CRITICAL: relationship customer→webshop was deleted from model after view filter change — #167 bug (relationship)")
	}

	// Reverse sync should report 0 deletions.
	if r2.Reverse.ElementsDeleted != 0 {
		t.Errorf("round 2: expected 0 reverse element deletions, got %d", r2.Reverse.ElementsDeleted)
	}
	if r2.Reverse.RelationshipsDeleted != 0 {
		t.Errorf("round 2: expected 0 reverse relationship deletions, got %d", r2.Reverse.RelationshipsDeleted)
	}
	if r3.Reverse.ElementsDeleted != 0 {
		t.Errorf("round 3: expected 0 reverse element deletions, got %d", r3.Reverse.ElementsDeleted)
	}
	if r3.Reverse.RelationshipsDeleted != 0 {
		t.Errorf("round 3: expected 0 reverse relationship deletions, got %d", r3.Reverse.RelationshipsDeleted)
	}

	_ = r2
	_ = r3
}

// --- Test 6c: User deletion in draw.io still works with views (#167) ---
//
// Verifies that when a user manually deletes an element from draw.io (while
// the element IS still in the view's filter), the reverse sync correctly
// deletes it from the model. This must continue to work after the #167 fix.

func TestRun_UserDeletionInDrawioStillWorksWithViews(t *testing.T) {
	ts := minimalTemplates(t)

	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"customer": {Kind: "container", Title: "Customer"},
			"webshop":  {Kind: "container", Title: "Webshop"},
		},
		Relationships: []model.Relationship{},
		Views: map[string]model.View{
			"main": {Title: "Main", Include: []string{"customer", "webshop"}},
		},
	}

	// Build doc with view page containing both elements.
	doc := drawio.NewDocument()
	page := doc.AddPage("view-main", "Main")
	_ = page.CreateElement(drawio.ElementData{
		ID: "customer", CellID: "main--customer", Kind: "container", Title: "Customer",
	}, "")
	_ = page.CreateElement(drawio.ElementData{
		ID: "webshop", CellID: "main--webshop", Kind: "container", Title: "Webshop",
	}, "")

	// State from previous sync includes both elements.
	// Both were rendered to draw.io pages during the last sync.
	state := &SyncState{
		Elements: map[string]ElementState{
			"customer": {Title: "Customer", Kind: "container"},
			"webshop":  {Title: "Webshop", Kind: "container"},
		},
		Relationships:    []RelationshipState{},
		RenderedElements: map[string]bool{"customer": true, "webshop": true},
	}

	// User manually deletes "customer" from draw.io.
	page.DeleteElement("customer")

	// Run sync — reverse should detect and apply the deletion.
	r := Run(m, doc, state, ts, nil)

	// customer should be deleted from the model (user intended this).
	if _, ok := m.Model["customer"]; ok {
		t.Error("customer should have been deleted from model (user deleted it in draw.io)")
	}

	if r.Reverse.ElementsDeleted != 1 {
		t.Errorf("expected 1 reverse element deletion, got %d", r.Reverse.ElementsDeleted)
	}

	// webshop should still be in the model.
	if _, ok := m.Model["webshop"]; !ok {
		t.Error("webshop should still be in the model")
	}
}

// --- Test 7: Multiple relationships between same pair (#142) ---
//
// Verifies that two relationships from A to B with different labels both
// survive a full sync round-trip and produce two distinct connectors.

func TestRun_MultipleRelsSamePair(t *testing.T) {
	ts := minimalTemplates(t)

	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"a": {Kind: "container", Title: "A"},
			"b": {Kind: "container", Title: "B"},
		},
		Relationships: []model.Relationship{
			{From: "a", To: "b", Label: "uses"},
			{From: "a", To: "b", Label: "calls"},
		},
	}

	doc := emptyDoc()
	state := emptyState()

	// Round 1: first sync populates an empty doc.
	r1 := Run(m, doc, state, ts, nil)
	if r1.Forward.ElementsCreated != 2 {
		t.Fatalf("round 1: expected 2 elements created, got %d", r1.Forward.ElementsCreated)
	}
	if r1.Forward.ConnectorsCreated != 2 {
		t.Fatalf("round 1: expected 2 connectors created, got %d", r1.Forward.ConnectorsCreated)
	}

	// Verify both connectors exist on the page.
	page := requireFirstPage(t, doc)
	conns := page.FindAllConnectors()
	if len(conns) != 2 {
		t.Fatalf("round 1: expected 2 connectors on page, got %d", len(conns))
	}

	// Build state after round 1.
	state1 := &SyncState{
		Elements: map[string]ElementState{
			"a": {Title: "A", Kind: "container"},
			"b": {Title: "B", Kind: "container"},
		},
		Relationships: []RelationshipState{
			{From: "a", To: "b", Index: 0, Label: "uses"},
			{From: "a", To: "b", Index: 1, Label: "calls"},
		},
	}

	// Round 2: nothing changed — sync should be a complete no-op.
	r2 := Run(m, doc, state1, ts, nil)

	if r2.Forward.ConnectorsCreated != 0 || r2.Forward.ConnectorsUpdated != 0 || r2.Forward.ConnectorsDeleted != 0 {
		t.Errorf("round 2: expected no forward connector changes, got created=%d updated=%d deleted=%d",
			r2.Forward.ConnectorsCreated, r2.Forward.ConnectorsUpdated, r2.Forward.ConnectorsDeleted)
	}
	if r2.Reverse.RelationshipsCreated != 0 || r2.Reverse.RelationshipsUpdated != 0 || r2.Reverse.RelationshipsDeleted != 0 {
		t.Errorf("round 2: expected no reverse relationship changes, got created=%d updated=%d deleted=%d",
			r2.Reverse.RelationshipsCreated, r2.Reverse.RelationshipsUpdated, r2.Reverse.RelationshipsDeleted)
	}
	if len(r2.Conflicts) != 0 {
		t.Errorf("round 2: expected no conflicts, got %d", len(r2.Conflicts))
	}
}

// --- Test: Sync warns when model elements aren't visible in any view (#183) ---
//
// Model has 3 elements: "a", "b", "c".
// View only includes "a" and "b".
// After sync, result.Warnings should contain a warning about "c" not being
// visible in any view.

func TestRun_WarnsAboutInvisibleElements(t *testing.T) {
	ts := minimalTemplates(t)

	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"a": {Kind: "container", Title: "A"},
			"b": {Kind: "container", Title: "B"},
			"c": {Kind: "container", Title: "C"},
		},
		Relationships: []model.Relationship{},
		Views: map[string]model.View{
			"v1": {Title: "V1", Include: []string{"a", "b"}},
		},
	}

	doc := drawio.NewDocument()
	doc.AddPage("view-v1", "V1")
	state := emptyState()

	result := Run(m, doc, state, ts, nil)

	// Elements "a" and "b" should be created on the page.
	if result.Forward.ElementsCreated != 2 {
		t.Errorf("expected 2 elements created, got %d", result.Forward.ElementsCreated)
	}

	// There should be a warning about "c" not being visible in any view.
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "c") && strings.Contains(w, "not visible") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning about element 'c' not visible in any view, got warnings: %v", result.Warnings)
	}
}

// Test: No warning about invisible elements when model has no views.
func TestRun_NoWarningWithoutViews(t *testing.T) {
	ts := minimalTemplates(t)

	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"a": {Kind: "container", Title: "A"},
		},
		Relationships: []model.Relationship{},
	}

	doc := emptyDoc()
	state := emptyState()

	result := Run(m, doc, state, ts, nil)

	for _, w := range result.Warnings {
		if strings.Contains(w, "not visible") {
			t.Errorf("unexpected visibility warning without views: %s", w)
		}
	}
}

// --- Test: Adding a new view doesn't delete elements from model (#184) ---
//
// Round 1: model has "a" and "b", view v1 includes only "a".
// Round 2: add view v2 that includes "b".
// Element "b" should NOT be deleted from the model.

func TestRun_AddingNewViewDoesNotDeleteElements(t *testing.T) {
	ts := minimalTemplates(t)

	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"a": {Kind: "container", Title: "A"},
			"b": {Kind: "container", Title: "B"},
		},
		Relationships: []model.Relationship{},
		Views: map[string]model.View{
			"v1": {Title: "V1", Include: []string{"a"}},
		},
	}

	doc := drawio.NewDocument()
	doc.AddPage("view-v1", "V1")
	state := emptyState()

	// Round 1: forward sync populates v1 with "a".
	r1 := Run(m, doc, state, ts, nil)
	if r1.Forward.ElementsCreated != 1 {
		t.Fatalf("round 1: expected 1 element created, got %d", r1.Forward.ElementsCreated)
	}

	// Build state after round 1.
	state1 := &SyncState{
		Elements: map[string]ElementState{
			"a": {Title: "A", Kind: "container"},
			"b": {Title: "B", Kind: "container"},
		},
		Relationships: []RelationshipState{},
	}

	// Round 2: add view v2 that includes "b".
	m.Views["v2"] = model.View{Title: "V2", Include: []string{"b"}}
	newPage := doc.AddPage("view-v2", "V2")
	_ = newPage

	newPageIDs := map[string]bool{"view-v2": true}
	r2 := Run(m, doc, state1, ts, newPageIDs)

	// Element "b" should NOT be deleted by reverse sync.
	if r2.Reverse.ElementsDeleted != 0 {
		t.Errorf("round 2: expected 0 reverse deletions, got %d", r2.Reverse.ElementsDeleted)
	}

	// Element "b" should be created on v2 by forward sync.
	if r2.Forward.ElementsCreated < 1 {
		t.Errorf("round 2: expected at least 1 forward creation (b on v2), got %d", r2.Forward.ElementsCreated)
	}

	// Model should still have "b".
	if _, ok := m.Model["b"]; !ok {
		t.Errorf("element 'b' was deleted from model — data loss!")
	}
}

// --- Test: Renaming a view key doesn't delete elements (#189) ---
//
// Round 1: view "all" includes "a".
// Round 2: view "all" renamed to "overview" (same include).
// Element "a" should NOT be deleted.

func TestRun_RenamingViewDoesNotDeleteElements(t *testing.T) {
	ts := minimalTemplates(t)

	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"a": {Kind: "container", Title: "A"},
		},
		Relationships: []model.Relationship{},
		Views: map[string]model.View{
			"all": {Title: "All", Include: []string{"a"}},
		},
	}

	doc := drawio.NewDocument()
	doc.AddPage("view-all", "All")
	state := emptyState()

	// Round 1: forward sync populates view "all" with "a".
	r1 := Run(m, doc, state, ts, nil)
	if r1.Forward.ElementsCreated != 1 {
		t.Fatalf("round 1: expected 1 element created, got %d", r1.Forward.ElementsCreated)
	}

	state1 := &SyncState{
		Elements: map[string]ElementState{
			"a": {Title: "A", Kind: "container"},
		},
		Relationships: []RelationshipState{},
	}

	// Round 2: rename view "all" → "overview".
	delete(m.Views, "all")
	m.Views["overview"] = model.View{Title: "Overview", Include: []string{"a"}}

	// Remove orphaned page and add new one (simulating what sync.go does).
	RemoveOrphanedViewPages(doc, m)
	doc.AddPage("view-overview", "Overview")

	newPageIDs := map[string]bool{"view-overview": true}
	r2 := Run(m, doc, state1, ts, newPageIDs)

	// "a" should NOT be deleted.
	if r2.Reverse.ElementsDeleted != 0 {
		t.Errorf("round 2: expected 0 reverse deletions, got %d", r2.Reverse.ElementsDeleted)
	}

	if _, ok := m.Model["a"]; !ok {
		t.Errorf("element 'a' was deleted from model — data loss!")
	}
}

// --- Test: New view pages get populated (#188) ---
//
// After adding a new view and syncing, the new page should contain elements.

func TestRun_NewViewPageGetsPopulated(t *testing.T) {
	ts := minimalTemplates(t)

	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"a": {Kind: "container", Title: "A"},
			"b": {Kind: "container", Title: "B"},
		},
		Relationships: []model.Relationship{},
		Views: map[string]model.View{
			"v1": {Title: "V1", Include: []string{"a"}},
		},
	}

	doc := drawio.NewDocument()
	doc.AddPage("view-v1", "V1")
	state := emptyState()

	// Round 1: populate v1.
	r1 := Run(m, doc, state, ts, nil)
	_ = r1

	state1 := &SyncState{
		Elements: map[string]ElementState{
			"a": {Title: "A", Kind: "container"},
			"b": {Title: "B", Kind: "container"},
		},
		Relationships: []RelationshipState{},
	}

	// Round 2: add v2.
	m.Views["v2"] = model.View{Title: "V2", Include: []string{"b"}}
	doc.AddPage("view-v2", "V2")

	newPageIDs := map[string]bool{"view-v2": true}
	r2 := Run(m, doc, state1, ts, newPageIDs)

	// "b" should be created on v2.
	if r2.Forward.ElementsCreated < 1 {
		t.Errorf("expected at least 1 forward creation, got %d", r2.Forward.ElementsCreated)
	}

	// Verify "b" is actually on v2 page.
	v2Page := requirePage(t, doc, "view-v2")
	if v2Page == nil {
		t.Fatal("v2 page not found")
	}
	if v2Page.FindElement("b") == nil {
		t.Error("element 'b' not found on v2 page")
	}
}
