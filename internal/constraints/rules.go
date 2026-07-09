package constraints

import (
	"fmt"
	"strings"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// endpointSelector describes how a single endpoint ("from" or "to") of a
// no-relationship/allowed-relationship constraint targets elements: by exact
// ID, by tag, or by kind — checked in that order of specificity (#542).
// Whichever field is non-empty is used; if none is set, matches nothing.
type endpointSelector struct {
	id   string
	tag  string
	kind string
}

func (s endpointSelector) describe() string {
	switch {
	case s.id != "":
		return fmt.Sprintf("element %q", s.id)
	case s.tag != "":
		return fmt.Sprintf("%q tag", s.tag)
	case s.kind != "":
		return fmt.Sprintf("%q kind", s.kind)
	default:
		return "(unset)"
	}
}

func (s endpointSelector) matches(elemID string, el *model.Element) bool {
	switch {
	case s.id != "":
		return elemID == s.id
	case s.tag != "":
		return el != nil && hasTag(el.Tags, s.tag)
	case s.kind != "":
		return el != nil && el.Kind == s.kind
	default:
		return false
	}
}

// hasTag reports whether tags contains tag.
func hasTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

// noRelationship enforces that no relationship exists from any element
// matching the "from" selector to any element matching the "to" selector.
func noRelationship(c model.Constraint, m *model.BausteinsichtModel) []Violation {
	flat, err := model.FlattenElements(m)
	if err != nil {
		return []Violation{{ConstraintID: c.ID, Message: err.Error()}}
	}
	from := endpointSelector{id: c.From, tag: c.FromTag, kind: c.FromKind}
	to := endpointSelector{id: c.To, tag: c.ToTag, kind: c.ToKind}

	var bad []string
	for _, rel := range m.Relationships {
		if from.matches(rel.From, flat[rel.From]) && to.matches(rel.To, flat[rel.To]) {
			bad = append(bad, fmt.Sprintf("%s → %s", rel.From, rel.To))
		}
	}
	if len(bad) == 0 {
		return nil
	}
	return []Violation{{
		ConstraintID: c.ID,
		Message:      fmt.Sprintf("%s: %s must not relate to %s", c.Description, from.describe(), to.describe()),
		Elements:     bad,
	}}
}

// allowedRelationship enforces that only elements matching the "from"
// allow-list (by tag if FromTags is set, otherwise by kind via FromKinds)
// may have relationships pointing to elements matching the "to" selector.
func allowedRelationship(c model.Constraint, m *model.BausteinsichtModel) []Violation {
	flat, err := model.FlattenElements(m)
	if err != nil {
		return []Violation{{ConstraintID: c.ID, Message: err.Error()}}
	}
	to := endpointSelector{id: c.To, tag: c.ToTag, kind: c.ToKind}

	var allowed func(elemID string, el *model.Element) bool
	var allowDescription string
	switch {
	case len(c.FromTags) > 0:
		tagSet := make(map[string]bool, len(c.FromTags))
		for _, t := range c.FromTags {
			tagSet[t] = true
		}
		allowed = func(_ string, el *model.Element) bool {
			if el == nil {
				return false
			}
			for _, t := range el.Tags {
				if tagSet[t] {
					return true
				}
			}
			return false
		}
		allowDescription = fmt.Sprintf("tagged [%s]", strings.Join(c.FromTags, ", "))
	default:
		kindSet := make(map[string]bool, len(c.FromKinds))
		for _, k := range c.FromKinds {
			kindSet[k] = true
		}
		allowed = func(_ string, el *model.Element) bool {
			return el != nil && kindSet[el.Kind]
		}
		allowDescription = fmt.Sprintf("[%s] kind", strings.Join(c.FromKinds, ", "))
	}

	var bad []string
	for _, rel := range m.Relationships {
		if to.matches(rel.To, flat[rel.To]) && !allowed(rel.From, flat[rel.From]) {
			fromKind := ""
			if el := flat[rel.From]; el != nil {
				fromKind = el.Kind
			}
			bad = append(bad, fmt.Sprintf("%s (%s) → %s", rel.From, fromKind, rel.To))
		}
	}
	if len(bad) == 0 {
		return nil
	}
	return []Violation{{
		ConstraintID: c.ID,
		Message:      fmt.Sprintf("%s: only %s may relate to %s", c.Description, allowDescription, to.describe()),
		Elements:     bad,
	}}
}

// requiredField enforces that all elements matching the constraint's element
// selector (by Tag if set, otherwise by ElementKind) have the given field set
// to a non-empty value. Supported fields: "description", "technology", "title".
func requiredField(c model.Constraint, m *model.BausteinsichtModel) []Violation {
	flat, err := model.FlattenElements(m)
	if err != nil {
		return []Violation{{ConstraintID: c.ID, Message: err.Error()}}
	}

	selectorDescription := c.ElementKind + " kind"
	matchesElement := func(el *model.Element) bool { return el.Kind == c.ElementKind }
	if c.Tag != "" {
		selectorDescription = fmt.Sprintf("%q-tagged", c.Tag)
		matchesElement = func(el *model.Element) bool { return hasTag(el.Tags, c.Tag) }
	}

	var bad []string
	for id, el := range flat {
		if !matchesElement(el) {
			continue
		}
		var missing bool
		switch c.Field {
		case "description":
			missing = el.Description == ""
		case "technology":
			missing = el.Technology == ""
		case "title":
			missing = el.Title == ""
		default:
			// Unsupported field name — return error violation immediately
			return []Violation{{
				ConstraintID: c.ID,
				Message:      fmt.Sprintf("%s: unsupported field %q (valid: description, technology, title)", c.Description, c.Field),
			}}
		}
		if missing {
			bad = append(bad, fmt.Sprintf("%s: missing %s", id, c.Field))
		}
	}
	if len(bad) == 0 {
		return nil
	}
	return []Violation{{
		ConstraintID: c.ID,
		Message:      fmt.Sprintf("%s: all %s elements must have %q set", c.Description, selectorDescription, c.Field),
		Elements:     bad,
	}}
}

// maxDepth enforces that no element is nested deeper than max levels.
// Root-level elements have depth 1.
func maxDepth(c model.Constraint, m *model.BausteinsichtModel) []Violation {
	var bad []string
	walkDepth(m.Model, 1, c.Max, &bad)
	if len(bad) == 0 {
		return nil
	}
	return []Violation{{
		ConstraintID: c.ID,
		Message:      fmt.Sprintf("%s: maximum nesting depth is %d", c.Description, c.Max),
		Elements:     bad,
	}}
}

func walkDepth(elements map[string]model.Element, depth, max int, bad *[]string) {
	for id, el := range elements {
		if depth > max {
			*bad = append(*bad, fmt.Sprintf("%s (depth %d)", id, depth))
		}
		if len(el.Children) > 0 {
			walkDepth(el.Children, depth+1, max, bad)
		}
	}
}

// noCircularDependency detects cycles in the relationship graph using DFS.
func noCircularDependency(c model.Constraint, m *model.BausteinsichtModel) []Violation {
	// Build adjacency list.
	adj := make(map[string][]string)
	flat, err := model.FlattenElements(m)
	if err != nil {
		return []Violation{{ConstraintID: c.ID, Message: err.Error()}}
	}
	for id := range flat {
		adj[id] = nil
	}
	for _, rel := range m.Relationships {
		adj[rel.From] = append(adj[rel.From], rel.To)
	}

	visited := make(map[string]bool)
	inStack := make(map[string]bool)
	var cycles []string

	var dfs func(node string, path []string)
	dfs = func(node string, path []string) {
		visited[node] = true
		inStack[node] = true
		path = append(path, node)

		for _, neighbour := range adj[node] {
			if !visited[neighbour] {
				dfs(neighbour, path)
			} else if inStack[neighbour] {
				// Found a cycle — record the loop segment.
				for i, n := range path {
					if n == neighbour {
						cycle := strings.Join(append(path[i:], neighbour), " → ")
						cycles = append(cycles, cycle)
						break
					}
				}
			}
		}
		inStack[node] = false
	}

	for node := range adj {
		if !visited[node] {
			dfs(node, nil)
		}
	}

	if len(cycles) == 0 {
		return nil
	}
	return []Violation{{
		ConstraintID: c.ID,
		Message:      c.Description + ": circular dependencies detected",
		Elements:     cycles,
	}}
}

// technologyAllowed enforces that elements of elementKind only use technologies
// from the given allowed list.
func technologyAllowed(c model.Constraint, m *model.BausteinsichtModel) []Violation {
	flat, err := model.FlattenElements(m)
	if err != nil {
		return []Violation{{ConstraintID: c.ID, Message: err.Error()}}
	}
	allowed := make(map[string]bool, len(c.Technologies))
	for _, t := range c.Technologies {
		allowed[strings.ToLower(t)] = true
	}

	var bad []string
	for id, el := range flat {
		if el.Kind != c.ElementKind {
			continue
		}
		if el.Technology == "" {
			continue // technology not set — use required-field rule to enforce that separately
		}
		if !allowed[strings.ToLower(el.Technology)] {
			bad = append(bad, fmt.Sprintf("%s: technology %q not in allowed list [%s]",
				id, el.Technology, strings.Join(c.Technologies, ", ")))
		}
	}
	if len(bad) == 0 {
		return nil
	}
	return []Violation{{
		ConstraintID: c.ID,
		Message:      fmt.Sprintf("%s: %s elements must use one of [%s]", c.Description, c.ElementKind, strings.Join(c.Technologies, ", ")),
		Elements:     bad,
	}}
}
