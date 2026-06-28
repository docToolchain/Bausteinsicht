package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// newFileReplState creates a replState backed by a real temp model file for
// tests that exercise saveCommand / patchSave.
func newFileReplState(t *testing.T, input string) (*replState, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "model.jsonc")
	initial := `{
  "specification": {
    "elements": {
      "actor":  {"notation": "Person"},
      "system": {"notation": "System"}
    }
  },
  "model": {
    "customer": {"kind": "actor", "title": "Customer"},
    "webshop": {"kind": "system", "title": "Webshop"}
  },
  "relationships": [],
  "views": {}
}`
	if err := os.WriteFile(path, []byte(initial), 0600); err != nil {
		t.Fatalf("write model: %v", err)
	}
	m, err := model.Load(path)
	if err != nil {
		t.Fatalf("load model: %v", err)
	}
	return &replState{
		model:      m,
		modelPath:  path,
		undoStack:  make([]*model.BausteinsichtModel, 0),
		maxUndoLen: 50,
		scanner:    bufio.NewScanner(strings.NewReader(input)),
	}, path
}

// TestReplSaveCommand_PatchSave verifies that saveCommand writes a new element
// to the model file using the comment-preserving patch path.
func TestReplSaveCommand_PatchSave(t *testing.T) {
	s, path := newFileReplState(t, "")
	// Add a new element in memory.
	s.saveUndo()
	s.model.Model["payments"] = model.Element{Kind: "system", Title: "Payments"}
	s.modified = true

	if err := s.saveCommand(); err != nil {
		t.Fatalf("saveCommand: %v", err)
	}

	// Reload and verify the element was written.
	reloaded, err := model.Load(path)
	if err != nil {
		t.Fatalf("reloading model: %v", err)
	}
	if _, ok := reloaded.Model["payments"]; !ok {
		t.Error("payments element not found after save")
	}
	if s.modified {
		t.Error("modified flag should be false after save")
	}
	if s.undoStack != nil {
		t.Error("undoStack should be nil after save")
	}
}

// TestReplSaveCommand_PatchSave_Relationship verifies that new relationships
// are appended via the comment-preserving path.
func TestReplSaveCommand_PatchSave_Relationship(t *testing.T) {
	s, path := newFileReplState(t, "")
	s.saveUndo()
	s.model.Relationships = append(s.model.Relationships, model.Relationship{
		From: "customer", To: "webshop", Label: "uses",
	})
	s.modified = true

	if err := s.saveCommand(); err != nil {
		t.Fatalf("saveCommand: %v", err)
	}

	reloaded, err := model.Load(path)
	if err != nil {
		t.Fatalf("reloading model: %v", err)
	}
	if len(reloaded.Relationships) != 1 {
		t.Errorf("expected 1 relationship after save, got %d", len(reloaded.Relationships))
	}
}

// TestReplSaveCommand_FallsBackOnDeletion verifies that deleting an element
// causes saveCommand to fall back to the full model.Save path.
func TestReplSaveCommand_FallsBackOnDeletion(t *testing.T) {
	s, path := newFileReplState(t, "")
	// Delete an element that was on disk.
	s.saveUndo()
	delete(s.model.Model, "webshop")
	s.modified = true

	if err := s.saveCommand(); err != nil {
		t.Fatalf("saveCommand: %v", err)
	}

	reloaded, err := model.Load(path)
	if err != nil {
		t.Fatalf("reloading model: %v", err)
	}
	if _, ok := reloaded.Model["webshop"]; ok {
		t.Error("webshop should have been removed by full save fallback")
	}
}

// TestReplUndoSetsModified verifies that undoing all changes resets modified to false.
func TestReplUndoSetsModified(t *testing.T) {
	s, _ := newFileReplState(t, "")
	// Add then undo.
	s.saveUndo()
	s.model.Model["tmp"] = model.Element{Kind: "system", Title: "Tmp"}
	s.modified = true

	if err := s.undoCommand(); err != nil {
		t.Fatalf("undoCommand: %v", err)
	}
	if s.modified {
		t.Error("modified should be false after undoing all changes")
	}
}

// TestReplPatchSave_InvalidModelPath verifies patchSave returns an error when the
// model file does not exist (triggers the model.Load error path).
func TestReplPatchSave_InvalidModelPath(t *testing.T) {
	s := newTestReplState("")
	s.modelPath = "/nonexistent/path/model.jsonc"
	if err := s.patchSave(); err == nil {
		t.Fatal("expected error for nonexistent model path")
	}
}

// TestReplSaveCommand_ModifiedElementFallback verifies that modifying an existing
// element triggers the full-save fallback (regression for patchSave element overwrite).
func TestReplSaveCommand_ModifiedElementFallback(t *testing.T) {
	s, path := newFileReplState(t, "")
	// Modify an existing element in memory.
	s.saveUndo()
	existing := s.model.Model["webshop"]
	existing.Title = "Webshop v2"
	s.model.Model["webshop"] = existing
	s.modified = true

	if err := s.saveCommand(); err != nil {
		t.Fatalf("saveCommand: %v", err)
	}

	reloaded, err := model.Load(path)
	if err != nil {
		t.Fatalf("reloading model: %v", err)
	}
	if reloaded.Model["webshop"].Title != "Webshop v2" {
		t.Errorf("expected title 'Webshop v2' after full-save fallback, got %q",
			reloaded.Model["webshop"].Title)
	}
}

