package diagram

import (
	"fmt"
	"sort"
	"strings"

	"github.com/docToolchain/Bauteinsicht/internal/model"
)

// Format represents the output diagram format.
type Format int

const (
	PlantUML Format = iota
	Mermaid
)

// C4-PlantUML is part of the PlantUML stdlib since v2.x, so we use
// the <C4/...> include syntax which resolves locally without network access.

// FormatView renders a view as a C4 diagram in the given format.
func FormatView(m *model.BausteinsichtModel, viewKey string, f Format) (string, error) {
	view, ok := m.Views[viewKey]
	if !ok {
		return "", fmt.Errorf("view %q not found", viewKey)
	}

	resolved, err := model.ResolveView(m, &view)
	if err != nil {
		return "", err
	}

	flat, _ := model.FlattenElements(m)
	sort.Strings(resolved)

	// Determine C4 level from view content.
	level := detectLevel(resolved, flat, view.Scope)

	// Separate scope-internal elements from external ones.
	scopeElems, externalElems := partitionElements(resolved, flat, view.Scope)

	// Filter relationships to those visible in this view.
	elemSet := make(map[string]bool, len(resolved))
	for _, id := range resolved {
		elemSet[id] = true
	}
	if view.Scope != "" {
		elemSet[view.Scope] = true
	}
	rels := filterRelationships(m.Relationships, elemSet)

	var b strings.Builder
	switch f {
	case PlantUML:
		writePlantUML(&b, view, level, scopeElems, externalElems, rels, flat)
	case Mermaid:
		writeMermaid(&b, view, level, scopeElems, externalElems, rels, flat)
	}
	return b.String(), nil
}

type elemEntry struct {
	ID   string
	Elem *model.Element
}

func detectLevel(resolved []string, flat map[string]*model.Element, scope string) string {
	hasContainer := false
	for _, id := range resolved {
		elem := flat[id]
		if elem == nil {
			continue
		}
		if elem.Kind == "component" {
			return "Component"
		}
		if elem.Kind == "container" {
			hasContainer = true
		}
	}
	if hasContainer || scope != "" {
		return "Container"
	}
	return "Context"
}

func partitionElements(resolved []string, flat map[string]*model.Element, scope string) (inside, outside []elemEntry) {
	for _, id := range resolved {
		elem := flat[id]
		if elem == nil {
			continue
		}
		if scope != "" && strings.HasPrefix(id, scope+".") {
			inside = append(inside, elemEntry{id, elem})
		} else {
			outside = append(outside, elemEntry{id, elem})
		}
	}
	return
}

type relEntry struct {
	From, To, Label string
}

func filterRelationships(rels []model.Relationship, elemSet map[string]bool) []relEntry {
	var result []relEntry
	seen := make(map[string]bool)
	for _, r := range rels {
		from := liftToVisible(r.From, elemSet)
		to := liftToVisible(r.To, elemSet)
		if from == "" || to == "" || from == to {
			continue
		}
		key := from + ":" + to
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, relEntry{from, to, r.Label})
	}
	return result
}

func liftToVisible(id string, elemSet map[string]bool) string {
	if elemSet[id] {
		return id
	}
	for {
		dot := strings.LastIndex(id, ".")
		if dot < 0 {
			return ""
		}
		id = id[:dot]
		if elemSet[id] {
			return id
		}
	}
}

func c4Macro(kind string) string {
	switch kind {
	case "actor":
		return "Person"
	case "system":
		return "System"
	case "external_system":
		return "System_Ext"
	case "container", "ui", "mobile":
		return "Container"
	case "datastore":
		return "ContainerDb"
	case "queue":
		return "ContainerQueue"
	case "filestore":
		return "Container"
	case "component":
		return "Component"
	default:
		return "System"
	}
}

func sanitizeID(id string) string {
	return strings.ReplaceAll(strings.ReplaceAll(id, ".", "_"), "-", "_")
}

func escapeQuotes(s string) string {
	return strings.ReplaceAll(s, "\"", "'")
}

// --- PlantUML ---

