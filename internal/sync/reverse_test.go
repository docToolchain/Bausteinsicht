package sync

import (
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/drawio"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// modelWithRel builds a model with one element and one relationship.
func modelWithRel(fromID, toID, label string) *model.BausteinsichtModel {
	return &model.BausteinsichtModel{
		Model: map[string]model.Element{
			fromID: {Kind: "container", Title: fromID},
			toID:   {Kind: "container", Title: toID},
		},
		Relationships: []model.Relationship{
			{From: fromID, To: toID, Label: label},
		},
	}
}

// modelWithChild builds a model with a nested child element.
func modelWithChild(parentID, childKey string) *model.BausteinsichtModel {
	return &model.BausteinsichtModel{
		Model: map[string]model.Element{
			parentID: {
				Kind:  "container",
				Title: parentID,
				Children: map[string]model.Element{
					childKey: {Kind: "component", Title: "child-title"},
				},
			},
		},
		Relationships: []model.Relationship{},
	}
}

// changeSet builds a ChangeSet with a single DrawioElementChange.
func elemChangeSet(id string, ct ChangeType, field, oldVal, newVal string) *ChangeSet {
	return &ChangeSet{
		DrawioElementChanges: []ElementChange{
			{ID: id, Type: ct, Field: field, OldValue: oldVal, NewValue: newVal},
		},
	}
}

// relChangeSet builds a ChangeSet with a single DrawioRelationshipChange.
func relChangeSet(from, to string, ct ChangeType, field, oldVal, newVal string) *ChangeSet {
	return &ChangeSet{
		DrawioRelationshipChanges: []RelationshipChange{
			{From: from, To: to, Type: ct, Field: field, OldValue: oldVal, NewValue: newVal},
		},
	}
}

// --- Element change tests ---

func TestApplyReverse_TitleUpdate(t *testing.T) {
	m := simpleModel("api", "Old Title", "", "")
	cs := elemChangeSet("api", Modified, "title", "Old Title", "New Title")

	r := ApplyReverse(cs, m, nil)

	if m.Model["api"].Title != "New Title" {
		t.Errorf("expected title %q, got %q", "New Title", m.Model["api"].Title)
	}
	if r.ElementsUpdated != 1 {
		t.Errorf("expected ElementsUpdated=1, got %d", r.ElementsUpdated)
	}
	if len(r.Warnings) != 0 {
		t.Errorf("unexpected warnings: %v", r.Warnings)
	}
}

func TestApplyReverse_DescriptionUpdate(t *testing.T) {
	m := simpleModel("api", "Title", "Old Desc", "")
	cs := elemChangeSet("api", Modified, "description", "Old Desc", "New Desc")

	r := ApplyReverse(cs, m, nil)

	if m.Model["api"].Description != "New Desc" {
		t.Errorf("expected description %q, got %q", "New Desc", m.Model["api"].Description)
	}
	if r.ElementsUpdated != 1 {
		t.Errorf("expected ElementsUpdated=1, got %d", r.ElementsUpdated)
	}
}

func TestApplyReverse_TechnologyUpdate(t *testing.T) {
	m := simpleModel("api", "Title", "", "Go")
	cs := elemChangeSet("api", Modified, "technology", "Go", "Rust")

	r := ApplyReverse(cs, m, nil)

	if m.Model["api"].Technology != "Rust" {
		t.Errorf("expected technology %q, got %q", "Rust", m.Model["api"].Technology)
	}
	if r.ElementsUpdated != 1 {
		t.Errorf("expected ElementsUpdated=1, got %d", r.ElementsUpdated)
	}
}

func TestApplyReverse_NestedElementUpdate(t *testing.T) {
	m := modelWithChild("webshop", "api")
	cs := elemChangeSet("webshop.api", Modified, "title", "child-title", "Updated API")

	r := ApplyReverse(cs, m, nil)

	child := m.Model["webshop"].Children["api"]
	if child.Title != "Updated API" {
		t.Errorf("expected nested title %q, got %q", "Updated API", child.Title)
	}
	if r.ElementsUpdated != 1 {
		t.Errorf("expected ElementsUpdated=1, got %d", r.ElementsUpdated)
	}
}

func TestApplyReverse_ElementDeleted(t *testing.T) {
	m := simpleModel("api", "Title", "", "")
	cs := elemChangeSet("api", Deleted, "", "", "")

	r := ApplyReverse(cs, m, nil)

	if _, exists := m.Model["api"]; exists {
		t.Error("expected element to be removed from model")
	}
	if r.ElementsDeleted != 1 {
		t.Errorf("expected ElementsDeleted=1, got %d", r.ElementsDeleted)
	}
	if len(r.Warnings) != 1 || !strings.Contains(r.Warnings[0], "deleted in draw.io") {
		t.Errorf("expected deletion warning, got %v", r.Warnings)
	}
}

func TestApplyReverse_AddedElementWarning(t *testing.T) {
	m := emptyModel()
	cs := elemChangeSet("unknown", Added, "", "", "")

	r := ApplyReverse(cs, m, nil)

	if len(r.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", len(r.Warnings), r.Warnings)
	}
	// The warning should include the element ID. (#115, #196)
	w := r.Warnings[0]
	if !strings.Contains(w, `"unknown"`) {
		t.Errorf("expected warning to include element ID %q, got: %s", "unknown", w)
	}
	if r.ElementsCreated != 1 {
		t.Errorf("expected ElementsCreated=1, got %d", r.ElementsCreated)
	}
}

func TestApplyReverse_AddedNestedElementSkipped(t *testing.T) {
	// When sync state is reset (e.g. .bausteinsicht-sync deleted), all draw.io
	// elements appear as "Added" from the draw.io side. Nested elements with
	// dot-path IDs (e.g. "parent.child") must NOT be created as spurious
	// top-level entries — they already exist in the model hierarchy. (#307)
	// On first sync (lastState is nil), skip silently without warning.
	m := modelWithChild("parent", "child")
	cs := elemChangeSet("parent.child", Added, "", "", "")

	r := ApplyReverse(cs, m, nil)

	// The nested element must NOT be created as a top-level entry.
	if _, exists := m.Model["parent.child"]; exists {
		t.Error("should not create a spurious top-level entry for a nested element")
	}
	if r.ElementsCreated != 0 {
		t.Errorf("expected ElementsCreated=0, got %d", r.ElementsCreated)
	}
	// On first sync, skip silently without warning (elements exist in both places by design).
	if len(r.Warnings) != 0 {
		t.Errorf("expected no warnings on first sync, got %v", r.Warnings)
	}
}

func TestApplyReverse_ModifyMissingElement(t *testing.T) {
	m := emptyModel()
	cs := elemChangeSet("nonexistent", Modified, "title", "", "New")

	r := ApplyReverse(cs, m, nil)

	if r.ElementsUpdated != 0 {
		t.Errorf("expected no update, got %d", r.ElementsUpdated)
	}
	if len(r.Warnings) == 0 {
		t.Error("expected warning for missing element")
	}
}

// --- Relationship change tests ---

func TestApplyReverse_RelationshipLabelUpdated(t *testing.T) {
	m := modelWithRel("a", "b", "old label")
	cs := relChangeSet("a", "b", Modified, "label", "old label", "new label")

	r := ApplyReverse(cs, m, nil)

	if m.Relationships[0].Label != "new label" {
		t.Errorf("expected label %q, got %q", "new label", m.Relationships[0].Label)
	}
	if r.RelationshipsUpdated != 1 {
		t.Errorf("expected RelationshipsUpdated=1, got %d", r.RelationshipsUpdated)
	}
}

func TestApplyReverse_RelationshipDeleted(t *testing.T) {
	m := modelWithRel("a", "b", "calls")
	cs := relChangeSet("a", "b", Deleted, "", "", "")

	r := ApplyReverse(cs, m, nil)

	if len(m.Relationships) != 0 {
		t.Errorf("expected relationship to be removed, got %d remaining", len(m.Relationships))
	}
	if r.RelationshipsDeleted != 1 {
		t.Errorf("expected RelationshipsDeleted=1, got %d", r.RelationshipsDeleted)
	}
}

func TestApplyReverse_DeleteElementCleansViewIncludes(t *testing.T) {
	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"api":    {Kind: "container", Title: "API"},
			"webapp": {Kind: "container", Title: "WebApp"},
		},
		Relationships: []model.Relationship{},
		Views: map[string]model.View{
			"overview": {
				Title:   "Overview",
				Include: []string{"api", "webapp"},
				Exclude: []string{"api"},
			},
		},
	}

	cs := elemChangeSet("api", Deleted, "", "", "")
	ApplyReverse(cs, m, nil)

	v := m.Views["overview"]
	for _, inc := range v.Include {
		if inc == "api" {
			t.Error("deleted element 'api' should be removed from view Include")
		}
	}
	for _, exc := range v.Exclude {
		if exc == "api" {
			t.Error("deleted element 'api' should be removed from view Exclude")
		}
	}
	// "webapp" should remain
	found := false
	for _, inc := range v.Include {
		if inc == "webapp" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'webapp' to remain in Include")
	}
}

