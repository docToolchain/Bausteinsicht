package main

import (
	"bufio"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// newTestReplState returns a replState with an in-memory scanner for testing.
func newTestReplState(input string) *replState {
	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"customer": {Kind: "actor", Title: "Customer"},
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
// prompts and respects "yes" confirmation.
func TestReplExitWithUnsavedChanges(t *testing.T) {
	// Answer "yes" to the "Exit anyway?" prompt.
	s := newTestReplState("yes\n")
	s.modified = true
	err := s.executeCommand("exit", nil)
	if err != errReplExit {
		t.Errorf("exit yes: got error %v, want errReplExit", err)
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
	input := "backend\ncontainer\nBackend\n\n"
	s := newTestReplState(input)

	s.addElementInteractive()

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
	input := "customer\nwebshop\nuses\n"
	s := newTestReplState(input)

	s.addRelationshipInteractive()

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

// TestReplUndoStackCapped verifies the undo stack is trimmed to maxUndoLen.
func TestReplUndoStackCapped(t *testing.T) {
	s := newTestReplState("")
	s.maxUndoLen = 3

	// Push 5 undo entries.
	for i := 0; i < 5; i++ {
		s.saveUndo()
	}
	if len(s.undoStack) > 3 {
		t.Errorf("undoStack length: got %d, want <= 3", len(s.undoStack))
	}
}