func writePlantUML(b *strings.Builder, view model.View, level string, inside, outside []elemEntry, rels []relEntry, flat map[string]*model.Element) {
	b.WriteString("@startuml\n")
	fmt.Fprintf(b, "!include <C4/C4_%s>\n\n", level)

	// External elements (outside scope boundary).
	for _, e := range outside {
		writePlantUMLElement(b, e, "")
	}

	// Scope boundary with internal elements.
	if view.Scope != "" {
		scopeElem := flat[view.Scope]
		scopeTitle := view.Scope
		if scopeElem != nil {
			scopeTitle = scopeElem.Title
		}
		boundaryMacro := "System_Boundary"
		if scopeElem != nil && scopeElem.Kind == "container" {
			boundaryMacro = "Container_Boundary"
		}
		fmt.Fprintf(b, "%s(%s, \"%s\") {\n", boundaryMacro, sanitizeID(view.Scope), escapeQuotes(scopeTitle))
		for _, e := range inside {
			writePlantUMLElement(b, e, "  ")
		}
		b.WriteString("}\n")
	} else {
		for _, e := range inside {
			writePlantUMLElement(b, e, "")
		}
	}

	// Relationships.
	if len(rels) > 0 {
		b.WriteString("\n")
	}
	for _, r := range rels {
		fmt.Fprintf(b, "Rel(%s, %s, \"%s\")\n", sanitizeID(r.From), sanitizeID(r.To), escapeQuotes(r.Label))
	}

	b.WriteString("@enduml\n")
}

func writePlantUMLElement(b *strings.Builder, e elemEntry, indent string) {
	macro := c4Macro(e.Elem.Kind)
	if e.Elem.Technology != "" {
		fmt.Fprintf(b, "%s%s(%s, \"%s\", \"%s\", \"%s\")\n",
			indent, macro, sanitizeID(e.ID),
			escapeQuotes(e.Elem.Title), escapeQuotes(e.Elem.Technology), escapeQuotes(e.Elem.Description))
	} else {
		fmt.Fprintf(b, "%s%s(%s, \"%s\", \"%s\")\n",
			indent, macro, sanitizeID(e.ID),
			escapeQuotes(e.Elem.Title), escapeQuotes(e.Elem.Description))
	}
}

// --- Mermaid ---

func writeMermaid(b *strings.Builder, view model.View, level string, inside, outside []elemEntry, rels []relEntry, flat map[string]*model.Element) {
	fmt.Fprintf(b, "C4%s\n", level)
	fmt.Fprintf(b, "    title %s\n\n", view.Title)

	for _, e := range outside {
		writeMermaidElement(b, e, "    ")
	}

	if view.Scope != "" {
		scopeElem := flat[view.Scope]
		scopeTitle := view.Scope
		if scopeElem != nil {
			scopeTitle = scopeElem.Title
		}
		boundaryMacro := "System_Boundary"
		if scopeElem != nil && scopeElem.Kind == "container" {
			boundaryMacro = "Container_Boundary"
		}
		fmt.Fprintf(b, "    %s(%s, \"%s\") {\n", boundaryMacro, sanitizeID(view.Scope), escapeQuotes(scopeTitle))
		for _, e := range inside {
			writeMermaidElement(b, e, "        ")
		}
		b.WriteString("    }\n")
	} else {
		for _, e := range inside {
			writeMermaidElement(b, e, "    ")
		}
	}

	if len(rels) > 0 {
		b.WriteString("\n")
	}
	for _, r := range rels {
		fmt.Fprintf(b, "    Rel(%s, %s, \"%s\")\n", sanitizeID(r.From), sanitizeID(r.To), escapeQuotes(r.Label))
	}
}

func writeMermaidElement(b *strings.Builder, e elemEntry, indent string) {
	macro := c4Macro(e.Elem.Kind)
	if e.Elem.Technology != "" {
		fmt.Fprintf(b, "%s%s(%s, \"%s\", \"%s\", \"%s\")\n",
			indent, macro, sanitizeID(e.ID),
			escapeQuotes(e.Elem.Title), escapeQuotes(e.Elem.Technology), escapeQuotes(e.Elem.Description))
	} else {
		fmt.Fprintf(b, "%s%s(%s, \"%s\", \"%s\")\n",
			indent, macro, sanitizeID(e.ID),
			escapeQuotes(e.Elem.Title), escapeQuotes(e.Elem.Description))
	}
}
