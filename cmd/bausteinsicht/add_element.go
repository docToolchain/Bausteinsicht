package main

import (
	"encoding/json"
	"fmt"

	"github.com/docToolchain/Bauteinsicht/internal/model"
	"github.com/spf13/cobra"
)

func newAddElementCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "element",
		Short: "Add an element to the model",
		RunE:  runAddElement,
	}

	cmd.Flags().String("id", "", "Unique identifier for the element (required)")
	cmd.Flags().String("kind", "", "Element kind as defined in specification (required)")
	cmd.Flags().String("title", "", "Display title (required)")
	cmd.Flags().String("parent", "", "Parent element ID (dot notation)")
	cmd.Flags().String("technology", "", "Technology description")
	cmd.Flags().String("description", "", "Element description")

	_ = cmd.MarkFlagRequired("id")
	_ = cmd.MarkFlagRequired("kind")
	_ = cmd.MarkFlagRequired("title")

	return cmd
}

func runAddElement(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetString("id")
	kind, _ := cmd.Flags().GetString("kind")
	title, _ := cmd.Flags().GetString("title")
	parent, _ := cmd.Flags().GetString("parent")
	technology, _ := cmd.Flags().GetString("technology")
	description, _ := cmd.Flags().GetString("description")

	modelPath, _ := cmd.Flags().GetString("model")
	format, _ := cmd.Flags().GetString("format")

	// Load model
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

	// Validate kind
	if _, ok := m.Specification.Elements[kind]; !ok {
		return exitWithCode(
			fmt.Errorf("unknown element kind %q; valid kinds: %s", kind, validKinds(m)),
			1,
		)
	}

	// Build element
	elem := model.Element{
		Kind:        kind,
		Title:       title,
		Technology:  technology,
		Description: description,
	}

	fullID := id

	if parent != "" {
		// Validate parent exists
		parentElem, err := model.Resolve(m, parent)
		if err != nil {
			return exitWithCode(fmt.Errorf("parent %q not found: %w", parent, err), 1)
		}

		// Validate parent's kind allows children.
		if spec, ok := m.Specification.Elements[parentElem.Kind]; !ok || !spec.Container {
			return exitWithCode(
				fmt.Errorf("element %q (kind: %s) is not a container and cannot have children", parent, parentElem.Kind),
				1,
			)
		}

		// Check duplicate within parent's children
		if parentElem.Children != nil {
			if _, exists := parentElem.Children[id]; exists {
				return exitWithCode(fmt.Errorf("element %q already exists under %q", id, parent), 1)
			}
		}

		// Add to parent's children — need to update in-place through the model
		if err := addChildToParent(m, parent, id, elem); err != nil {
			return exitWithCode(err, 1)
		}

		fullID = parent + "." + id
	} else {
		// Check duplicate at top level
		if _, exists := m.Model[id]; exists {
			return exitWithCode(fmt.Errorf("element %q already exists at top level", id), 1)
		}

		if m.Model == nil {
			m.Model = make(map[string]model.Element)
		}
		m.Model[id] = elem
	}

	// Save model — use comment-preserving insertion. (#122)
	if err := saveAddedElement(modelPath, m, fullID, parent, id, elem); err != nil {
		return exitWithCode(fmt.Errorf("saving model: %w", err), 2)
	}

	// Output
	if format == "json" {
		out := map[string]string{
			"id":    fullID,
			"kind":  kind,
			"title": title,
		}
		if technology != "" {
			out["technology"] = technology
		}
		if description != "" {
			out["description"] = description
		}
		data, _ := json.Marshal(out)
		fmt.Println(string(data))
	} else {
		fmt.Printf("Added element '%s' (kind: %s) to model.\n", fullID, kind)
	}

	return nil
}

// addChildToParent traverses the model to the parent and adds the child element.
func addChildToParent(m *model.BausteinsichtModel, parentPath, childID string, child model.Element) error {
	parts := splitDotPath(parentPath)

	// Get root element
	root, ok := m.Model[parts[0]]
	if !ok {
		return fmt.Errorf("element %q not found", parts[0])
	}

	if len(parts) == 1 {
		if root.Children == nil {
			root.Children = make(map[string]model.Element)
		}
		root.Children[childID] = child
		m.Model[parts[0]] = root
		return nil
	}

	// Traverse to parent, building a stack of elements to update
	stack := []model.Element{root}
	current := root
	for _, part := range parts[1:] {
		if current.Children == nil {
			return fmt.Errorf("no children at %q", part)
		}
		next, ok := current.Children[part]
		if !ok {
			return fmt.Errorf("element %q not found", part)
		}
		stack = append(stack, next)
		current = next
	}

	// Add child to the deepest parent
	if current.Children == nil {
		current.Children = make(map[string]model.Element)
	}
	current.Children[childID] = child
	stack[len(stack)-1] = current

	// Walk back up the stack updating parents
	for i := len(stack) - 1; i > 0; i-- {
		parentElem := stack[i-1]
		if parentElem.Children == nil {
			parentElem.Children = make(map[string]model.Element)
		}
		parentElem.Children[parts[i]] = stack[i]
		stack[i-1] = parentElem
	}

	m.Model[parts[0]] = stack[0]
	return nil
}

func splitDotPath(path string) []string {
	result := []string{}
	current := ""
	for _, c := range path {
		if c == '.' {
			if current != "" {
				result = append(result, current)
			}
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

// saveAddedElement saves a newly added element using comment-preserving
// insertion. Falls back to model.Save if patching fails. (#122)
func saveAddedElement(modelPath string, m *model.BausteinsichtModel, fullID, parent, id string, elem model.Element) error {
	elemJSON := marshalElementJSON(elem)

	// Build the object path for insertion.
	var objectPath []string
	if parent != "" {
		// Insert into the parent's "children" object.
		parts := splitDotPath(parent)
		objectPath = append([]string{"model"}, parts...)
		objectPath = append(objectPath, "children")
	} else {
		objectPath = []string{"model"}
	}

	err := model.PatchInsert(modelPath, func(data []byte) ([]byte, error) {
		return model.InsertObjectEntry(data, objectPath, id, elemJSON)
	})
	if err != nil {
		// Fall back to full save if patching fails.
		return model.Save(modelPath, m)
	}
	return nil
}

// marshalElementJSON builds a compact JSON object for an element.
func marshalElementJSON(elem model.Element) string {
	parts := []string{fmt.Sprintf(`"kind": %q`, elem.Kind)}
	parts = append(parts, fmt.Sprintf(`"title": %q`, elem.Title))
	if elem.Technology != "" {
		parts = append(parts, fmt.Sprintf(`"technology": %q`, elem.Technology))
	}
	if elem.Description != "" {
		parts = append(parts, fmt.Sprintf(`"description": %q`, elem.Description))
	}

	result := "{\n"
	for i, p := range parts {
		result += "      " + p
		if i < len(parts)-1 {
			result += ","
		}
		result += "\n"
	}
	result += "    }"
	return result
}

func validKinds(m *model.BausteinsichtModel) string {
	kinds := ""
	for k := range m.Specification.Elements {
		if kinds != "" {
			kinds += ", "
		}
		kinds += k
	}
	return kinds
}
