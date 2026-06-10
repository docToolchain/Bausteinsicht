package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/spf13/cobra"
)

var errReplExit = errors.New("exit")

func newReplCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repl",
		Short: "Interactive REPL for editing architecture model",
		Long: `Start an interactive shell for editing the architecture model.
Useful for guided model editing without writing JSONC directly.`,
		RunE: runRepl,
	}

	return cmd
}

type replState struct {
	model      *model.BausteinsichtModel
	modelPath  string
	undoStack  []*model.BausteinsichtModel
	modified   bool
	maxUndoLen int
	scanner    *bufio.Scanner // shared scanner — multiple scanners on os.Stdin lose buffered data
}

func runRepl(cmd *cobra.Command, _ []string) error {
	modelPath, _ := cmd.Flags().GetString("model")
	if modelPath == "" {
		detected, err := model.AutoDetect(".")
		if err != nil {
			return exitWithCode(fmt.Errorf("auto-detecting model: %w", err), 2)
		}
		modelPath = detected
	}

	m, err := model.Load(modelPath)
	if err != nil {
		return exitWithCode(fmt.Errorf("loading model: %w", err), 2)
	}

	state := &replState{
		model:      m,
		modelPath:  modelPath,
		undoStack:  make([]*model.BausteinsichtModel, 0),
		maxUndoLen: 50,
		scanner:    bufio.NewScanner(os.Stdin),
	}

	fmt.Printf("Bausteinsicht REPL — %s (%d elements)\n", modelPath, len(m.Model))
	fmt.Println("Type 'help' for commands, 'exit' to quit")

	for {
		fmt.Print("> ")
		if !state.scanner.Scan() {
			break // EOF
		}

		line := strings.TrimSpace(state.scanner.Text())
		if line == "" {
			continue
		}

		if err := state.executeCommand(line, cmd); err != nil {
			if errors.Is(err, errReplExit) {
				break
			}
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Error: %v\n", err)
		}
	}

	return nil
}

func (s *replState) executeCommand(line string, cmd *cobra.Command) error {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return nil
	}

	switch parts[0] {
	case "help":
		s.printHelp()
	case "list":
		if len(parts) < 2 {
			fmt.Println("Usage: list <elements|relationships|views>")
			return nil
		}
		s.listCommand(parts[1:])
	case "add":
		if len(parts) < 2 {
			fmt.Println("Usage: add <element|relationship>")
			return nil
		}
		s.addCommand(parts[1:])
	case "show":
		if len(parts) < 2 {
			fmt.Println("Usage: show <element-id>")
			return nil
		}
		s.showCommand(parts[1:])
	case "remove":
		if len(parts) < 2 {
			fmt.Println("Usage: remove element <id> | remove relationship <from> <to>")
			return nil
		}
		s.removeCommand(parts[1:])
	case "validate":
		s.validateCommand()
	case "save":
		if err := s.saveCommand(); err != nil {
			return err
		}
	case "undo":
		if err := s.undoCommand(); err != nil {
			return err
		}
	case "exit":
		return s.exitCommand()
	default:
		fmt.Printf("Unknown command: %s\n", parts[0])
	}

	return nil
}

func (s *replState) exitCommand() error {
	if !s.modified {
		return errReplExit
	}
	fmt.Print("Model has unsaved changes. Exit anyway? (yes/no): ")
	if s.scanner.Scan() && strings.ToLower(s.scanner.Text()) == "yes" {
		return errReplExit
	}
	return nil
}

func (s *replState) printHelp() {
	fmt.Print(`
Commands:
  list elements          — List all elements
  list relationships     — List all relationships
  list views             — List all views
  add element            — Add new element (guided prompts)
  add relationship       — Add new relationship (guided prompts)
  show <id>              — Show element details
  remove element <id>    — Remove top-level element
  remove relationship <from> <to> — Remove relationship
  validate               — Validate model
  save                   — Save changes to file
  undo                   — Undo last change
  exit                   — Exit REPL
  help                   — Show this help
`)
}

