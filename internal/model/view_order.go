package model

import "sort"

// OrderedViewKeys returns the keys of m.Views in the order their pages are
// written, exported and listed: views without a scope (landscape and context
// views) first, then the scoped ones, each group sorted by key. The model file
// is read into a map, so the declaration order is not available, and Go's map
// order would change the page order from one run to the next (#683).
func OrderedViewKeys(m *BausteinsichtModel) []string {
	keys := make([]string, 0, len(m.Views))
	for k := range m.Views {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]
		aScoped, bScoped := m.Views[a].Scope != "", m.Views[b].Scope != ""
		if aScoped != bScoped {
			return !aScoped
		}
		return a < b
	})
	return keys
}