func TestApplyReverse_EmptyTitleSkipped(t *testing.T) {
	m := simpleModel("api", "Original Title", "", "")
	cs := elemChangeSet("api", Modified, "title", "Original Title", "")

	r := ApplyReverse(cs, m, nil)

	// Title should remain unchanged
	if m.Model["api"].Title != "Original Title" {
		t.Errorf("expected title to remain %q, got %q", "Original Title", m.Model["api"].Title)
	}
	// Should have a warning
	if len(r.Warnings) == 0 {
		t.Error("expected warning about empty title")
	}
	found := false
	for _, w := range r.Warnings {
		if strings.Contains(w, "empty title") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning containing 'empty title', got: %v", r.Warnings)
	}
	// Should NOT count as an update
	if r.ElementsUpdated != 0 {
		t.Errorf("expected 0 updates (skipped), got %d", r.ElementsUpdated)
	}
}

func TestApplyReverse_RelationshipAdded(t *testing.T) {
	m := modelWithRel("x", "y", "")
	m.Relationships = []model.Relationship{} // clear the initial relationship for this test
	cs := relChangeSet("x", "y", Added, "", "", "uses")

	r := ApplyReverse(cs, m, nil)

	if len(m.Relationships) != 1 {
		t.Fatalf("expected 1 relationship, got %d", len(m.Relationships))
	}
	rel := m.Relationships[0]
	if rel.From != "x" || rel.To != "y" {
		t.Errorf("unexpected relationship: %+v", rel)
	}
	if r.RelationshipsCreated != 1 {
		t.Errorf("expected RelationshipsCreated=1, got %d", r.RelationshipsCreated)
	}
	if len(r.Warnings) != 0 {
		t.Errorf("unexpected warnings: %v", r.Warnings)
	}
}