func (s *replState) listCommand(parts []string) {
	if len(parts) == 0 {
		return
	}

	switch parts[0] {
	case "elements":
		flat, _ := model.FlattenElements(s.model)
		fmt.Printf("\n%-30s %-15s %-40s\n", "ID", "Kind", "Title")
		fmt.Println(strings.Repeat("-", 85))
		for id, elem := range flat {
			fmt.Printf("%-30s %-15s %-40s\n", id, elem.Kind, elem.Title)
		}

	case "relationships":
		fmt.Printf("\n%-20s → %-20s %-30s\n", "From", "To", "Label")
		fmt.Println(strings.Repeat("-", 70))
		for _, rel := range s.model.Relationships {
			fmt.Printf("%-20s → %-20s %-30s\n", rel.From, rel.To, rel.Label)
		}

	case "views":
		fmt.Printf("\n%-20s %-50s\n", "Key", "Title")
		fmt.Println(strings.Repeat("-", 70))
		for key, view := range s.model.Views {
			fmt.Printf("%-20s %-50s\n", key, view.Title)
		}
	}
	fmt.Println()
}

func (s *replState) addCommand(parts []string) {
	if len(parts) == 0 {
		return
	}

	switch parts[0] {
	case "element":
		s.addElementInteractive()
	case "relationship":
		s.addRelationshipInteractive()
	}
}

func (s *replState) addElementInteractive() {
	fmt.Print("Element ID: ")
	s.scanner.Scan()
	id := strings.TrimSpace(s.scanner.Text())
	if id == "" {
		fmt.Println("Aborted (empty ID)")
		return
	}
	if !isValidID(id) {
		fmt.Printf("Invalid ID %q: must contain only letters, digits, hyphens, or underscores\n", id)
		return
	}

	if _, exists := s.model.Model[id]; exists {
		fmt.Printf("Element '%s' already exists. Overwrite? (yes/no): ", id)
		s.scanner.Scan()
		if strings.ToLower(strings.TrimSpace(s.scanner.Text())) != "yes" {
			fmt.Println("Aborted")
			return
		}
	}

	fmt.Print("Kind: ")
	s.scanner.Scan()
	kind := strings.TrimSpace(s.scanner.Text())
	if kind == "" {
		fmt.Println("Aborted (kind must not be empty)")
		return
	}

	fmt.Print("Title: ")
	s.scanner.Scan()
	title := strings.TrimSpace(s.scanner.Text())
	if title == "" {
		fmt.Println("Aborted (title must not be empty)")
		return
	}

	fmt.Print("Description (optional): ")
	s.scanner.Scan()
	desc := strings.TrimSpace(s.scanner.Text())

	s.saveUndo()
	s.model.Model[id] = model.Element{
		Kind:        kind,
		Title:       title,
		Description: desc,
	}
	s.modified = true
	fmt.Printf("✅ Added element '%s'\n", id)
}

func (s *replState) addRelationshipInteractive() {
	fmt.Print("From (element ID): ")
	s.scanner.Scan()
	from := strings.TrimSpace(s.scanner.Text())
	if from == "" {
		fmt.Println("Aborted (from must not be empty)")
		return
	}

	fmt.Print("To (element ID): ")
	s.scanner.Scan()
	to := strings.TrimSpace(s.scanner.Text())
	if to == "" {
		fmt.Println("Aborted (to must not be empty)")
		return
	}

	flat, _ := model.FlattenElements(s.model)
	if _, ok := flat[from]; !ok {
		fmt.Printf("Element '%s' not found in model\n", from)
		return
	}
	if _, ok := flat[to]; !ok {
		fmt.Printf("Element '%s' not found in model\n", to)
		return
	}

	fmt.Print("Label (optional): ")
	s.scanner.Scan()
	label := strings.TrimSpace(s.scanner.Text())

	s.saveUndo()
	s.model.Relationships = append(s.model.Relationships, model.Relationship{
		From:  from,
		To:    to,
		Label: label,
	})
	s.modified = true
	fmt.Printf("✅ Added relationship %s → %s\n", from, to)
}

func (s *replState) showCommand(parts []string) {
	if len(parts) == 0 {
		return
	}

	id := parts[0]
	flat, _ := model.FlattenElements(s.model)

	if elem, ok := flat[id]; ok {
		data, _ := json.MarshalIndent(elem, "", "  ")
		fmt.Printf("\nElement: %s\n%s\n\n", id, string(data))
		return
	}

	fmt.Printf("Element '%s' not found\n", id)
}

