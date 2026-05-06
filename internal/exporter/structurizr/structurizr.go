// Package structurizr converts a BausteinsichtModel to Structurizr DSL format.
package structurizr

import (
	"fmt"
	"sort"
	"strings"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// Export converts m to a Structurizr DSL workspace string.
//
// Variable names prefer the leaf key (e.g. "webApp") and fall back to the
// full dot-path-with-underscores ("orderSystem_webApp") only when the leaf
// key is ambiguous across the whole model. This ensures a clean roundtrip:
// re-importing the output reconstructs the same dot-paths.
func Export(m *model.BausteinsichtModel) string {
	flat, _ := model.FlattenElements(m)
	varMap := buildVarMap(flat)
	e := &exporter{m: m, flat: flat, varMap: varMap}

	var b strings.Builder
	b.WriteString("workspace {\n")
	b.WriteString("    model {\n")

	// Write root elements (sorted for deterministic output).
	for _, key := range sortedKeys(m.Model) {
		elem := m.Model[key]
		e.writeElement(&b, key, elem, "        ")
	}

	// Write global relationships.
	if len(m.Relationships) > 0 {
		b.WriteString("\n")
		for _, r := range m.Relationships {
			fromVar := varMap[r.From]
			toVar := varMap[r.To]
			if r.Label != "" {
				fmt.Fprintf(&b, "        %s -> %s \"%s\"\n", fromVar, toVar, escDQ(r.Label))
			} else {
				fmt.Fprintf(&b, "        %s -> %s\n", fromVar, toVar)
			}
		}
	}

	b.WriteString("    }\n\n")
	b.WriteString("    views {\n")
	e.writeViews(&b, "        ")
	b.WriteString("    }\n")
	b.WriteString("}\n")

	return b.String()
}

type exporter struct {
	m      *model.BausteinsichtModel
	flat   map[string]*model.Element
	varMap map[string]string // dot-path → Structurizr variable name
}

// writeElement writes one element (and recursively its children) to b.
// dotPath is the full dot-separated path (e.g. "orderSystem.webApp").
// The variable name is looked up from e.varMap.
func (e *exporter) writeElement(b *strings.Builder, dotPath string, elem model.Element, indent string) {
	varName := e.varMap[dotPath]
	kind := toStructurizrKind(elem.Kind)
	desc := escDQ(elem.Description)
	tech := escDQ(elem.Technology)
	title := escDQ(elem.Title)
	if title == "" {
		title = varName
	}

	hasChildren := len(elem.Children) > 0

	if hasChildren {
		if tech != "" {
			fmt.Fprintf(b, "%s%s = %s \"%s\" \"%s\" \"%s\" {\n", indent, varName, kind, title, tech, desc)
		} else if desc != "" {
			fmt.Fprintf(b, "%s%s = %s \"%s\" \"%s\" {\n", indent, varName, kind, title, desc)
		} else {
			fmt.Fprintf(b, "%s%s = %s \"%s\" {\n", indent, varName, kind, title)
		}
		for _, childKey := range sortedKeys(elem.Children) {
			childElem := elem.Children[childKey]
			childDotPath := dotPath + "." + childKey
			e.writeElement(b, childDotPath, childElem, indent+"    ")
		}
		fmt.Fprintf(b, "%s}\n", indent)
	} else {
		if tech != "" {
			fmt.Fprintf(b, "%s%s = %s \"%s\" \"%s\" \"%s\"\n", indent, varName, kind, title, tech, desc)
		} else if desc != "" {
			fmt.Fprintf(b, "%s%s = %s \"%s\" \"%s\"\n", indent, varName, kind, title, desc)
		} else {
			fmt.Fprintf(b, "%s%s = %s \"%s\"\n", indent, varName, kind, title)
		}
	}
}

func (e *exporter) writeViews(b *strings.Builder, indent string) {
	if len(e.m.Views) == 0 {
		return
	}
	for _, key := range sortedKeys(e.m.Views) {
		v := e.m.Views[key]
		e.writeOneView(b, key, v, indent)
	}
}

func (e *exporter) writeOneView(b *strings.Builder, key string, v model.View, indent string) {
	viewType := e.detectViewType(v)
	title := escDQ(v.Title)
	if title == "" {
		title = key
	}

	if viewType == "systemLandscape" || v.Scope == "" {
		fmt.Fprintf(b, "%ssystemLandscape \"%s\" \"%s\" {\n", indent, key, title)
	} else {
		scopeVar := e.varMap[v.Scope]
		if scopeVar == "" {
			// Scope exists in flat map (verified by detectViewType), use its variable name
			scopeVar = dotToVar(v.Scope)
		}
		fmt.Fprintf(b, "%s%s %s \"%s\" \"%s\" {\n", indent, viewType, scopeVar, key, title)
	}
	fmt.Fprintf(b, "%s    include *\n", indent)
	fmt.Fprintf(b, "%s}\n", indent)
}

// detectViewType returns the Structurizr view type keyword for v.
func (e *exporter) detectViewType(v model.View) string {
	if v.Scope == "" {
		return "systemLandscape"
	}
	scopeElem := e.flat[v.Scope]
	if scopeElem == nil {
		return "systemContext"
	}
	if isContainerKind(scopeElem.Kind) {
		// Scope is a container → component view (shows what's inside a container).
		return "component"
	}
	// System-kind scope: if the scope element has container-kind children it's a container view.
	for _, child := range scopeElem.Children {
		if isContainerKind(child.Kind) {
			return "container"
		}
	}
	return "systemContext"
}

// toStructurizrKind maps a Bausteinsicht element kind to a Structurizr keyword.
func toStructurizrKind(kind string) string {
	switch kind {
	case "actor", "person":
		return "person"
	case "system", "external_system":
		return "softwareSystem"
	case "container", "ui", "mobile", "datastore", "queue", "filestore":
		return "container"
	case "component":
		return "component"
	default:
		return "softwareSystem"
	}
}

// isContainerKind reports whether kind is one of the Structurizr "container" equivalents.
func isContainerKind(kind string) bool {
	switch kind {
	case "container", "ui", "mobile", "datastore", "queue", "filestore":
		return true
	}
	return false
}

// buildVarMap assigns a Structurizr variable name to every element dot-path.
// Leaf keys are used when globally unique; otherwise the full
// dot-path-with-underscores is used to avoid collisions.
func buildVarMap(flat map[string]*model.Element) map[string]string {
	leafCount := make(map[string]int, len(flat))
	for id := range flat {
		parts := strings.Split(id, ".")
		leafCount[parts[len(parts)-1]]++
	}

	varMap := make(map[string]string, len(flat))
	for id := range flat {
		parts := strings.Split(id, ".")
		leaf := parts[len(parts)-1]
		if leafCount[leaf] == 1 {
			varMap[id] = leaf
		} else {
			varMap[id] = dotToVar(id)
		}
	}
	return varMap
}

// dotToVar converts a dot-path to a valid Structurizr variable name.
func dotToVar(path string) string {
	return strings.ReplaceAll(path, ".", "_")
}

// escDQ escapes double quotes and newlines in s for embedding in a Structurizr string literal.
func escDQ(s string) string {
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return s
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
