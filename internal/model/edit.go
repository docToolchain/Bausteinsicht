package model

import (
	"fmt"
	"sort"
	"strings"
)

// Change describes one location that an edit rewrote, removed or set.
type Change struct {
	Path string `json:"path"`
	From string `json:"from"`
	To   string `json:"to,omitempty"`
}

// ElementUpdate holds the element fields "update element" may change. A nil
// field is left alone.
type ElementUpdate struct {
	Title       *string
	Technology  *string
	Description *string
	Kind        *string
}

// RenameElement renames the element id to newKey (the new last path segment)
// and rewrites every reference to it and to its children: relationships, view
// scope/include/exclude, dynamic view steps and constraint selectors. The
// returned changes list every rewritten location (#634).
func (m *BausteinsichtModel) RenameElement(id, newKey string) ([]Change, error) {
	parent, key, err := m.locateElement(id)
	if err != nil {
		return nil, err
	}
	switch {
	case newKey == "":
		return nil, fmt.Errorf("the new key must not be empty")
	case strings.Contains(newKey, "."):
		return nil, fmt.Errorf("the new key %q must not contain a dot; --to takes the new last path segment", newKey)
	case newKey == key:
		return nil, fmt.Errorf("the new key %q is the same as the current one", newKey)
	}
	if _, exists := parent[newKey]; exists {
		return nil, fmt.Errorf("an element %q already exists next to %q", newKey, id)
	}

	newID := newKey
	if dot := strings.LastIndex(id, "."); dot >= 0 {
		newID = id[:dot+1] + newKey
	}
	parent[newKey] = parent[key]
	delete(parent, key)

	changes := []Change{{Path: "model." + strings.ReplaceAll(id, ".", ".children."), From: id, To: newID}}
	m.visitRefs(func(path string, ref *string) {
		if isRefTo(*ref, id) {
			rewritten := newID + (*ref)[len(id):]
			changes = append(changes, Change{Path: path, From: *ref, To: rewritten})
			*ref = rewritten
		}
	})
	return changes, nil
}

// RemoveElement removes the element id and its children. References to it
// (relationships, view include/exclude entries, dynamic view steps) make the
// removal fail unless cascade is set, which drops them too. A view scope or a
// constraint selector that names the element always blocks the removal, since
// dropping it silently would change what that view or rule means.
func (m *BausteinsichtModel) RemoveElement(id string, cascade bool) ([]Change, error) {
	parent, key, err := m.locateElement(id)
	if err != nil {
		return nil, err
	}

	var blockers, droppable []Change
	m.visitRefs(func(path string, ref *string) {
		if !isRefTo(*ref, id) {
			return
		}
		c := Change{Path: path, From: *ref}
		if strings.HasSuffix(path, ".scope") || strings.HasPrefix(path, "constraints[") {
			blockers = append(blockers, c)
		} else {
			droppable = append(droppable, c)
		}
	})
	if len(blockers) > 0 {
		return nil, fmt.Errorf("cannot remove %q: %s must be changed first", id, joinPaths(blockers))
	}
	if len(droppable) > 0 && !cascade {
		return nil, fmt.Errorf("cannot remove %q: still referenced by %s (use --cascade to drop these)", id, joinPaths(droppable))
	}

	delete(parent, key)
	changes := []Change{{Path: "model." + strings.ReplaceAll(id, ".", ".children."), From: id}}
	if cascade {
		changes = append(changes, droppable...)
		changes = append(changes, m.dropRefs(id)...)
	}
	return changes, nil
}

// RemoveRelationship removes the first relationship from -> to; if label is
// not empty it must match as well.
func (m *BausteinsichtModel) RemoveRelationship(from, to, label string) ([]Change, error) {
	for i, r := range m.Relationships {
		if r.From == from && r.To == to && (label == "" || r.Label == label) {
			m.Relationships = append(m.Relationships[:i], m.Relationships[i+1:]...)
			return []Change{{Path: fmt.Sprintf("relationships[%d]", i), From: from + " -> " + to}}, nil
		}
	}
	return nil, fmt.Errorf("relationship %s -> %s not found", from, to)
}

// UpdateElement sets the given fields of the element id. Fields that already
// have the requested value are not reported as changes.
func (m *BausteinsichtModel) UpdateElement(id string, u ElementUpdate) ([]Change, error) {
	parent, key, err := m.locateElement(id)
	if err != nil {
		return nil, err
	}
	if u.Title == nil && u.Technology == nil && u.Description == nil && u.Kind == nil {
		return nil, fmt.Errorf("nothing to update: give at least one of --title, --technology, --description, --kind")
	}
	if u.Kind != nil {
		if _, ok := m.Specification.Elements[*u.Kind]; !ok {
			return nil, fmt.Errorf("unknown element kind %q; valid kinds: %s", *u.Kind, strings.Join(sortedKeys(m.Specification.Elements), ", "))
		}
	}

	elem := parent[key]
	var changes []Change
	set := func(field string, dst *string, value *string) {
		if value != nil && *dst != *value {
			changes = append(changes, Change{Path: "model." + strings.ReplaceAll(id, ".", ".children.") + "." + field, From: *dst, To: *value})
			*dst = *value
		}
	}
	set("title", &elem.Title, u.Title)
	set("technology", &elem.Technology, u.Technology)
	set("description", &elem.Description, u.Description)
	set("kind", &elem.Kind, u.Kind)
	parent[key] = elem
	return changes, nil
}