// TestApplyReverse_RelationshipAddedDuplicateSkipped verifies that adding a
// relationship that already exists in the model is skipped and a warning is
// issued. This prevents duplicate relationships when sync state is stale or
// when the same relationship appears in both model and draw.io. Regression
// test for #547.
func TestApplyReverse_RelationshipAddedDuplicateSkipped(t *testing.T) {
	// Build a model with existing elements and a relationship between them.
	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"frontend": {Kind: "container", Title: "Frontend"},
			"backend":  {Kind: "container", Title: "Backend"},
		},
		Relationships: []model.Relationship{
			{From: "frontend", To: "backend", Label: "calls API"},
		},
	}

	// Simulate "lost sync state" - previous sync had OTHER relationships,
	// but not this one. The relationship was manually added to the model
	// (not synced from draw.io), and now draw.io also has it.
	// This is a genuine conflict: model has relationship that wasn't synced from draw.io.
	lastState := &SyncState{
		Timestamp: "2026-07-26T00:00:00Z", // a prior sync completed
		Relationships: []RelationshipState{
			// Some OTHER relationship that was synced before
			{From: "other", To: "thing"},
		},
	}

	cs := relChangeSet("frontend", "backend", Added, "", "", "calls API")

	r := ApplyReverse(cs, m, lastState)

	// Should still have exactly 1 relationship (no duplicate added).
	if len(m.Relationships) != 1 {
		t.Fatalf("expected 1 relationship (no duplicate), got %d", len(m.Relationships))
	}

	// Should NOT count as created.
	if r.RelationshipsCreated != 0 {
		t.Errorf("expected RelationshipsCreated=0 (duplicate skipped), got %d", r.RelationshipsCreated)
	}

	// Should have a warning about the duplicate (because lastState exists but
	// doesn't include this relationship - it's a genuine conflict).
	if len(r.Warnings) == 0 {
		t.Fatal("expected warning about duplicate relationship, got none")
	}
	found := false
	for _, w := range r.Warnings {
		if strings.Contains(w, "already exists") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning mentioning 'already exists', got: %v", r.Warnings)
	}
}

// TestApplyReverse_RelationshipAddedSecondBetweenSamePair verifies that a
// newly-drawn SECOND relationship between the same pair (#142) is imported even
// when a FIRST relationship between that pair was already synced. Keying the
// last-sync set by "from->to" (ignoring the per-pair index) previously matched
// the already-synced first relationship and dropped the new one silently (#548).
func TestApplyReverse_RelationshipAddedSecondBetweenSamePair(t *testing.T) {
	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"a": {Kind: "container", Title: "A"},
			"b": {Kind: "container", Title: "B"},
		},
		Relationships: []model.Relationship{
			{From: "a", To: "b", Label: "uses"}, // index 0, previously synced
		},
	}
	// Previous sync recorded a->b at index 0 only.
	lastState := &SyncState{
		Timestamp: "2026-07-26T00:00:00Z", // a prior sync completed
		Relationships: []RelationshipState{
			{From: "a", To: "b", Index: 0, Label: "uses"},
		},
	}
	// draw.io now has a SECOND a->b connector (index 1) - a genuinely new relationship.
	cs := &ChangeSet{
		DrawioRelationshipChanges: []RelationshipChange{
			{From: "a", To: "b", Index: 1, Type: Added, NewValue: "calls"},
		},
	}

	r := ApplyReverse(cs, m, lastState)

	if len(m.Relationships) != 2 {
		t.Fatalf("expected 2 relationships (new one imported), got %d", len(m.Relationships))
	}
	if r.RelationshipsCreated != 1 {
		t.Errorf("expected RelationshipsCreated=1, got %d", r.RelationshipsCreated)
	}
	if len(r.Warnings) != 0 {
		t.Errorf("unexpected warnings: %v", r.Warnings)
	}
}

