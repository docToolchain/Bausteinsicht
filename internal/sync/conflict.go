package sync

import "fmt"

// Conflict represents a field that was changed on both sides since the last sync.
type Conflict struct {
	ElementID     string
	Field         string // "title", "description", "technology"
	ModelValue    string
	DrawioValue   string
	LastSyncValue string
}

// ResolvedConflict is a conflict with its resolution decision.
type ResolvedConflict struct {
	Conflict
	Winner  string // "model" or "drawio"
	Warning string // human-readable warning message
}

// ConflictResolver resolves conflicts between model and draw.io changes.
// Designed as an interface for future extension (interactive, merge strategies).
type ConflictResolver interface {
	Resolve(conflicts []Conflict) []ResolvedConflict
}

// ModelWinsResolver always picks the model value (v1 default strategy).
type ModelWinsResolver struct{}

// NewModelWinsResolver creates a new ModelWinsResolver.
func NewModelWinsResolver() *ModelWinsResolver {
	return &ModelWinsResolver{}
}

// Resolve resolves all conflicts by choosing the model value.
func (r *ModelWinsResolver) Resolve(conflicts []Conflict) []ResolvedConflict {
	resolved := make([]ResolvedConflict, 0, len(conflicts))
	for _, c := range conflicts {
		warning := fmt.Sprintf(
			"WARNING: Conflict detected for element %q:\n"+
				"  Field: %s\n"+
				"  Model value:   %q\n"+
				"  draw.io value: %q\n"+
				"  Last sync:     %q\n"+
				"  → Keeping model value. Edit draw.io manually if needed.",
			c.ElementID, c.Field, c.ModelValue, c.DrawioValue, c.LastSyncValue,
		)
		resolved = append(resolved, ResolvedConflict{
			Conflict: c,
			Winner:   "model",
			Warning:  warning,
		})
	}
	return resolved
}
