package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/spf13/cobra"
)

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
	}

	fmt.Printf("Bausteinsicht REPL — %s (%d elements)\n", modelPath, len(m.Model))
	fmt.Println("Type 'help' for commands, 'exit' to quit")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break // EOF
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if err := state.executeCommand(line, cmd); err != nil {
			if err.Error() == "exit" {
				break
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "Error: %v\n", err)
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
		} else {
			s.listCommand(parts[1:])
		}
	case "add":
		if len(parts) < 2 {
			fmt.Println("Usage: add <element|relationship>")
		} else {
			s.addCommand(parts[1:])
		}
	case "show":
		if len(parts) < 2 {
			fmt.Println("Usage: show <element-id>")
		} else {
			s.showCommand(parts[1:])
		}
	case "remove":
		if len(parts) < 3 {
			fmt.Println("Usage: remove <element|relationship> <id>")
		} else {
			s.removeCommand(parts[1:])
		}
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
		if s.modified {
			fmt.Print("Model has unsaved changes. Exit anyway? (yes/no): ")
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() && strings.ToLower(scanner.Text()) == "yes" {
				return fmt.Errorf("exit")
			}
			return nil
		}
		return fmt.Errorf("exit")
	default:
		fmt.Printf("Unknown command: %s\n", parts[0])
	}

	return nil
}

func (s *replState) printHelp() {
	fmt.Println(`
Commands:
  list elements          — List all elements
  list relationships     — List all relationships
  list views             — List all views
  add element            — Add new element (guided prompts)
  add relationship       — Add new relationship (guided prompts)
  show <id>              — Show element details
  remove element <id>    — Remove element
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

	s.saveUndo()

	switch parts[0] {
	case "element":
		s.addElementInteractive()
	case "relationship":
		s.addRelationshipInteractive()
	}
}

func (s *replState) addElementInteractive() {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("Element ID: ")
	scanner.Scan()
	id := strings.TrimSpace(scanner.Text())
	if id == "" {
		fmt.Println("Aborted (empty ID)")
		return
	}

	fmt.Print("Kind: ")
	scanner.Scan()
	kind := strings.TrimSpace(scanner.Text())

	fmt.Print("Title: ")
	scanner.Scan()
	title := strings.TrimSpace(scanner.Text())

	fmt.Print("Description (optional): ")
	scanner.Scan()
	desc := strings.TrimSpace(scanner.Text())

	s.model.Model[id] = model.Element{
		Kind:        kind,
		Title:       title,
		Description: desc,
	}

	s.modified = true
	fmt.Printf("✅ Added element '%s'\n", id)
}

func (s *replState) addRelationshipInteractive() {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("From (element ID): ")
	scanner.Scan()
	from := strings.TrimSpace(scanner.Text())

	fmt.Print("To (element ID): ")
	scanner.Scan()
	to := strings.TrimSpace(scanner.Text())

	fmt.Print("Label (optional): ")
	scanner.Scan()
	label := strings.TrimSpace(scanner.Text())

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
		delete(s.model.Model, id)
		s.modified = true
		fmt.Printf("✅ Removed element '%s'\n", id)
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
	if err := model.Save(s.modelPath, s.model); err != nil {
		return err
	}
	s.modified = false
	fmt.Printf("✅ Saved to %s\n", s.modelPath)
	return nil
}

func (s *replState) undoCommand() error {
	if len(s.undoStack) == 0 {
		fmt.Println("Nothing to undo")
		return nil
	}

	s.model = s.undoStack[len(s.undoStack)-1]
	s.undoStack = s.undoStack[:len(s.undoStack)-1]
	fmt.Println("✅ Undone")
	return nil
}

func (s *replState) saveUndo() {
	// Deep copy current model state
	data, _ := json.Marshal(s.model)
	var copy model.BausteinsichtModel
	json.Unmarshal(data, &copy)

	s.undoStack = append(s.undoStack, &copy)

	// Trim undo stack to max length
	if len(s.undoStack) > s.maxUndoLen {
		s.undoStack = s.undoStack[1:]
	}
}