// TestReplSaveCommand_RemoveAddEqualCount verifies that removing one relationship
// and adding another (equal count) correctly writes both changes — not masked by
// count-based comparison (regression for multiset fix).
func TestReplSaveCommand_RemoveAddEqualCount(t *testing.T) {
	s, path := newFileReplState(t, "")
	// Seed relationship a→b on disk.
	s.model.Relationships = []model.Relationship{{From: "customer", To: "webshop", Label: "original"}}
	if err := s.saveCommand(); err != nil {
		t.Fatalf("initial save: %v", err)
	}

	// Remove original, add new one (equal count).
	s.model.Relationships = []model.Relationship{{From: "customer", To: "webshop", Label: "replacement"}}
	s.modified = true
	if err := s.saveCommand(); err != nil {
		t.Fatalf("second save: %v", err)
	}

	reloaded, err := model.Load(path)
	if err != nil {
		t.Fatalf("reloading: %v", err)
	}
	if len(reloaded.Relationships) != 1 {
		t.Fatalf("expected 1 relationship, got %d", len(reloaded.Relationships))
	}
	if reloaded.Relationships[0].Label != "replacement" {
		t.Errorf("expected label 'replacement', got %q", reloaded.Relationships[0].Label)
	}
}

// TestReplSaveCommand_ParallelLabelRoundtrip verifies that two relationships with
// the same from/to but different labels are both preserved after save (regression
// for full-signature multiset dedup in patchSave). Seeds one relationship on disk
// first so the dedup logic for existing entries is exercised.
func TestReplSaveCommand_ParallelLabelRoundtrip(t *testing.T) {
	s, path := newFileReplState(t, "")

	// Seed "reads" on disk.
	s.model.Relationships = []model.Relationship{{From: "customer", To: "webshop", Label: "reads"}}
	s.modified = true
	if err := s.saveCommand(); err != nil {
		t.Fatalf("initial save: %v", err)
	}

	// Now add "writes" in memory — patchSave must append only the new rel, not duplicate "reads".
	s.model.Relationships = []model.Relationship{
		{From: "customer", To: "webshop", Label: "reads"},
		{From: "customer", To: "webshop", Label: "writes"},
	}
	s.modified = true
	if err := s.saveCommand(); err != nil {
		t.Fatalf("second saveCommand: %v", err)
	}

	reloaded, err := model.Load(path)
	if err != nil {
		t.Fatalf("reloading: %v", err)
	}
	if len(reloaded.Relationships) != 2 {
		t.Fatalf("expected exactly 2 relationships, got %d", len(reloaded.Relationships))
	}
	labels := map[string]bool{}
	for _, r := range reloaded.Relationships {
		labels[r.Label] = true
	}
	if !labels["reads"] || !labels["writes"] {
		t.Errorf("expected both 'reads' and 'writes' labels, got: %v", labels)
	}
}

// TestReplAddElement_OverwritePreservesChildren verifies that overwriting an
// element preserves its children, technology, tags and other fields not re-prompted,
// including after save+reload from disk.
func TestReplAddElement_OverwritePreservesChildren(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "model.jsonc")
	// Spec must declare system as container and include the container kind.
	initial := `{
  "specification": {
    "elements": {
      "actor":     {"notation": "Person"},
      "system":    {"notation": "System", "container": true},
      "container": {"notation": "Container"}
    }
  },
  "model": {
    "customer": {"kind": "actor", "title": "Customer"},
    "shop": {
      "kind": "system", "title": "Shop", "technology": "Go",
      "tags": ["critical"],
      "children": {"api": {"kind": "container", "title": "API"}}
    }
  },
  "relationships": [],
  "views": {}
}`
	if err := os.WriteFile(path, []byte(initial), 0600); err != nil {
		t.Fatalf("write model: %v", err)
	}
	m, err := model.Load(path)
	if err != nil {
		t.Fatalf("load model: %v", err)
	}
	s := &replState{
		model:      m,
		modelPath:  path,
		undoStack:  make([]*model.BausteinsichtModel, 0),
		maxUndoLen: 50,
		// Input: id=shop → overwrite=yes → kind=system → title=Shop v2 → desc=
		scanner: bufio.NewScanner(strings.NewReader("shop\nyes\nsystem\nShop v2\n\n")),
	}

	s.addElementInteractive()

	// Verify in-memory preservation.
	elem := s.model.Model["shop"]
	if elem.Kind != "system" || elem.Title != "Shop v2" {
		t.Errorf("overwrite should update kind/title, got kind=%q title=%q", elem.Kind, elem.Title)
	}
	if elem.Technology != "Go" {
		t.Errorf("Technology should be preserved on overwrite, got %q", elem.Technology)
	}
	if len(elem.Tags) != 1 || elem.Tags[0] != "critical" {
		t.Errorf("Tags should be preserved on overwrite, got %v", elem.Tags)
	}
	if len(elem.Children) != 1 {
		t.Errorf("Children should be preserved on overwrite, got %d children", len(elem.Children))
	}

	// Verify preservation after save and disk reload.
	if err := s.saveCommand(); err != nil {
		t.Fatalf("saveCommand after overwrite: %v", err)
	}
	diskModel, err := model.Load(path)
	if err != nil {
		t.Fatalf("final reload: %v", err)
	}
	diskElem := diskModel.Model["shop"]
	if diskElem.Technology != "Go" {
		t.Errorf("Technology not preserved on disk, got %q", diskElem.Technology)
	}
	if len(diskElem.Children) != 1 {
		t.Errorf("Children not preserved on disk, got %d children", len(diskElem.Children))
	}
}

