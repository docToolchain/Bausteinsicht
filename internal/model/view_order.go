package model

// OrderedViewKeys returns the keys of m.Views in the order their pages are
// written.
func OrderedViewKeys(m *BausteinsichtModel) []string {
	keys := make([]string, 0, len(m.Views))
	for k := range m.Views {
		keys = append(keys, k)
	}
	return keys
}
