package diagram

import (
	"fmt"
	"sort"
	"strings"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// RenderD2 renders a view as a D2 diagram.
func RenderD2(m *model.BausteinsichtModel, viewKey string) (string, error) {
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
	rels := filterRelationships(m.Relationships, elemSet, &m.Specification)

	var b strings.Builder
	b.WriteString("direction: right\n\n")

	// Write nodes
	for _, id := range resolved {
		elem := flat[id]
		if elem == nil {
			continue
		}

		style := ColorForKind(elem.Kind)
		nodeID := sanitizeD2ID(id)

		title := elem.Title
		if title == "" {
			title = id
		}

		// Node with styling
		// The kind goes into the label: D2 does not accept text after the quoted
		// label (#713).
		nodeLabel := escapeD2String(title)
		if elem.Kind != "" {
			nodeLabel = "\"" + escapeD2Inner(title) + "\\n[" + escapeD2Inner(elem.Kind) + "]\""
		}
		fmt.Fprintf(&b, "%s: %s {\n", nodeID, nodeLabel)
		fmt.Fprintf(&b, "  shape: rectangle\n")
		fmt.Fprintf(&b, "  style.fill: \"%s\"\n", style.Fill)
		fmt.Fprintf(&b, "  style.stroke: \"%s\"\n", style.Stroke)
		if elem.Description != "" {
			fmt.Fprintf(&b, "  note: %s\n", escapeD2String(elem.Description))
		}
		b.WriteString("}\n\n")
	}

	// Write relationships. Dashed ones use D2's edge style block
	// (style.stroke-dash) — always paired with an explicit (possibly empty)
	// label, since that's the documented/verified form for attaching a
	// style block to an edge.
	for _, r := range rels {
		fromID := sanitizeD2ID(r.From)
		toID := sanitizeD2ID(r.To)
		label := r.displayLabel()
		switch {
		case r.Dashed:
			fmt.Fprintf(&b, "%s -> %s: %s {\n  style.stroke-dash: 3\n}\n", fromID, toID, escapeD2String(label))
		case label != "":
			fmt.Fprintf(&b, "%s -> %s: %s\n", fromID, toID, escapeD2String(label))
		default:
			fmt.Fprintf(&b, "%s -> %s\n", fromID, toID)
		}
	}

	return b.String(), nil
}

// sanitizeD2ID converts a dot-notation ID to a valid D2 identifier.
func sanitizeD2ID(id string) string {
	s := strings.ReplaceAll(id, ".", "_")
	s = strings.ReplaceAll(s, "-", "_")
	return s
}

// escapeD2String escapes a string for use in D2 string literals.
func escapeD2String(s string) string {
	return "\"" + escapeD2Inner(s) + "\""
}

// escapeD2Inner escapes s for the inside of a double-quoted D2 string.
func escapeD2Inner(s string) string {
	return strings.ReplaceAll(s, "\"", "\\\"")
}