// TestApplyReverse_RelationshipAddedWithUnrelatedAhead exercises the
// global-index-vs-per-pair-count distinction (#548 review): an unrelated
// relationship sits AHEAD of the target pair in m.Relationships, so a per-pair
// count would not equal the global connector index. A genuinely new hand-drawn
// a->b connector (parseConnectorIndex yields 0 for non-"rel-a-b-N" ids) must
// still be imported, not dropped.
func TestApplyReverse_RelationshipAddedWithUnrelatedAhead(t *testing.T) {
	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"x": {Kind: "container", Title: "X"},
			"y": {Kind: "container", Title: "Y"},
			"a": {Kind: "container", Title: "A"},
			"b": {Kind: "container", Title: "B"},
		},
		Relationships: []model.Relationship{
			{From: "x", To: "y", Label: "unrelated"}, // global index 0
			{From: "a", To: "b", Label: "uses"},      // global index 1
		},
	}
	// Prior sync recorded both, a->b at its global index 1.
	lastState := &SyncState{
		Timestamp: "2026-07-26T00:00:00Z",
		Relationships: []RelationshipState{
			{From: "x", To: "y", Index: 0, Label: "unrelated"},
			{From: "a", To: "b", Index: 1, Label: "uses"},
		},
	}
	// A genuinely new, hand-drawn second a->b connector: its id is not a
	// bausteinsicht-generated "rel-a-b-N", so parseConnectorIndex yields 0.
	cs := &ChangeSet{
		DrawioRelationshipChanges: []RelationshipChange{
			{From: "a", To: "b", Index: 0, Type: Added, NewValue: "calls"},
		},
	}

	r := ApplyReverse(cs, m, lastState)

	if len(m.Relationships) != 3 {
		t.Fatalf("expected 3 relationships (new a->b imported), got %d", len(m.Relationships))
	}
	if r.RelationshipsCreated != 1 {
		t.Errorf("expected RelationshipsCreated=1, got %d", r.RelationshipsCreated)
	}
	if len(r.Warnings) != 0 {
		t.Errorf("unexpected warnings: %v", r.Warnings)
	}
}

// TestApplyReverse_RelationshipAddedFirstSyncNoWarning verifies the first-sync
// gate works for real callers (#548 review): production callers pass a non-nil
// but timestamp-less state on the first sync (LoadState fabricates it), so the
// "already exists" conflict warning must be suppressed via the timestamp, not
// via nil-ness. A user who added the same relationship to both the model and
// draw.io on their very first sync should get no warning.
func TestApplyReverse_RelationshipAddedFirstSyncNoWarning(t *testing.T) {
	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"a": {Kind: "container", Title: "A"},
			"b": {Kind: "container", Title: "B"},
		},
		Relationships: []model.Relationship{
			{From: "a", To: "b", Label: "uses"}, // global index 0
		},
	}
	// First sync: non-nil state as LoadState fabricates it, but no timestamp.
	lastState := &SyncState{
		Elements:      map[string]ElementState{},
		Relationships: []RelationshipState{},
		// Timestamp intentionally empty: no prior sync has completed.
	}
	cs := relChangeSet("a", "b", Added, "", "", "uses")

	r := ApplyReverse(cs, m, lastState)

	if len(m.Relationships) != 1 {
		t.Fatalf("expected 1 relationship (no duplicate), got %d", len(m.Relationships))
	}
	if r.RelationshipsCreated != 0 {
		t.Errorf("expected RelationshipsCreated=0, got %d", r.RelationshipsCreated)
	}
	if len(r.Warnings) != 0 {
		t.Errorf("expected no warning on first sync, got %v", r.Warnings)
	}
}

// TestApplyReverse_RelationshipAddedStaleElementsRejected verifies that
// relationships referencing non-existent elements are rejected during reverse sync (#329).
func TestApplyReverse_RelationshipAddedStaleElementsRejected(t *testing.T) {
	m := modelWithRel("x", "y", "")
	m.Relationships = []model.Relationship{} // clear the initial relationship for this test
	cs := relChangeSet("nonexistent", "y", Added, "", "", "uses")

	r := ApplyReverse(cs, m, nil)

	if len(m.Relationships) != 0 {
		t.Fatalf("expected 0 relationships (stale rejected), got %d", len(m.Relationships))
	}
	if r.RelationshipsCreated != 0 {
		t.Errorf("expected RelationshipsCreated=0, got %d", r.RelationshipsCreated)
	}
	if len(r.Warnings) == 0 {
		t.Errorf("expected warning about stale relationship, got none")
	}
	warnText := strings.Join(r.Warnings, "; ")
	if !strings.Contains(warnText, "nonexistent") || !strings.Contains(warnText, "does not exist") {
		t.Errorf("expected warning mentioning non-existent element, got: %s", warnText)
	}
}

// TestApplyReverse_SwapConnectorDirectionPreservesMetadata verifies that
// swapping a connector's source and target in draw.io preserves the
// relationship's kind, label, and description (#185).
func TestApplyReverse_SwapConnectorDirectionPreservesMetadata(t *testing.T) {
	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"a": {Kind: "system", Title: "A"},
			"b": {Kind: "system", Title: "B"},
		},
		Relationships: []model.Relationship{
			{From: "a", To: "b", Kind: "uses", Label: "important-label", Description: "important desc"},
		},
	}

	// Simulate draw.io direction swap: old connector deleted, new one added with swapped endpoints.
	cs := &ChangeSet{
		DrawioRelationshipChanges: []RelationshipChange{
			{From: "a", To: "b", Index: 0, Type: Deleted},
			{From: "b", To: "a", Type: Added},
		},
	}

	result := ApplyReverse(cs, m, nil)

	if len(m.Relationships) != 1 {
		t.Fatalf("expected 1 relationship, got %d", len(m.Relationships))
	}
	rel := m.Relationships[0]
	if rel.From != "b" || rel.To != "a" {
		t.Errorf("expected from=b, to=a; got from=%s, to=%s", rel.From, rel.To)
	}
	if rel.Kind != "uses" {
		t.Errorf("expected kind=uses, got %q", rel.Kind)
	}
	if rel.Label != "important-label" {
		t.Errorf("expected label=important-label, got %q", rel.Label)
	}
	if rel.Description != "important desc" {
		t.Errorf("expected description='important desc', got %q", rel.Description)
	}
	// Should be an update, not a delete+create.
	if result.RelationshipsDeleted != 0 {
		t.Errorf("expected RelationshipsDeleted=0 (swap, not delete), got %d", result.RelationshipsDeleted)
	}
	if result.RelationshipsUpdated != 1 {
		t.Errorf("expected RelationshipsUpdated=1, got %d", result.RelationshipsUpdated)
	}
}

