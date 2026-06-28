package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// newTestReplState returns a replState with an in-memory scanner for testing.
func newTestReplState(input string) *replState {
	m := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"actor":     {Notation: "Person"},
				"system":    {Notation: "System", Container: true},
				"container": {Notation: "Container"},
			},
		},
		Model: map[string]model.Element{
			"customer": {Kind: "actor", Title: "Customer"},
			"webshop":  {Kind: "system", Title: "Webshop"},
		},
		Relationships: []model.Relationship{},
		Views:         map[string]model.View{},
	}
	return &replState{
		model:      m,
		modelPath:  "test.jsonc",
		undoStack:  make([]*model.BausteinsichtModel, 0),
		maxUndoLen: 50,
		scanner:    bufio.NewScanner(strings.NewReader(input)),
	}
}

// TestReplCommandDispatch verifies that executeCommand routes to the correct handler.
func TestReplCommandDispatch(t *testing.T) {
	tests := []struct {
		name    string
		cmd     string
		wantErr bool
	}{
		{"help", "help", false},
		{"list elements", "list elements", false},
		{"list relationships", "list relationships", false},
		{"list views", "list views", false},
		{"validate", "validate", false},
		{"undo empty", "undo", false},
		{"unknown command", "foobar", false},
		{"empty line", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestReplState("")
			err := s.executeCommand(tt.cmd, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("executeCommand(%q) error = %v, wantErr %v", tt.cmd, err, tt.wantErr)
			}
		})
	}
}

// TestReplExitCommand verifies that "exit" returns errReplExit.
func TestReplExitCommand(t *testing.T) {
	s := newTestReplState("")
	err := s.executeCommand("exit", nil)
	if err != errReplExit {
		t.Errorf("exit: got error %v, want errReplExit", err)
	}
}

// TestReplExitWithUnsavedChanges verifies that "exit" with unsaved changes
// prompts and respects "yes" confirmation, including with surrounding whitespace.
func TestReplExitWithUnsavedChanges(t *testing.T) {
	// Answer "yes" to the "Exit anyway?" prompt.
	s := newTestReplState("yes\n")
	s.modified = true
	err := s.executeCommand("exit", nil)
	if err != errReplExit {
		t.Errorf("exit yes: got error %v, want errReplExit", err)
	}

	// Answer "yes " (trailing space) — should still exit.
	s3 := newTestReplState("yes \n")
	s3.modified = true
	if err := s3.executeCommand("exit", nil); err != errReplExit {
		t.Errorf("exit 'yes ': got error %v, want errReplExit", err)
	}

	// Answer "no" to the prompt — should not exit.
	s2 := newTestReplState("no\n")
	s2.modified = true
	err2 := s2.executeCommand("exit", nil)
	if err2 != nil {
		t.Errorf("exit no: got error %v, want nil", err2)
	}
}

// TestReplAddElementAndUndo verifies add element followed by undo.
func TestReplAddElementAndUndo(t *testing.T) {
	// Simulate: add element with id="backend", kind="container", title="Backend", no desc.
	// addCommand calls saveUndo() before delegating to addElementInteractive.
	input := "backend\ncontainer\nBackend\n\n"
	s := newTestReplState(input)

	s.addCommand([]string{"element"})

	if _, ok := s.model.Model["backend"]; !ok {
		t.Fatal("element 'backend' was not added")
	}
	if !s.modified {
		t.Error("modified flag should be true after add")
	}
	if len(s.undoStack) != 1 {
		t.Errorf("undoStack length: got %d, want 1", len(s.undoStack))
	}

	// Undo should remove the element.
	if err := s.undoCommand(); err != nil {
		t.Fatalf("undoCommand: %v", err)
	}
	if _, ok := s.model.Model["backend"]; ok {
		t.Error("element 'backend' should be gone after undo")
	}
}

// TestReplAddRelationshipAndUndo verifies add relationship followed by undo.
func TestReplAddRelationshipAndUndo(t *testing.T) {
	// addCommand calls saveUndo() before delegating to addRelationshipInteractive.
	input := "customer\nwebshop\nuses\n"
	s := newTestReplState(input)

	s.addCommand([]string{"relationship"})

	if len(s.model.Relationships) != 1 {
		t.Fatalf("expected 1 relationship, got %d", len(s.model.Relationships))
	}
	rel := s.model.Relationships[0]
	if rel.From != "customer" || rel.To != "webshop" {
		t.Errorf("relationship: got %s→%s, want customer→webshop", rel.From, rel.To)
	}

	// Undo.
	if err := s.undoCommand(); err != nil {
		t.Fatalf("undoCommand: %v", err)
	}
	if len(s.model.Relationships) != 0 {
		t.Error("relationship should be gone after undo")
	}
}

// TestReplRemoveElement verifies that removeCommand deletes a top-level element.
func TestReplRemoveElement(t *testing.T) {
	s := newTestReplState("")
	s.removeCommand([]string{"element", "customer"})
	if _, ok := s.model.Model["customer"]; ok {
		t.Error("element 'customer' should have been removed")
	}
	if !s.modified {
		t.Error("modified flag should be true after remove")
	}
}

