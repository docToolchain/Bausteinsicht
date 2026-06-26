package main

import (
	"encoding/json"
	"fmt"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

func (s *replState) saveCommand() error {
	if errs := model.Validate(s.model); len(errs) > 0 {
		// Allow saving when errors are pre-existing (e.g. broken view reference that
		// cannot be fixed via REPL commands). Only block when the edit would increase
		// the error count above the on-disk baseline.
		onDisk, loadErr := model.Load(s.modelPath)
		diskErrCount := 0
		if loadErr == nil {
			diskErrCount = len(model.Validate(onDisk))
		}
		if len(errs) > diskErrCount {
			fmt.Printf("❌ Save would introduce %d new validation error(s) — save aborted:\n", len(errs)-diskErrCount)
			for _, ve := range errs {
				fmt.Printf("  %s\n", ve.Error())
			}
			return nil
		}
	}

	err := s.patchSave()
	if err != nil {
		// Fall back to full save (loses comments) when patch is not possible.
		fmt.Printf("(note: comments may not be preserved — %v)\n", err)
		if err2 := model.Save(s.modelPath, s.model); err2 != nil {
			return err2
		}
	}
	s.modified = false
	s.evicted = false
	s.undoStack = nil
	fmt.Printf("✅ Saved to %s\n", s.modelPath)
	return nil
}

// patchSave writes only the changes between the on-disk model and the in-memory
// model using comment-preserving patch operations. Falls back by returning an error
// whenever any existing element was modified or any relationship was removed or changed
// (which require a full rewrite). Only pure inserts use the comment-preserving path.
func (s *replState) patchSave() error {
	onDisk, err := model.Load(s.modelPath)
	if err != nil {
		return fmt.Errorf("reading model: %w", err)
	}

	// Reject element deletions or modifications (value-based compare).
	for id, onDiskElem := range onDisk.Model {
		memElem, ok := s.model.Model[id]
		if !ok {
			return fmt.Errorf("element %q was deleted; full save required", id)
		}
		if !elementsEqual(onDiskElem, memElem) {
			return fmt.Errorf("element %q was modified; full save required", id)
		}
	}

	// Reject relationship deletions or modifications (multiset compare by full value).
	type relSig struct{ from, to, label, kind string }
	memRelMultiset := make(map[relSig]int, len(s.model.Relationships))
	for _, r := range s.model.Relationships {
		memRelMultiset[relSig{r.From, r.To, r.Label, r.Kind}]++
	}
	for _, r := range onDisk.Relationships {
		sig := relSig{r.From, r.To, r.Label, r.Kind}
		if memRelMultiset[sig] == 0 {
			return fmt.Errorf("relationship %s→%s was removed or changed; full save required", r.From, r.To)
		}
		memRelMultiset[sig]--
	}

	// Insert new top-level elements.
	for id, elem := range s.model.Model {
		if _, exists := onDisk.Model[id]; exists {
			continue
		}
		elemJSON, merr := json.Marshal(elem)
		if merr != nil {
			return fmt.Errorf("marshaling element %s: %w", id, merr)
		}
		capturedID := id
		capturedJSON := string(elemJSON)
		if perr := model.PatchInsert(s.modelPath, func(data []byte) ([]byte, error) {
			return model.InsertObjectEntry(data, []string{"model"}, capturedID, capturedJSON)
		}); perr != nil {
			return fmt.Errorf("inserting element %s: %w", capturedID, perr)
		}
	}

	// Append only truly new relationships (those not already on disk).
	onDiskRelMultiset := make(map[relSig]int, len(onDisk.Relationships))
	for _, r := range onDisk.Relationships {
		onDiskRelMultiset[relSig{r.From, r.To, r.Label, r.Kind}]++
	}
	for _, rel := range s.model.Relationships {
		sig := relSig{rel.From, rel.To, rel.Label, rel.Kind}
		if onDiskRelMultiset[sig] > 0 {
			onDiskRelMultiset[sig]--
			continue
		}
		relJSON, merr := json.Marshal(rel)
		if merr != nil {
			return fmt.Errorf("marshaling relationship: %w", merr)
		}
		capturedJSON := string(relJSON)
		if perr := model.PatchInsert(s.modelPath, func(data []byte) ([]byte, error) {
			return model.AppendArrayEntry(data, []string{"relationships"}, capturedJSON)
		}); perr != nil {
			return fmt.Errorf("appending relationship: %w", perr)
		}
	}

	return nil
}

// elementsEqual returns true if two Element values are semantically identical.
// Uses JSON round-trip for a field-by-field comparison without importing reflect.
func elementsEqual(a, b model.Element) bool {
	aj, err1 := json.Marshal(a)
	bj, err2 := json.Marshal(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return string(aj) == string(bj)
}

func (s *replState) undoCommand() error {
	if len(s.undoStack) == 0 {
		fmt.Println("Nothing to undo")
		return nil
	}

	s.model = s.undoStack[len(s.undoStack)-1]
	s.undoStack = s.undoStack[:len(s.undoStack)-1]
	// Once an entry was evicted from the stack, we can no longer determine whether
	// the current in-memory state matches disk — keep modified=true in that case.
	s.modified = s.evicted || len(s.undoStack) > 0
	fmt.Println("✅ Undone")
	return nil
}

// saveUndo pushes a deep copy of the current model onto the undo stack.
// Returns true if the push succeeded (both marshal and unmarshal succeeded).
// Callers that need to roll back a no-op must check the return value before popping.
func (s *replState) saveUndo() bool {
	data, err := json.Marshal(s.model)
	if err != nil {
		return false
	}
	var snapshot model.BausteinsichtModel
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return false
	}

	s.undoStack = append(s.undoStack, &snapshot)

	if len(s.undoStack) > s.maxUndoLen {
		s.undoStack = s.undoStack[1:]
		s.evicted = true
	}
	return true
}