// locateElement returns the map that holds the element id and its key in it.
func (m *BausteinsichtModel) locateElement(id string) (map[string]Element, string, error) {
	parts := strings.Split(id, ".")
	current := m.Model
	for _, part := range parts[:len(parts)-1] {
		current = current[part].Children
	}
	key := parts[len(parts)-1]
	if _, ok := current[key]; !ok {
		return nil, "", fmt.Errorf("element %q not found", id)
	}
	return current, key, nil
}

// visitRefs calls visit with the path and a pointer to every string in the
// model that names an element (or an element pattern): relationship ends, view
// scope/include/exclude, dynamic view message steps and constraint selectors.
// Changes made through the pointer are kept.
func (m *BausteinsichtModel) visitRefs(visit func(path string, ref *string)) {
	for i := range m.Relationships {
		visit(fmt.Sprintf("relationships[%d].from", i), &m.Relationships[i].From)
		visit(fmt.Sprintf("relationships[%d].to", i), &m.Relationships[i].To)
	}
	for _, key := range sortedKeys(m.Views) {
		v := m.Views[key]
		visit("views."+key+".scope", &v.Scope)
		for j := range v.Include {
			visit(fmt.Sprintf("views.%s.include[%d]", key, j), &v.Include[j])
		}
		for j := range v.Exclude {
			visit(fmt.Sprintf("views.%s.exclude[%d]", key, j), &v.Exclude[j])
		}
		m.Views[key] = v
	}
	for i := range m.DynamicViews {
		for j := range m.DynamicViews[i].Steps {
			step := &m.DynamicViews[i].Steps[j]
			if isBlockStep(step.Type) {
				continue
			}
			visit(fmt.Sprintf("dynamicViews[%d].steps[%d].from", i, j), &step.From)
			visit(fmt.Sprintf("dynamicViews[%d].steps[%d].to", i, j), &step.To)
		}
	}
	for i := range m.Constraints {
		visit(fmt.Sprintf("constraints[%d].from", i), &m.Constraints[i].From)
		visit(fmt.Sprintf("constraints[%d].to", i), &m.Constraints[i].To)
	}
}

// dropRefs removes what references the element id: relationships with it as an
// end, view include/exclude entries for it and dynamic view message steps. A
// dynamic view that is left without any message step is removed as well; those
// removals are returned.
func (m *BausteinsichtModel) dropRefs(id string) []Change {
	rels := m.Relationships[:0]
	for _, r := range m.Relationships {
		if !isRefTo(r.From, id) && !isRefTo(r.To, id) {
			rels = append(rels, r)
		}
	}
	m.Relationships = rels

	for key, v := range m.Views {
		v.Include = withoutRefs(v.Include, id)
		v.Exclude = withoutRefs(v.Exclude, id)
		m.Views[key] = v
	}
	var removedViews []Change
	views := m.DynamicViews[:0]
	for i, dv := range m.DynamicViews {
		steps := dv.Steps[:0]
		messages := 0
		for _, s := range dv.Steps {
			if isBlockStep(s.Type) {
				steps = append(steps, s)
			} else if !isRefTo(s.From, id) && !isRefTo(s.To, id) {
				steps = append(steps, s)
				messages++
			}
		}
		dv.Steps = steps
		if messages == 0 {
			removedViews = append(removedViews, Change{Path: fmt.Sprintf("dynamicViews[%d]", i), From: dv.Key})
			continue
		}
		views = append(views, dv)
	}
	m.DynamicViews = views
	return removedViews
}

func withoutRefs(list []string, id string) []string {
	var kept []string
	for _, s := range list {
		if !isRefTo(s, id) {
			kept = append(kept, s)
		}
	}
	return kept
}

// isRefTo reports whether s is the element id or a path below it, including a
// pattern such as "id.*".
func isRefTo(s, id string) bool {
	return s == id || strings.HasPrefix(s, id+".")
}

func joinPaths(changes []Change) string {
	paths := make([]string, len(changes))
	for i, c := range changes {
		paths[i] = c.Path
	}
	if len(paths) > 8 {
		paths = append(paths[:8], fmt.Sprintf("and %d more", len(changes)-8))
	}
	return strings.Join(paths, ", ")
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
