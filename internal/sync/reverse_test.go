package sync

import (
	"strings"
	"testing"

	"github.com/docToolchain/Bauteinsicht/internal/model"
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

	r := ApplyReverse(cs, m)

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

	r := ApplyReverse(cs, m)

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

	r := ApplyReverse(cs, m)

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

	r := ApplyReverse(cs, m)

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

	r := ApplyReverse(cs, m)

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

	r := ApplyReverse(cs, m)

	if len(r.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", len(r.Warnings), r.Warnings)
	}
	// The warning should include the element ID and mention the model. (#115)
	w := r.Warnings[0]
	if !strings.Contains(w, `"unknown"`) {
		t.Errorf("expected warning to include element ID %q, got: %s", "unknown", w)
	}
	if !strings.Contains(w, "model") {
		t.Errorf("expected warning to mention 'model', got: %s", w)
	}
	if r.ElementsCreated != 0 {
		t.Errorf("expected ElementsCreated=0, got %d", r.ElementsCreated)
	}
}

func TestApplyReverse_ModifyMissingElement(t *testing.T) {
	m := emptyModel()
	cs := elemChangeSet("nonexistent", Modified, "title", "", "New")

	r := ApplyReverse(cs, m)

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

	r := ApplyReverse(cs, m)

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

	r := ApplyReverse(cs, m)

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
	ApplyReverse(cs, m)

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

	r := ApplyReverse(cs, m)

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
	m := emptyModel()
	cs := relChangeSet("x", "y", Added, "", "", "uses")

	r := ApplyReverse(cs, m)

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
}