// TestReplRemoveNonExistentElement verifies no-op and undo stack cleanup.
func TestReplRemoveNonExistentElement(t *testing.T) {
	s := newTestReplState("")
	s.removeCommand([]string{"element", "nonexistent"})
	if len(s.undoStack) != 0 {
		t.Errorf("undo stack should be empty after no-op remove, got %d", len(s.undoStack))
	}
}

// TestReplShowElement verifies that showCommand does not panic on a valid ID.
func TestReplShowElement(t *testing.T) {
	s := newTestReplState("")
	// Should not panic or error.
	s.showCommand([]string{"customer"})
}

// TestReplUndoEmpty verifies that undoCommand on empty stack is a no-op.
func TestReplUndoEmpty(t *testing.T) {
	s := newTestReplState("")
	if err := s.undoCommand(); err != nil {
		t.Errorf("undoCommand on empty stack: %v", err)
	}
}

// TestReplValidateCommand verifies validateCommand runs without panic.
func TestReplValidateCommand(t *testing.T) {
	s := newTestReplState("")
	s.validateCommand()
}

// TestReplAutoDetectFallback_NoModelFile verifies that the REPL command returns
// an error when no model file is found and --model is not set.
func TestReplAutoDetectFallback_NoModelFile(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()

	cmd := newReplCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when no .jsonc file found, got nil")
	}
}

// TestReplAutoDetectFallback_FindsModel verifies that the REPL loads a model
// via auto-detection (no --model flag) when exactly one .jsonc file is present.
// The REPL loop exits immediately on EOF — no stdin interaction needed.
func TestReplAutoDetectFallback_FindsModel(t *testing.T) {
	dir := t.TempDir()

	m := &model.BausteinsichtModel{
		Model:         map[string]model.Element{"svc": {Kind: "system", Title: "Svc"}},
		Relationships: []model.Relationship{},
		Views:         map[string]model.View{},
	}
	data, _ := json.Marshal(m)
	modelPath := filepath.Join(dir, "arch.jsonc")
	if err := os.WriteFile(modelPath, data, 0600); err != nil {
		t.Fatalf("writing model: %v", err)
	}

	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()

	// Replace stdin with an empty reader so the REPL loop exits immediately on EOF.
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	_ = w.Close()
	defer func() { os.Stdin = oldStdin }()

	cmd := newReplCmd()
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected clean exit on auto-detected model, got: %v", err)
	}
}

// TestReplAutoDetectFallback_AmbiguousModel verifies that two .jsonc files
// in the same directory cause auto-detection to fail with an error.
func TestReplAutoDetectFallback_AmbiguousModel(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.jsonc", "b.jsonc"} {
		_ = os.WriteFile(filepath.Join(dir, name), []byte(`{}`), 0600)
	}

	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()

	cmd := newReplCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for ambiguous model, got nil")
	}
}

// TestReplUndoStackCapped verifies the undo stack is trimmed to exactly maxUndoLen.
func TestReplUndoStackCapped(t *testing.T) {
	s := newTestReplState("")
	s.maxUndoLen = 3

	// Push 5 undo entries.
	for i := 0; i < 5; i++ {
		s.saveUndo()
	}
	if len(s.undoStack) != 3 {
		t.Errorf("undoStack length: got %d, want 3", len(s.undoStack))
	}
}

// TestReplEvictedModifiedFlag verifies that modified stays true after undo-stack
// cap eviction even when all remaining entries are undone.
func TestReplEvictedModifiedFlag(t *testing.T) {
	s := newTestReplState("")
	s.maxUndoLen = 2

	// Push 3 entries — triggers eviction.
	for i := 0; i < 3; i++ {
		s.saveUndo()
	}
	if !s.evicted {
		t.Fatal("expected evicted=true after exceeding maxUndoLen")
	}

	// Undo all remaining stack entries.
	for len(s.undoStack) > 0 {
		if err := s.undoCommand(); err != nil {
			t.Fatalf("undoCommand: %v", err)
		}
	}
	if !s.modified {
		t.Error("modified should remain true after eviction even when undo stack is empty")
	}
}

// TestReplExecuteCommand_UsageMessages verifies that commands without required
// sub-arguments print usage and return nil.
func TestReplExecuteCommand_UsageMessages(t *testing.T) {
	for _, cmd := range []string{"list", "add", "show", "remove"} {
		t.Run(cmd+" alone", func(t *testing.T) {
			s := newTestReplState("")
			err := s.executeCommand(cmd, nil)
			if err != nil {
				t.Errorf("executeCommand(%q) = %v, want nil", cmd, err)
			}
		})
	}
}

// TestReplPrintHelp verifies printHelp runs without panic.
func TestReplPrintHelp(t *testing.T) {
	s := newTestReplState("")
	s.printHelp() // just verify it doesn't panic
}