// TestDetectChanges_NewElementFromDrawio verifies that a new shape added in
// draw.io (without bausteinsicht_id) is detected as an Added element change. (#196)
func TestDetectChanges_NewElementFromDrawio(t *testing.T) {
	m := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"system": {Notation: "System"},
			},
		},
		Model: map[string]model.Element{
			"existing": {Kind: "system", Title: "Existing"},
		},
		Views: map[string]model.View{
			"context": {Title: "System Context", Include: []string{"existing"}},
		},
	}

	// Create a drawio doc with the existing element plus a new unmanaged shape.
	doc := drawio.NewDocument()
	doc.AddPage("view-context", "System Context")
	page := requirePage(t, doc, "view-context")
	if err := page.CreateElement(drawio.ElementData{
		ID: "existing", CellID: "context--existing",
		Kind: "system", Title: "Existing",
	}, "shape=test;"); err != nil {
		t.Fatal(err)
	}

	// Add a new shape WITHOUT bausteinsicht_id — simulates user drawing in draw.io.
	root := page.Root()
	obj := root.CreateElement("object")
	obj.CreateAttr("label", "New Service")
	obj.CreateAttr("id", "new-shape-1")
	cell := obj.CreateElement("mxCell")
	cell.CreateAttr("style", "rounded=1;")
	cell.CreateAttr("vertex", "1")
	cell.CreateAttr("parent", "1")

	lastState := &SyncState{
		Elements: map[string]ElementState{
			"existing": {Title: "Existing"},
		},
	}

	cs := DetectChanges(m, doc, lastState, nil)

	// Should detect the new element as a drawio Added change.
	var found bool
	for _, ch := range cs.DrawioElementChanges {
		if ch.Type == Added && ch.NewValue == "New Service" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected Added change for new drawio element 'New Service', got changes: %+v", cs.DrawioElementChanges)
	}
}

// TestApplyReverse_NewElementFromDrawio verifies that a new element detected
// from draw.io is added to the model with an auto-generated ID. (#196)
func TestApplyReverse_NewElementFromDrawio(t *testing.T) {
	m := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"system": {Notation: "System"},
			},
		},
		Model: map[string]model.Element{
			"existing": {Kind: "system", Title: "Existing"},
		},
	}

	cs := &ChangeSet{
		DrawioElementChanges: []ElementChange{
			{
				ID:       "newservice",
				Type:     Added,
				NewValue: "New Service",
			},
		},
	}

	result := ApplyReverse(cs, m, nil)

	// The new element should be added to the model.
	if _, ok := m.Model["newservice"]; !ok {
		t.Error("expected element 'newservice' to be added to model")
	}
	if result.ElementsCreated != 1 {
		t.Errorf("expected ElementsCreated=1, got %d", result.ElementsCreated)
	}
	// Should warn the user to assign a meaningful ID.
	var hasWarning bool
	for _, w := range result.Warnings {
		if strings.Contains(w, "newservice") {
			hasWarning = true
			break
		}
	}
	if !hasWarning {
		t.Error("expected warning about the auto-generated ID")
	}
}

// TestApplyReverse_NewElementCollidingIDSkipped verifies that a new element
// from draw.io whose auto-generated ID collides with an existing model element
// is NOT imported and a warning is issued instead of overwriting (#203).
// On first sync (lastState is nil), skip silently - element exists in both places by design.
// On subsequent sync, warn about genuine conflicts (element in model wasn't synced from draw.io).
func TestApplyReverse_NewElementCollidingIDSkipped(t *testing.T) {
	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"api": {Kind: "container", Title: "API Gateway", Description: "Main service", Technology: "Go"},
		},
	}

	// Simulate "lost sync state" - previous sync had OTHER elements,
	// but not this one. The element was manually added to the model
	// (not synced from draw.io), and now draw.io also has it.
	// This is a genuine conflict: model has element that wasn't synced from draw.io.
	lastState := &SyncState{
		Timestamp: "2026-07-26T00:00:00Z", // a prior sync completed
		Elements: map[string]ElementState{
			// Some OTHER element that was synced before
			"other": {Title: "Other", Kind: "system"},
		},
	}

	cs := &ChangeSet{
		DrawioElementChanges: []ElementChange{
			{ID: "api", Type: Added, NewValue: "API"},
		},
	}

	result := ApplyReverse(cs, m, lastState)

	// Existing element must NOT be overwritten.
	elem := m.Model["api"]
	if elem.Title != "API Gateway" {
		t.Errorf("existing element overwritten: title=%q, want %q", elem.Title, "API Gateway")
	}
	if elem.Description != "Main service" {
		t.Errorf("existing element overwritten: description=%q, want %q", elem.Description, "Main service")
	}
	if elem.Technology != "Go" {
		t.Errorf("existing element overwritten: technology=%q, want %q", elem.Technology, "Go")
	}
	// Should NOT count as created.
	if result.ElementsCreated != 0 {
		t.Errorf("expected ElementsCreated=0, got %d", result.ElementsCreated)
	}
	// Should have a warning about the collision (because lastState exists but
	// doesn't include this element - it's a genuine conflict).
	var found bool
	for _, w := range result.Warnings {
		if strings.Contains(w, "api") && strings.Contains(w, "already exists") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected collision warning mentioning 'already exists', got: %v", result.Warnings)
	}
}