func (s *replState) removeCommand(parts []string) {
	if len(parts) < 2 {
		return
	}

	s.saveUndo()

	switch parts[0] {
	case "element":
		id := parts[1]
		// Only top-level elements can be removed this way; nested children
		// (dot-path IDs like "shop.api") must be edited in the JSONC directly.
		if _, exists := s.model.Model[id]; !exists {
			fmt.Printf("Element '%s' not found (nested elements must be edited in the model file)\n", id)
			s.undoStack = s.undoStack[:len(s.undoStack)-1] // discard no-op undo entry
			return
		}
		delete(s.model.Model, id)
		s.modified = true
		fmt.Printf("✅ Removed element '%s'\n", id)

	case "relationship":
		if len(parts) < 3 {
			fmt.Println("Usage: remove relationship <from> <to>")
			s.undoStack = s.undoStack[:len(s.undoStack)-1]
			return
		}
		from, to := parts[1], parts[2]
		removed := false
		rels := s.model.Relationships[:0]
		for _, r := range s.model.Relationships {
			if r.From == from && r.To == to {
				removed = true
				continue
			}
			rels = append(rels, r)
		}
		s.model.Relationships = rels
		if removed {
			s.modified = true
			fmt.Printf("✅ Removed relationship %s → %s\n", from, to)
		} else {
			fmt.Printf("Relationship %s → %s not found\n", from, to)
			s.undoStack = s.undoStack[:len(s.undoStack)-1]
		}
	}
}

func (s *replState) validateCommand() {
	errs := model.Validate(s.model)
	if len(errs) == 0 {
		fmt.Println("✅ Model valid")
		return
	}

	fmt.Printf("❌ %d validation errors:\n", len(errs))
	for _, err := range errs {
		fmt.Printf("  %s\n", err.Error())
	}
}

func (s *replState) saveCommand() error {
	err := s.patchSave()
	if err != nil {
		// Fall back to full save (loses comments) when patch is not possible.
		fmt.Printf("(note: comments may not be preserved — %v)\n", err)
		if err2 := model.Save(s.modelPath, s.model); err2 != nil {
			return err2
		}
	}
	s.modified = false
	s.undoStack = nil
	fmt.Printf("✅ Saved to %s\n", s.modelPath)
	return nil
}

// patchSave writes only the changes between the on-disk model and the in-memory
// model using comment-preserving patch operations. Returns an error if the diff
// contains deletions (which require a full rewrite) or if any patch fails.
func (s *replState) patchSave() error {
	onDisk, err := model.Load(s.modelPath)
	if err != nil {
		return fmt.Errorf("reading model: %w", err)
	}

	// Reject deletions — they require a full rewrite.
	for id := range onDisk.Model {
		if _, ok := s.model.Model[id]; !ok {
			return fmt.Errorf("element %q was deleted; full save required", id)
		}
	}
	if len(s.model.Relationships) < len(onDisk.Relationships) {
		return fmt.Errorf("relationships were deleted; full save required")
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

	// Append new relationships.
	type relKey struct{ from, to string }
	onDiskRels := make(map[relKey]bool, len(onDisk.Relationships))
	for _, r := range onDisk.Relationships {
		onDiskRels[relKey{r.From, r.To}] = true
	}
	for _, rel := range s.model.Relationships {
		if onDiskRels[relKey{rel.From, rel.To}] {
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

func (s *replState) undoCommand() error {
	if len(s.undoStack) == 0 {
		fmt.Println("Nothing to undo")
		return nil
	}

	s.model = s.undoStack[len(s.undoStack)-1]
	s.undoStack = s.undoStack[:len(s.undoStack)-1]
	s.modified = len(s.undoStack) > 0
	fmt.Println("✅ Undone")
	return nil
}

func (s *replState) saveUndo() {
	// Deep copy current model state
	data, err := json.Marshal(s.model)
	if err != nil {
		return
	}
	var copy model.BausteinsichtModel
	_ = json.Unmarshal(data, &copy)

	s.undoStack = append(s.undoStack, &copy)

	// Trim undo stack to max length
	if len(s.undoStack) > s.maxUndoLen {
		s.undoStack = s.undoStack[1:]
	}
}