// TestReplAddElement_EmptyID verifies that an empty element ID aborts the add.
func TestReplAddElement_EmptyID(t *testing.T) {
	s := newTestReplState("\n") // empty line → empty ID
	s.addElementInteractive()
	if len(s.model.Model) != 2 { // only the pre-existing elements
		t.Errorf("expected no new element added, got model size %d", len(s.model.Model))
	}
}

// TestReplAddElement_OverwriteNo verifies that answering "no" to overwrite aborts.
func TestReplAddElement_OverwriteNo(t *testing.T) {
	// "customer" already exists; answer "no" to overwrite prompt.
	s := newTestReplState("customer\nno\n")
	s.addElementInteractive()
	// Element should remain unchanged (not overwritten).
	if s.model.Model["customer"].Kind != "actor" {
		t.Error("element should not have been overwritten")
	}
}

// TestReplRemoveRelationship_InsufficientArgs verifies usage message and undo cleanup.
func TestReplRemoveRelationship_InsufficientArgs(t *testing.T) {
	s := newTestReplState("")
	s.removeCommand([]string{"relationship"}) // missing <from> and <to>
	if len(s.undoStack) != 0 {
		t.Errorf("undo stack should be cleaned up, got len %d", len(s.undoStack))
	}
}

// TestReplShowElement_NotFound verifies show prints "not found" for unknown IDs.
func TestReplShowElement_NotFound(t *testing.T) {
	s := newTestReplState("")
	s.showCommand([]string{"nonexistent"}) // should not panic
}

// TestReplListCommand_UnknownSubcommand verifies list with unknown subcommand is a no-op.
func TestReplListCommand_UnknownSubcommand(t *testing.T) {
	s := newTestReplState("")
	s.listCommand([]string{"unknown"}) // falls through switch, prints newline
}

// TestReplRemoveRelationship_Found verifies relationship removal by from/to pair.
func TestReplRemoveRelationship_Found(t *testing.T) {
	s := newTestReplState("")
	s.model.Relationships = []model.Relationship{
		{From: "a", To: "b", Label: "calls"},
	}
	s.removeCommand([]string{"relationship", "a", "b"})
	if len(s.model.Relationships) != 0 {
		t.Error("relationship should have been removed")
	}
	if !s.modified {
		t.Error("modified flag should be set")
	}
}

// TestReplRemoveRelationship_NotFound verifies undo cleanup when relationship missing.
func TestReplRemoveRelationship_NotFound(t *testing.T) {
	s := newTestReplState("")
	s.removeCommand([]string{"relationship", "x", "y"})
	if len(s.undoStack) != 0 {
		t.Errorf("undo stack should be cleaned up after no-op, got %d", len(s.undoStack))
	}
}

// TestReplAddElement_InvalidID verifies that an invalid ID (dots, spaces) aborts.
func TestReplAddElement_InvalidID(t *testing.T) {
	s := newTestReplState("foo.bar\n")
	s.addElementInteractive()
	if len(s.model.Model) != 2 {
		t.Errorf("expected no new element, got model size %d", len(s.model.Model))
	}
}

// TestReplAddElement_EmptyKind verifies that an empty kind aborts the add.
func TestReplAddElement_EmptyKind(t *testing.T) {
	// valid ID, then empty kind → abort
	s := newTestReplState("newservice\n\n")
	s.addElementInteractive()
	if _, ok := s.model.Model["newservice"]; ok {
		t.Error("element should not have been added (empty kind)")
	}
}

// TestReplAddElement_EmptyTitle verifies that an empty title aborts the add.
func TestReplAddElement_EmptyTitle(t *testing.T) {
	// valid ID, valid kind, then empty title → abort
	s := newTestReplState("newservice\nsystem\n\n")
	s.addElementInteractive()
	if _, ok := s.model.Model["newservice"]; ok {
		t.Error("element should not have been added (empty title)")
	}
}

// TestReplAddRelationship_EmptyFrom verifies that an empty from ID aborts.
func TestReplAddRelationship_EmptyFrom(t *testing.T) {
	s := newTestReplState("\n")
	s.addRelationshipInteractive()
	if len(s.model.Relationships) != 0 {
		t.Error("no relationship should have been added")
	}
}

// TestReplAddRelationship_EmptyTo verifies that an empty to ID aborts.
func TestReplAddRelationship_EmptyTo(t *testing.T) {
	s := newTestReplState("customer\n\n")
	s.addRelationshipInteractive()
	if len(s.model.Relationships) != 0 {
		t.Error("no relationship should have been added")
	}
}

// TestReplAddRelationship_FromNotFound verifies that an unknown from ID aborts.
func TestReplAddRelationship_FromNotFound(t *testing.T) {
	s := newTestReplState("unknown\nwebshop\n")
	s.addRelationshipInteractive()
	if len(s.model.Relationships) != 0 {
		t.Error("no relationship should have been added for unknown from")
	}
}

// TestReplAddRelationship_ToNotFound verifies that an unknown to ID aborts.
func TestReplAddRelationship_ToNotFound(t *testing.T) {
	s := newTestReplState("customer\nunknown\n")
	s.addRelationshipInteractive()
	if len(s.model.Relationships) != 0 {
		t.Error("no relationship should have been added for unknown to")
	}
}