// TestApplyReverse_NewElementGetsDefaultKind verifies that a new element
// imported from draw.io receives a default kind from the specification
// rather than an empty string (#206).
func TestApplyReverse_NewElementGetsDefaultKind(t *testing.T) {
	m := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"system":    {Notation: "System"},
				"container": {Notation: "Container"},
			},
		},
		Model: map[string]model.Element{},
	}

	cs := &ChangeSet{
		DrawioElementChanges: []ElementChange{
			{ID: "newservice", Type: Added, NewValue: "New Service"},
		},
	}

	ApplyReverse(cs, m, nil)

	elem, ok := m.Model["newservice"]
	if !ok {
		t.Fatal("expected element 'newservice' to be added")
	}
	if elem.Kind == "" {
		t.Error("expected non-empty kind on new element, got empty string")
	}
	// Should be one of the specification kinds.
	if _, valid := m.Specification.Elements[elem.Kind]; !valid {
		t.Errorf("expected kind to be a valid spec kind, got %q", elem.Kind)
	}
}

// TestApplyReverse_NewElementWarningMentionsKind verifies that the warning
// for a new element mentions the assigned kind (#206).
func TestApplyReverse_NewElementWarningMentionsKind(t *testing.T) {
	m := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"system": {Notation: "System"},
			},
		},
		Model: map[string]model.Element{},
	}

	cs := &ChangeSet{
		DrawioElementChanges: []ElementChange{
			{ID: "svc", Type: Added, NewValue: "Service"},
		},
	}

	result := ApplyReverse(cs, m, nil)

	var found bool
	for _, w := range result.Warnings {
		if strings.Contains(w, "kind") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning to mention 'kind', got: %v", result.Warnings)
	}
}

// TestApplyReverse_DeletedElementCleansOrphanedRelationships verifies that
// when an element is deleted from draw.io, all relationships referencing it
// as from or to are also removed from the model. Regression test for #266.
func TestApplyReverse_DeletedElementCleansOrphanedRelationships(t *testing.T) {
	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"api":      {Kind: "container", Title: "API"},
			"db":       {Kind: "container", Title: "Database"},
			"payments": {Kind: "container", Title: "Payments"},
		},
		Relationships: []model.Relationship{
			{From: "api", To: "db", Label: "reads from"},
			{From: "api", To: "payments", Label: "charges"},
			{From: "payments", To: "db", Label: "stores"},
		},
	}

	// Delete "payments" element from draw.io (without deleting its connectors).
	cs := elemChangeSet("payments", Deleted, "", "", "")

	r := ApplyReverse(cs, m, nil)

	// Element should be removed.
	if _, exists := m.Model["payments"]; exists {
		t.Error("expected element 'payments' to be removed from model")
	}

	// All relationships referencing "payments" should be cleaned up.
	for _, rel := range m.Relationships {
		if rel.From == "payments" || rel.To == "payments" {
			t.Errorf("orphaned relationship found: %s -> %s (%s)", rel.From, rel.To, rel.Label)
		}
	}

	// Only api->db should remain.
	if len(m.Relationships) != 1 {
		t.Errorf("expected 1 remaining relationship, got %d: %+v", len(m.Relationships), m.Relationships)
	}

	if r.ElementsDeleted != 1 {
		t.Errorf("expected ElementsDeleted=1, got %d", r.ElementsDeleted)
	}
	if r.RelationshipsDeleted != 2 {
		t.Errorf("expected RelationshipsDeleted=2, got %d", r.RelationshipsDeleted)
	}
}

// TestApplyReverse_DeletedElementCleansNestedRelationships verifies that
// deleting a parent element also cleans up relationships referencing its
// children (e.g., deleting "shop" cleans "shop.api" relationships).
func TestApplyReverse_DeletedElementCleansNestedRelationships(t *testing.T) {
	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"shop": {
				Kind:  "system",
				Title: "Shop",
				Children: map[string]model.Element{
					"api": {Kind: "container", Title: "API"},
				},
			},
			"db": {Kind: "container", Title: "Database"},
		},
		Relationships: []model.Relationship{
			{From: "shop.api", To: "db", Label: "reads"},
			{From: "db", To: "shop.api", Label: "responds"},
		},
	}

	cs := elemChangeSet("shop", Deleted, "", "", "")

	r := ApplyReverse(cs, m, nil)

	if _, exists := m.Model["shop"]; exists {
		t.Error("expected element 'shop' to be removed")
	}

	// Relationships referencing shop.api (child of deleted shop) should be removed.
	if len(m.Relationships) != 0 {
		t.Errorf("expected 0 relationships after deleting parent, got %d: %+v",
			len(m.Relationships), m.Relationships)
	}

	if r.RelationshipsDeleted != 2 {
		t.Errorf("expected RelationshipsDeleted=2, got %d", r.RelationshipsDeleted)
	}
}