// TestReplRemoveRelationship_LabelWithSpaces verifies that a multi-word label can
// be matched when removing a relationship (regression for strings.Fields truncation).
func TestReplRemoveRelationship_LabelWithSpaces(t *testing.T) {
	s := newTestReplState("")
	s.model.Relationships = []model.Relationship{
		{From: "a", To: "b", Label: "sends data"},
	}
	s.removeCommand([]string{"relationship", "a", "b", "sends", "data"})
	if len(s.model.Relationships) != 0 {
		t.Error("relationship with multi-word label should have been removed")
	}
}

// TestReplSaveCommand_FallsBackOnRelationshipDeletion verifies that removing a
// relationship triggers the full-save fallback (patchSave cannot delete entries).
func TestReplSaveCommand_FallsBackOnRelationshipDeletion(t *testing.T) {
	s, path := newFileReplState(t, "")
	// Seed an on-disk relationship first.
	s.model.Relationships = append(s.model.Relationships, model.Relationship{From: "customer", To: "webshop"})
	if err := s.saveCommand(); err != nil {
		t.Fatalf("initial save: %v", err)
	}

	// Now remove the relationship in-memory — patchSave should reject this.
	s.model.Relationships = nil
	s.modified = true
	if err := s.saveCommand(); err != nil {
		t.Fatalf("fallback save: %v", err)
	}

	reloaded, err := model.Load(path)
	if err != nil {
		t.Fatalf("reloading: %v", err)
	}
	if len(reloaded.Relationships) != 0 {
		t.Errorf("expected 0 relationships after full-save fallback, got %d", len(reloaded.Relationships))
	}
}

// TestReplSaveCommand_PreInvalidModelAllowsSave verifies that saving is allowed
// when the on-disk model already has validation errors (e.g. a broken view reference).
// Regression test for #410: previously ALL saves were blocked on any validation error.
func TestReplSaveCommand_PreInvalidModelAllowsSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "model.jsonc")
	// Model with a broken view reference (refers to non-existent element "ghost").
	initial := `{
  "specification": {"elements": {"system": {"notation": "System"}}},
  "model": {"shop": {"kind": "system", "title": "Shop"}},
  "relationships": [],
  "views": {"context": {"title": "Context", "include": ["ghost"]}}
}`
	if err := os.WriteFile(path, []byte(initial), 0600); err != nil {
		t.Fatalf("write model: %v", err)
	}
	m, err := model.Load(path)
	if err != nil {
		t.Fatalf("load model: %v", err)
	}
	s := &replState{
		model: m, modelPath: path,
		undoStack: make([]*model.BausteinsichtModel, 0), maxUndoLen: 50,
		scanner: bufio.NewScanner(strings.NewReader("")),
	}

	// Add a valid element — save should succeed despite the pre-existing view error.
	s.model.Model["api"] = model.Element{Kind: "system", Title: "API"}
	s.modified = true

	if err := s.saveCommand(); err != nil {
		t.Fatalf("saveCommand should succeed on pre-invalid model: %v", err)
	}

	reloaded, err := model.Load(path)
	if err != nil {
		t.Fatalf("reloading: %v", err)
	}
	if _, ok := reloaded.Model["api"]; !ok {
		t.Error("api element should have been saved despite pre-existing validation error")
	}
}

// TestReplOverwrite_EmptyDescriptionKeepsExisting verifies that pressing Enter
// on the description prompt during an overwrite keeps the existing description.
// Regression test for #410: previously empty input silently cleared the description.
func TestReplOverwrite_EmptyDescriptionKeepsExisting(t *testing.T) {
	s := newTestReplState("")
	s.model.Model["webshop"] = model.Element{
		Kind: "system", Title: "Webshop", Description: "The main shop",
	}
	// Input: id=webshop → overwrite=yes → kind=system → title=Webshop → desc="" (Enter)
	s.scanner = bufio.NewScanner(strings.NewReader("webshop\nyes\nsystem\nWebshop\n\n"))

	s.addElementInteractive()

	if got := s.model.Model["webshop"].Description; got != "The main shop" {
		t.Errorf("empty input should keep existing description, got %q", got)
	}
}
