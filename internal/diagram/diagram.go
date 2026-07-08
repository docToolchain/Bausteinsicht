package diagram

import (
	"fmt"
	"sort"
	"strings"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// Format represents the output diagram format.
type Format int

const (
	PlantUML Format = iota
	Mermaid
)

// C4-PlantUML is part of the PlantUML stdlib since v2.x, so we use
// the <C4/...> include syntax which resolves locally without network access.

// applyTagFiltering filters resolved element IDs based on tag criteria.
// Elements must have ALL filterTags (intersection) and must not have ANY excludeTags (union).
func applyTagFiltering(resolved []string, flat map[string]*model.Element, filterTags, excludeTags []string) []string {
	if len(filterTags) == 0 && len(excludeTags) == 0 {
		return resolved
	}

	var result []string
	for _, id := range resolved {
		elem := flat[id]
		if elem == nil {
			// Element not found in flat map, skip it (shouldn't happen)
			continue
		}

		// Check exclude tags: if ANY exclude-tag matches, skip
		excluded := false
		for _, excludeTag := range excludeTags {
			for _, elemTag := range elem.Tags {
				if elemTag == excludeTag {
					excluded = true
					break
				}
			}
			if excluded {
				break
			}
		}
		if excluded {
			continue
		}

		// Check filter tags: if ANY filter-tags are specified, element must have ALL of them
		if len(filterTags) > 0 {
			hasAllFilterTags := true
			for _, filterTag := range filterTags {
				found := false
				for _, elemTag := range elem.Tags {
					if elemTag == filterTag {
						found = true
						break
					}
				}
				if !found {
					hasAllFilterTags = false
					break
				}
			}
			if !hasAllFilterTags {
				continue
			}
		}

		result = append(result, id)
	}

	return result
}

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

	// Apply tag-based filtering if specified in the view.
	resolved = applyTagFiltering(resolved, flat, view.FilterTags, view.ExcludeTags)
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
	rels := filterRelationships(m.Relationships, elemSet, &m.Specification)

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
	From   string `json:"from"`
	To     string `json:"to"`
	Label  string `json:"label,omitempty"`
	Dashed bool   `json:"dashed,omitempty"`
}

// filterRelationships lifts relationship endpoints to visible elements,
// deduplicates by (from, to) pair, and resolves each relationship's Dashed
// flag from its kind in spec.Relationships (#518). When multiple
// relationships collapse onto the same (from, to) pair (e.g. via endpoint
// lifting), the rendered connector is dashed if any of them is, regardless
// of iteration order.
func filterRelationships(rels []model.Relationship, elemSet map[string]bool, spec *model.Specification) []relEntry {
	var result []relEntry
	seenAt := make(map[string]int) // key -> index into result
	for _, r := range rels {
		from := liftToVisible(r.From, elemSet)
		to := liftToVisible(r.To, elemSet)
		if from == "" || to == "" || from == to {
			continue
		}
		key := from + ":" + to
		dashed := spec.IsDashed(r.Kind)
		if idx, ok := seenAt[key]; ok {
			if dashed {
				result[idx].Dashed = true
			}
			continue
		}
		seenAt[key] = len(result)
		result = append(result, relEntry{from, to, r.Label, dashed})
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

	// Relationships. Dashed ones bypass the C4-PlantUML Rel() macro in favor
	// of a raw dashed arrow (..>): C4-PlantUML's own per-relationship line
	// style override (UpdateRelStyle($lineStyle=DashedLine())) errors with
	// "Function not found" on the PlantUML versions tested (#518) — a plain
	// PlantUML arrow renders correctly alongside C4-macro-created elements,
	// since element aliases are valid regardless of which macro created them.
	if len(rels) > 0 {
		b.WriteString("\n")
	}
	for _, r := range rels {
		if r.Dashed {
			fmt.Fprintf(b, "%s ..> %s : \"%s\"\n", sanitizeID(r.From), sanitizeID(r.To), escapeQuotes(r.Label))
		} else {
			fmt.Fprintf(b, "Rel(%s, %s, \"%s\")\n", sanitizeID(r.From), sanitizeID(r.To), escapeQuotes(r.Label))
		}
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

	// Relationships. r.Dashed is intentionally not applied here: Mermaid's
	// own C4 diagram docs mark UpdateRelStyle's $lineStyle=DashedLine() as
	// "not yet implemented" (https://mermaid.js.org/syntax/c4.html, checked
	// 2026-07 — #518) — there is currently no dashed-line mechanism in
	// Mermaid's C4 syntax to fall back to (unlike PlantUML, C4 diagrams
	// don't support mixing in a raw arrow outside the C4 macro set). Revisit
	// once Mermaid ships the feature.
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

// ExportAllViewsToMermaid exports all views from the model as Mermaid diagrams.
// Returns a slice of view keys in order and a map of view key → Mermaid diagram code.
func ExportAllViewsToMermaid(m *model.BausteinsichtModel) ([]string, map[string]string, error) {
	diagrams := make(map[string]string)
	var viewKeys []string

	for viewKey := range m.Views {
		viewKeys = append(viewKeys, viewKey)
	}
	sort.Strings(viewKeys)

	for _, viewKey := range viewKeys {
		diagramCode, err := FormatView(m, viewKey, Mermaid)
		if err != nil {
			return nil, nil, err
		}
		diagrams[viewKey] = diagramCode
	}

	return viewKeys, diagrams, nil
}