// TestApplyReverse_RelationshipLabelUpdated_FallbackByFromTo covers
// relabelRelationshipByFromTo: when ch.Index no longer lines up with the
// model's current array position (e.g. after a reorder), the label update
// must still find the relationship by from/to.
func TestApplyReverse_RelationshipLabelUpdated_FallbackByFromTo(t *testing.T) {
	m := modelWithRel("a", "b", "old label")
	cs := relChangeSet("a", "b", Modified, "label", "old label", "new label")
	cs.DrawioRelationshipChanges[0].Index = 5 // stale/mismatched index

	r := ApplyReverse(cs, m, nil)

	if m.Relationships[0].Label != "new label" {
		t.Errorf("expected label %q via fallback, got %q", "new label", m.Relationships[0].Label)
	}
	if r.RelationshipsUpdated != 1 {
		t.Errorf("expected RelationshipsUpdated=1, got %d", r.RelationshipsUpdated)
	}
}

// TestApplyReverse_RelationshipDeleted_FallbackByFromTo covers
// filterRelationships via applyReverseRelationshipDeleted's fallback path:
// a stale ch.Index must not prevent the relationship from being found and
// deleted by from/to.
func TestApplyReverse_RelationshipDeleted_FallbackByFromTo(t *testing.T) {
	m := modelWithRel("a", "b", "calls")
	cs := relChangeSet("a", "b", Deleted, "", "", "")
	cs.DrawioRelationshipChanges[0].Index = 5 // stale/mismatched index

	r := ApplyReverse(cs, m, nil)

	if len(m.Relationships) != 0 {
		t.Errorf("expected relationship to be removed via fallback, got %d remaining", len(m.Relationships))
	}
	if r.RelationshipsDeleted != 1 {
		t.Errorf("expected RelationshipsDeleted=1, got %d", r.RelationshipsDeleted)
	}
}

// TestApplyReverse_RelationshipLabelUpdated_IndexMismatchFallsBack covers
// relabelRelationshipByIndex's from/to-mismatch branch: ch.Index is within
// bounds but points at a different relationship, so the update must still
// find the right one via the from/to fallback (not just an out-of-range
// index, which TestApplyReverse_RelationshipLabelUpdated_FallbackByFromTo
// already covers).
func TestApplyReverse_RelationshipLabelUpdated_IndexMismatchFallsBack(t *testing.T) {
	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"a": {Kind: "container", Title: "a"},
			"b": {Kind: "container", Title: "b"},
			"c": {Kind: "container", Title: "c"},
		},
		Relationships: []model.Relationship{
			{From: "a", To: "c", Label: "unrelated"},
			{From: "a", To: "b", Label: "old label"},
		},
	}
	cs := relChangeSet("a", "b", Modified, "label", "old label", "new label")
	cs.DrawioRelationshipChanges[0].Index = 0 // in bounds, but that slot is a->c, not a->b

	r := ApplyReverse(cs, m, nil)

	if m.Relationships[1].Label != "new label" {
		t.Errorf("expected label %q via fallback, got %q", "new label", m.Relationships[1].Label)
	}
	if r.RelationshipsUpdated != 1 {
		t.Errorf("expected RelationshipsUpdated=1, got %d", r.RelationshipsUpdated)
	}
}

// TestApplyReverse_RelationshipDeleted_IndexMismatchFallsBack is the delete
// counterpart of the index-mismatch-fallback case above.
func TestApplyReverse_RelationshipDeleted_IndexMismatchFallsBack(t *testing.T) {
	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"a": {Kind: "container", Title: "a"},
			"b": {Kind: "container", Title: "b"},
			"c": {Kind: "container", Title: "c"},
		},
		Relationships: []model.Relationship{
			{From: "a", To: "c", Label: "unrelated"},
			{From: "a", To: "b", Label: "calls"},
		},
	}
	cs := relChangeSet("a", "b", Deleted, "", "", "")
	cs.DrawioRelationshipChanges[0].Index = 0 // in bounds, but that slot is a->c, not a->b

	r := ApplyReverse(cs, m, nil)

	if len(m.Relationships) != 1 || m.Relationships[0].To != "c" {
		t.Errorf("expected only a->c to remain, got %+v", m.Relationships)
	}
	if r.RelationshipsDeleted != 1 {
		t.Errorf("expected RelationshipsDeleted=1, got %d", r.RelationshipsDeleted)
	}
}

// TestApplyReverse_RelationshipAddedToElementMissing is the "To" side
// counterpart of TestApplyReverse_RelationshipAddedStaleElementsRejected
// (which only covers a missing "From").
func TestApplyReverse_RelationshipAddedToElementMissing(t *testing.T) {
	m := modelWithRel("x", "y", "")
	m.Relationships = []model.Relationship{}
	cs := relChangeSet("x", "nonexistent", Added, "", "", "uses")

	r := ApplyReverse(cs, m, nil)

	if len(m.Relationships) != 0 {
		t.Fatalf("expected 0 relationships (stale rejected), got %d", len(m.Relationships))
	}
	warnText := strings.Join(r.Warnings, "; ")
	if !strings.Contains(warnText, "nonexistent") || !strings.Contains(warnText, "does not exist") {
		t.Errorf("expected warning mentioning non-existent To element, got: %s", warnText)
	}
}

