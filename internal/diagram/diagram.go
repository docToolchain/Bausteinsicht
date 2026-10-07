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
	scopeElems, externalElems := partitionElements(resolved, flat, view.Scope, &m.Specification)

	// Filter relationships to those visible in this view.
	elemSet := make(map[string]bool, len(resolved))
	for _, id := range resolved {
		elemSet[id] = true
	}
	if view.Scope != "" {
		elemSet[view.Scope] = true
	}
	rels := filterRelationships(m.Relationships, elemSet, &m.Specification)

	bnd := resolveBoundary(view, flat, &m.Specification)

	var b strings.Builder
	switch f {
	case PlantUML:
		writePlantUML(&b, view, level, scopeElems, externalElems, rels, bnd)
	case Mermaid:
		writeMermaid(&b, view, level, scopeElems, externalElems, rels, bnd)
	}
	return b.String(), nil
}

type elemEntry struct {
	ID    string
	Elem  *model.Element
	Macro string
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

func partitionElements(resolved []string, flat map[string]*model.Element, scope string, spec *model.Specification) (inside, outside []elemEntry) {
	for _, id := range resolved {
		elem := flat[id]
		if elem == nil {
			continue
		}
		macro, _ := C4Macro(spec, elem.Kind)
		entry := elemEntry{ID: id, Elem: elem, Macro: macro}
		if scope != "" && strings.HasPrefix(id, scope+".") {
			inside = append(inside, entry)
		} else {
			outside = append(outside, entry)
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

// c4MacroByKey maps well-known element kind keys to C4 macros.
var c4MacroByKey = map[string]string{
	"actor":           "Person",
	"person":          "Person",
	"system":          "System",
	"external_system": "System_Ext",
	"container":       "Container",
	"ui":              "Container",
	"mobile":          "Container",
	"filestore":       "Container",
	"datastore":       "ContainerDb",
	"queue":           "ContainerQueue",
	"component":       "Component",
}

// c4MacroByNotation maps lower-cased kind notations to C4 macros.
var c4MacroByNotation = map[string]string{
	"person":          "Person",
	"actor":           "Person",
	"system":          "System",
	"software system": "System",
	"external system": "System_Ext",
	"container":       "Container",
	"database":        "ContainerDb",
	"data store":      "ContainerDb",
	"datastore":       "ContainerDb",
	"queue":           "ContainerQueue",
	"component":       "Component",
}

// C4Macro resolves the C4 macro for an element kind: an explicit
// specification c4 override wins, then the kind key, then the kind's notation.
// ok is false when none matches and the macro falls back to System (#633).
func C4Macro(spec *model.Specification, kind string) (macro string, ok bool) {
	var def model.ElementKind
	if spec != nil {
		def = spec.Elements[kind]
	}
	if def.C4 != "" {
		return def.C4, true
	}
	if m, found := c4MacroByKey[strings.ToLower(kind)]; found {
		return m, true
	}
	if m, found := c4MacroByNotation[strings.ToLower(strings.TrimSpace(def.Notation))]; found {
		return m, true
	}
	return "System", false
}

// UnmappedKinds returns the sorted, de-duplicated element kinds in the view
// that C4Macro cannot resolve and that therefore render as System (#633).
func UnmappedKinds(m *model.BausteinsichtModel, viewKey string) ([]string, error) {
	view, ok := m.Views[viewKey]
	if !ok {
		return nil, fmt.Errorf("view %q not found", viewKey)
	}
	resolved, err := model.ResolveView(m, &view)
	if err != nil {
		return nil, err
	}
	flat, _ := model.FlattenElements(m)
	seen := map[string]bool{}
	var kinds []string
	for _, id := range resolved {
		elem := flat[id]
		if elem == nil || seen[elem.Kind] {
			continue
		}
		seen[elem.Kind] = true
		if _, ok := C4Macro(&m.Specification, elem.Kind); !ok {
			kinds = append(kinds, elem.Kind)
		}
	}
	sort.Strings(kinds)
	return kinds, nil
}

func sanitizeID(id string) string {
	return strings.ReplaceAll(strings.ReplaceAll(id, ".", "_"), "-", "_")
}

func escapeQuotes(s string) string {
	return strings.ReplaceAll(s, "\"", "'")
}

// writeC4Element writes one element as a C4 macro call. PlantUML-C4 and
// Mermaid's C4 diagram syntax use the same macro-call format, so both
// writePlantUML and writeMermaid share this.
func writeC4Element(b *strings.Builder, e elemEntry, indent string) {
	macro := e.Macro
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

// boundary is the C4 boundary box of a scoped view: its macro and display title.
type boundary struct {
	Macro string
	Title string
}

// resolveBoundary determines the C4 boundary macro and display title for a
// scoped view's boundary box, shared by PlantUML and Mermaid output:
// Container_Boundary when the scope element's kind resolves to a Container
// macro (via c4 override, key or notation, see C4Macro), System_Boundary
// otherwise. Falls back to the scope's raw ID as the title if the scope
// element isn't found in flat.
func resolveBoundary(view model.View, flat map[string]*model.Element, spec *model.Specification) boundary {
	scopeElem := flat[view.Scope]
	bnd := boundary{Macro: "System_Boundary", Title: view.Scope}
	if scopeElem != nil {
		bnd.Title = scopeElem.Title
		if macro, _ := C4Macro(spec, scopeElem.Kind); strings.HasPrefix(macro, "Container") {
			bnd.Macro = "Container_Boundary"
		}
	}
	return bnd
}

// writeScopeSection writes a view's scope boundary with its internal
// elements, or — for unscoped views — just the flat "inside" elements.
// Shared by writePlantUML and writeMermaid, which differ only in
// indentation convention (PlantUML: no base indent, 2-space nesting;
// Mermaid: 4-space base indent, 4-space nesting).
func writeScopeSection(b *strings.Builder, view model.View, bnd boundary, inside []elemEntry, outerIndent, innerIndent string) {
	if view.Scope == "" {
		for _, e := range inside {
			writeC4Element(b, e, outerIndent)
		}
		return
	}
	fmt.Fprintf(b, "%s%s(%s, \"%s\") {\n", outerIndent, bnd.Macro, sanitizeID(view.Scope), escapeQuotes(bnd.Title))
	for _, e := range inside {
		writeC4Element(b, e, innerIndent)
	}
	fmt.Fprintf(b, "%s}\n", outerIndent)
}

// --- PlantUML ---

func writePlantUML(b *strings.Builder, view model.View, level string, inside, outside []elemEntry, rels []relEntry, bnd boundary) {
	b.WriteString("@startuml\n")
	fmt.Fprintf(b, "!include <C4/C4_%s>\n\n", level)

	// External elements (outside scope boundary).
	for _, e := range outside {
		writeC4Element(b, e, "")
	}

	// Scope boundary with internal elements.
	writeScopeSection(b, view, bnd, inside, "", "  ")

	writePlantUMLRelationships(b, rels)

	b.WriteString("@enduml\n")
}

// writePlantUMLRelationships writes the Rel() lines for a view. Dashed ones
// bypass the C4-PlantUML Rel() macro in favor of a raw dashed arrow (..>):
// C4-PlantUML's own per-relationship line style override
// (UpdateRelStyle($lineStyle=DashedLine())) errors with "Function not found"
// on the PlantUML versions tested (#518) — a plain PlantUML arrow renders
// correctly alongside C4-macro-created elements, since element aliases are
// valid regardless of which macro created them.
func writePlantUMLRelationships(b *strings.Builder, rels []relEntry) {
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
}

// --- Mermaid ---

func writeMermaid(b *strings.Builder, view model.View, level string, inside, outside []elemEntry, rels []relEntry, bnd boundary) {
	fmt.Fprintf(b, "C4%s\n", level)
	fmt.Fprintf(b, "    title %s\n\n", view.Title)

	for _, e := range outside {
		writeC4Element(b, e, "    ")
	}

	writeScopeSection(b, view, bnd, inside, "    ", "        ")

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
