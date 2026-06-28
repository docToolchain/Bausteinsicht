package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
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
	evicted    bool // true once an undo entry was dropped due to maxUndoLen; modified can never go back to false
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
			if err := state.scanner.Err(); err != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Error reading input: %v\n", err)
			}
			if state.modified {
				fmt.Println("Warning: exiting with unsaved changes (EOF)")
			}
			break
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
			fmt.Println("Usage: remove element <id> | remove relationship <from> <to> [label]")
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
	if s.scanner.Scan() && strings.ToLower(strings.TrimSpace(s.scanner.Text())) == "yes" {
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
  remove element <id>              — Remove top-level element
  remove relationship <from> <to> [label] — Remove relationship
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
		ids := make([]string, 0, len(flat))
		for id := range flat {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		fmt.Printf("\n%-30s %-15s %-40s\n", "ID", "Kind", "Title")
		fmt.Println(strings.Repeat("-", 85))
		for _, id := range ids {
			elem := flat[id]
			fmt.Printf("%-30s %-15s %-40s\n", id, elem.Kind, elem.Title)
		}

	case "relationships":
		fmt.Printf("\n%-20s → %-20s %-30s\n", "From", "To", "Label")
		fmt.Println(strings.Repeat("-", 70))
		rels := make([]model.Relationship, len(s.model.Relationships))
		copy(rels, s.model.Relationships)
		sort.Slice(rels, func(i, j int) bool {
			if rels[i].From != rels[j].From {
				return rels[i].From < rels[j].From
			}
			if rels[i].To != rels[j].To {
				return rels[i].To < rels[j].To
			}
			return rels[i].Label < rels[j].Label
		})
		for _, rel := range rels {
			fmt.Printf("%-20s → %-20s %-30s\n", rel.From, rel.To, rel.Label)
		}

	case "views":
		keys := make([]string, 0, len(s.model.Views))
		for k := range s.model.Views {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		fmt.Printf("\n%-20s %-50s\n", "Key", "Title")
		fmt.Println(strings.Repeat("-", 70))
		for _, key := range keys {
			fmt.Printf("%-20s %-50s\n", key, s.model.Views[key].Title)
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

	if existing, exists := s.model.Model[id]; exists {
		fmt.Printf("Element '%s' already exists.\n", id)
		fmt.Printf("  Updating:  kind, title, description\n")
		preserved := []string{}
		if len(existing.Children) > 0 {
			preserved = append(preserved, fmt.Sprintf("%d child(ren)", len(existing.Children)))
		}
		if existing.Technology != "" {
			preserved = append(preserved, "technology")
		}
		if len(existing.Tags) > 0 {
			preserved = append(preserved, "tags")
		}
		if existing.Status != "" {
			preserved = append(preserved, "status")
		}
		if len(existing.Decisions) > 0 {
			preserved = append(preserved, "decisions")
		}
		if len(existing.Metadata) > 0 {
			preserved = append(preserved, "metadata")
		}
		if len(preserved) > 0 {
			fmt.Printf("  Preserving: %s\n", strings.Join(preserved, ", "))
		}
		fmt.Print("Overwrite? (yes/no): ")
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
	if len(s.model.Specification.Elements) > 0 {
		if _, ok := s.model.Specification.Elements[kind]; !ok {
			fmt.Printf("Unknown kind %q; valid kinds: %s\n", kind, validKinds(s.model))
			return
		}
	}

	fmt.Print("Title: ")
	s.scanner.Scan()
	title := strings.TrimSpace(s.scanner.Text())
	if title == "" {
		fmt.Println("Aborted (title must not be empty)")
		return
	}

	existingDesc := s.model.Model[id].Description
	if existingDesc != "" {
		fmt.Printf("Description (optional) [%s]: ", existingDesc)
	} else {
		fmt.Print("Description (optional): ")
	}
	s.scanner.Scan()
	desc := strings.TrimSpace(s.scanner.Text())
	if desc == "" {
		desc = existingDesc
	}

	if !s.saveUndo() {
		fmt.Println("(warning: undo not available for this change)")
	}
	if s.model.Model == nil {
		s.model.Model = make(map[string]model.Element)
	}
	updated := s.model.Model[id] // zero value if new; existing value if overwriting
	updated.Kind = kind
	updated.Title = title
	updated.Description = desc
	s.model.Model[id] = updated
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

	for _, r := range s.model.Relationships {
		if r.From == from && r.To == to && r.Label == label {
			fmt.Printf("Relationship %s → %s (label: %q) already exists\n", from, to, label)
			return
		}
	}

	if !s.saveUndo() {
		fmt.Println("(warning: undo not available for this change)")
	}
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

	pushed := s.saveUndo()

	popUndo := func() {
		if pushed && len(s.undoStack) > 0 {
			s.undoStack = s.undoStack[:len(s.undoStack)-1]
		}
	}

	switch parts[0] {
	case "element":
		id := parts[1]
		// Only top-level elements can be removed this way; nested children
		// (dot-path IDs like "shop.api") must be edited in the JSONC directly.
		if _, exists := s.model.Model[id]; !exists {
			fmt.Printf("Element '%s' not found (nested elements must be edited in the model file)\n", id)
			popUndo()
			return
		}
		delete(s.model.Model, id)
		s.modified = true
		fmt.Printf("✅ Removed element '%s'\n", id)

	case "relationship":
		if len(parts) < 3 {
			fmt.Println("Usage: remove relationship <from> <to> [label]")
			popUndo()
			return
		}
		from, to := parts[1], parts[2]
		wantLabel := ""
		if len(parts) >= 4 {
			wantLabel = strings.Join(parts[3:], " ")
		}
		removed := false
		rels := s.model.Relationships[:0]
		for _, r := range s.model.Relationships {
			if !removed && r.From == from && r.To == to && (wantLabel == "" || r.Label == wantLabel) {
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
			popUndo()
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
