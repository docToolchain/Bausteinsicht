package constraints

import (
	"fmt"
	"strings"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// noRelationship enforces that no relationship exists from any element of
// fromKind to any element of toKind.
func noRelationship(c model.Constraint, m *model.BausteinsichtModel) []Violation {
	flat, err := model.FlattenElements(m)
	if err != nil {
		return []Violation{{ConstraintID: c.ID, Message: err.Error()}}
	}
	kindOf := buildKindMap(flat)

	var bad []string
	for _, rel := range m.Relationships {
		if kindOf[rel.From] == c.FromKind && kindOf[rel.To] == c.ToKind {
			bad = append(bad, fmt.Sprintf("%s → %s", rel.From, rel.To))
		}
	}
	if len(bad) == 0 {
		return nil
	}
	return []Violation{{
		ConstraintID: c.ID,
		Message:      fmt.Sprintf("%s: %s kind must not relate to %s kind", c.Description, c.FromKind, c.ToKind),
		Elements:     bad,
	}}
}

// allowedRelationship enforces that only elements whose kind is in fromKinds
// may have relationships pointing to elements of toKind.
func allowedRelationship(c model.Constraint, m *model.BausteinsichtModel) []Violation {
	flat, err := model.FlattenElements(m)
	if err != nil {
		return []Violation{{ConstraintID: c.ID, Message: err.Error()}}
	}
	kindOf := buildKindMap(flat)

	allowed := make(map[string]bool, len(c.FromKinds))
	for _, k := range c.FromKinds {
		allowed[k] = true
	}

	var bad []string
	for _, rel := range m.Relationships {
		if kindOf[rel.To] == c.ToKind && !allowed[kindOf[rel.From]] {
			bad = append(bad, fmt.Sprintf("%s (%s) → %s", rel.From, kindOf[rel.From], rel.To))
		}
	}
	if len(bad) == 0 {
		return nil
	}
	return []Violation{{
		ConstraintID: c.ID,
		Message:      fmt.Sprintf("%s: only [%s] may relate to %s kind", c.Description, strings.Join(c.FromKinds, ", "), c.ToKind),
		Elements:     bad,
	}}
}

// requiredField enforces that all elements of elementKind have the given field
// set to a non-empty value. Supported fields: "description", "technology", "title".
func requiredField(c model.Constraint, m *model.BausteinsichtModel) []Violation {
	flat, err := model.FlattenElements(m)
	if err != nil {
		return []Violation{{ConstraintID: c.ID, Message: err.Error()}}
	}

	var bad []string
	for id, el := range flat {
		if el.Kind != c.ElementKind {
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
		Message:      fmt.Sprintf("%s: all %s elements must have %q set", c.Description, c.ElementKind, c.Field),
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

// buildKindMap returns a map from element ID to its kind for all flattened elements.
func buildKindMap(flat map[string]*model.Element) map[string]string {
	m := make(map[string]string, len(flat))
	for id, el := range flat {
		m[id] = el.Kind
	}
	return m
}
