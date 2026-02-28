package sync

import (
	"github.com/docToolchain/Bauteinsicht/internal/drawio"
	"github.com/docToolchain/Bauteinsicht/internal/model"
)

// SyncResult contains the comprehensive result of a sync cycle.
type SyncResult struct {
	Forward   *ForwardResult
	Reverse   *ReverseResult
	Conflicts []ResolvedConflict
	Warnings  []string
}

// Run executes one full bidirectional sync cycle.
// It is a pure function — no file I/O. All data is passed as parameters.
//
// Sequence (Chapter 6 - Runtime View):
//  1. DetectChanges → ChangeSet
//  2. Resolve conflicts (model wins)
//  3. Remove conflicting fields from DrawioElementChanges
//  4. ApplyForward → ForwardResult
//  5. ApplyReverse → ReverseResult
//  6. Collect all warnings
func Run(
	m *model.BausteinsichtModel,
	doc *drawio.Document,
	lastState *SyncState,
	templates *drawio.TemplateSet,
) *SyncResult {
	result := &SyncResult{}

	// Step 1: Detect changes from both sides.
	changes := DetectChanges(m, doc, lastState)

	// Step 2: Resolve conflicts.
	if len(changes.Conflicts) > 0 {
		resolved := NewModelWinsResolver().Resolve(changes.Conflicts)
		result.Conflicts = resolved

		// Step 3: For model-wins conflicts, drop the draw.io change so that
		// ApplyReverse does not overwrite the model value.
		changes.DrawioElementChanges = filterConflictingDrawioChanges(
			changes.DrawioElementChanges, resolved,
		)
	}

	// Step 4: Forward sync (model → draw.io).
	result.Forward = ApplyForward(changes, doc, templates, m)

	// Step 5: Reverse sync (draw.io → model).
	result.Reverse = ApplyReverse(changes, m)

	// Step 6: Collect all warnings.
	result.Warnings = append(result.Warnings, result.Forward.Warnings...)
	result.Warnings = append(result.Warnings, result.Reverse.Warnings...)
	for _, rc := range result.Conflicts {
		result.Warnings = append(result.Warnings, rc.Warning)
	}

	return result
}

// filterConflictingDrawioChanges removes draw.io element changes for fields
// that were resolved in favour of the model (Winner == "model").
func filterConflictingDrawioChanges(
	drawioChanges []ElementChange,
	resolved []ResolvedConflict,
) []ElementChange {
	// Build a set of (elementID, field) pairs that model won.
	type conflictKey struct {
		id    string
		field string
	}
	modelWins := make(map[conflictKey]struct{}, len(resolved))
	for _, rc := range resolved {
		if rc.Winner == "model" {
			modelWins[conflictKey{rc.ElementID, rc.Field}] = struct{}{}
		}
	}

	if len(modelWins) == 0 {
		return drawioChanges
	}

	filtered := make([]ElementChange, 0, len(drawioChanges))
	for _, ch := range drawioChanges {
		if _, skip := modelWins[conflictKey{ch.ID, ch.Field}]; !skip {
			filtered = append(filtered, ch)
		}
	}
	return filtered
}
