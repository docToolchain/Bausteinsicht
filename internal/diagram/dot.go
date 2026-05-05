package diagram

import (
	"fmt"
	"sort"
	"strings"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// RenderDOT renders a view as a GraphViz DOT graph.
func RenderDOT(m *model.BausteinsichtModel, viewKey string) (string, error) {
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

	// Filter elements visible in this view
	elemSet := make(map[string]bool, len(resolved))
	for _, id := range resolved {
		elemSet[id] = true
	}
	if view.Scope != "" {
		elemSet[view.Scope] = true
	}

	// Filter relationships
	rels := filterRelationships(m.Relationships, elemSet)

	var b strings.Builder
	b.WriteString("digraph \"" + escapeQuotes(view.Title) + "\" {\n")
	b.WriteString("  rankdir=LR\n")
	b.WriteString("  node [shape=box style=filled fontname=\"Arial\" fontsize=9]\n")
	b.WriteString("  edge [fontsize=8]\n\n")

	// Write nodes
	for _, id := range resolved {
		elem := flat[id]
		if elem == nil {
			continue
		}

		style := ColorForKind(elem.Kind)
		nodeID := sanitizeID(id)

		label := elem.Title
		if elem.Title == "" {
			label = id
		}
		if elem.Kind != "" {
			label = label + "\n[" + elem.Kind + "]"
		}

		fmt.Fprintf(&b, "  %s [label=\"%s\" fillcolor=\"%s\" color=\"%s\"]\n",
			nodeID, escapeQuotes(label), style.Fill, style.Stroke)
	}

	// Write relationships
	if len(rels) > 0 {
		b.WriteString("\n")
		for _, r := range rels {
			fromID := sanitizeID(r.From)
			toID := sanitizeID(r.To)
			if r.Label != "" {
				fmt.Fprintf(&b, "  %s -> %s [label=\"%s\"]\n", fromID, toID, escapeQuotes(r.Label))
			} else {
				fmt.Fprintf(&b, "  %s -> %s\n", fromID, toID)
			}
		}
	}

	b.WriteString("}\n")
	return b.String(), nil
}