// TestApplyReverse_ElementAddedAlreadySyncedSkipped covers the
// "already synced from draw.io before" skip branch: the element exists in
// the model AND was recorded in the last sync state, so a re-appearing
// Added change (e.g. the element moved to another page) is a silent no-op.
func TestApplyReverse_ElementAddedAlreadySyncedSkipped(t *testing.T) {
	m := simpleModel("api", "API", "", "")
	cs := elemChangeSet("api", Added, "", "", "API")
	lastState := &SyncState{
		Timestamp: "2026-01-01T00:00:00Z",
		Elements:  map[string]ElementState{"api": {Title: "API", Kind: "container"}},
	}

	r := ApplyReverse(cs, m, lastState)

	if r.ElementsCreated != 0 {
		t.Errorf("expected 0 elements created (already synced), got %d", r.ElementsCreated)
	}
	if len(r.Warnings) != 0 {
		t.Errorf("expected no warnings for an element already synced before, got: %v", r.Warnings)
	}
}

// TestApplyReverse_ElementDeletedNotFoundWarns covers deleteElement's error
// path: a Deleted change for an ID that doesn't exist in the model at all.
func TestApplyReverse_ElementDeletedNotFoundWarns(t *testing.T) {
	m := simpleModel("api", "API", "", "")
	cs := elemChangeSet("nonexistent", Deleted, "", "", "")

	r := ApplyReverse(cs, m, nil)

	if r.ElementsDeleted != 0 {
		t.Errorf("expected 0 elements deleted, got %d", r.ElementsDeleted)
	}
	warnText := strings.Join(r.Warnings, "; ")
	if !strings.Contains(warnText, "could not be deleted") {
		t.Errorf("expected a \"could not be deleted\" warning, got: %s", warnText)
	}
}

// TestApplyReverse_ElementAddedNilModelMap covers applyReverseElementAdded's
// m.Model-is-nil initialization branch.
func TestApplyReverse_ElementAddedNilModelMap(t *testing.T) {
	m := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{"container": {Notation: "box"}},
		},
	}
	cs := elemChangeSet("newElem", Added, "", "", "New Element")

	r := ApplyReverse(cs, m, nil)

	if r.ElementsCreated != 1 {
		t.Fatalf("expected 1 element created, got %d", r.ElementsCreated)
	}
	if _, ok := m.Model["newElem"]; !ok {
		t.Error("expected new element to be present in the initialized Model map")
	}
}

// TestApplyReverse_RelationshipAddedDuplicateWithPageID covers the
// pageInfo-in-warning branch of applyReverseRelationshipAdded: a duplicate
// relationship rejection where the incoming change carries a draw.io PageID.
func TestApplyReverse_RelationshipAddedDuplicateWithPageID(t *testing.T) {
	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"a": {Kind: "container", Title: "A"},
			"b": {Kind: "container", Title: "B"},
		},
		Relationships: []model.Relationship{
			{From: "a", To: "b", Label: "uses"}, // index 0, present in model but never synced from draw.io
		},
	}
	lastState := &SyncState{
		Timestamp: "2026-07-26T00:00:00Z", // a prior sync completed
	}
	cs := &ChangeSet{
		DrawioRelationshipChanges: []RelationshipChange{
			{From: "a", To: "b", Index: 0, Type: Added, NewValue: "uses", PageID: "view-context"},
		},
	}

	r := ApplyReverse(cs, m, lastState)

	if len(m.Relationships) != 1 {
		t.Fatalf("expected the duplicate to be rejected, got %d relationships", len(m.Relationships))
	}
	warnText := strings.Join(r.Warnings, "; ")
	if !strings.Contains(warnText, "view-context") {
		t.Errorf("expected warning to mention the draw.io page, got: %s", warnText)
	}
}

// TestApplyReverse_RelationshipAddedAlreadySyncedSkipped covers the
// lastRelMap-hit branch: the relationship was already synced from draw.io
// in a previous sync (e.g. it now appears on a different page), so a
// re-appearing Added change is a silent no-op.
func TestApplyReverse_RelationshipAddedAlreadySyncedSkipped(t *testing.T) {
	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"a": {Kind: "container", Title: "A"},
			"b": {Kind: "container", Title: "B"},
		},
		Relationships: []model.Relationship{
			{From: "a", To: "b", Label: "uses"},
		},
	}
	lastState := &SyncState{
		Timestamp: "2026-07-26T00:00:00Z",
		Relationships: []RelationshipState{
			{From: "a", To: "b", Index: 0, Label: "uses"},
		},
	}
	cs := &ChangeSet{
		DrawioRelationshipChanges: []RelationshipChange{
			{From: "a", To: "b", Index: 0, Type: Added, NewValue: "uses"},
		},
	}

	r := ApplyReverse(cs, m, lastState)

	if len(m.Relationships) != 1 {
		t.Fatalf("expected no new relationship, got %d", len(m.Relationships))
	}
	if r.RelationshipsCreated != 0 {
		t.Errorf("expected RelationshipsCreated=0, got %d", r.RelationshipsCreated)
	}
	if len(r.Warnings) != 0 {
		t.Errorf("expected no warnings for an already-synced relationship, got: %v", r.Warnings)
	}
}
