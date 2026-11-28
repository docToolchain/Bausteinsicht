package sync

import (
	"fmt"
	"sort"
	"strings"

	"github.com/docToolchain/Bausteinsicht/internal/drawio"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// SyncResult contains the comprehensive result of a sync cycle.
type SyncResult struct {
	Forward   *ForwardResult
	Reverse   *ReverseResult
	Changes   *ChangeSet // The (post-conflict-resolution) changes used for sync.
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
	newPageIDs map[string]bool,
	opts ...ForwardOptions,
) *SyncResult {
	result := &SyncResult{}

	// Step 0: A copy of an element cell carries the same bausteinsicht_id;
	// it must not be read as the element (#722).
	result.Warnings = append(result.Warnings, stripDuplicateElementIDs(doc)...)

	// Step 1: Detect changes from both sides.
	// Pass newPageIDs so that elements expected only on newly created pages
	// are not mistakenly treated as "deleted from draw.io" (#184, #188, #189).
	changes := DetectChanges(m, doc, lastState, newPageIDs)

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

	result.Changes = changes
	result.Warnings = append(result.Warnings, changes.Warnings...)

	// Step 4: Forward sync (model → draw.io).
	result.Forward = ApplyForward(changes, doc, templates, m, opts...)

	// Step 5: Reverse sync (draw.io → model).
	result.Reverse = ApplyReverse(changes, m, lastState)

	// Step 5b: Reverse sync changed the model on behalf of draw.io; bring the
	// document in line with the model as it is now, in this same run:
	//  - a title, description, technology or connector label edited on one page
	//    must show on every other page, otherwise those pages are stale and the
	//    next sync would read them as edits of their own (#684);
	//  - relationships removed by reverse sync shifted the index of the ones
	//    behind them and may free a pair for a lifted connector, and an element
	//    it was asked to remove may have been kept (scope of a view, constraint)
	//    (#719, #720).
	// This is the forward pipeline with the changes reverse sync applied, so the
	// next sync finds nothing to do.
	if settle := settleChangeSet(changes); hasDrawioDeletions(changes) || len(settle.ModelElementChanges)+len(settle.ModelRelationshipChanges) > 0 {
		settled := ApplyForward(settle, doc, templates, m, opts...)
		// The writes of an edit made in draw.io are not "model → draw.io updates";
		// only structural changes are counted.
		result.Forward.ElementsCreated += settled.ElementsCreated
		result.Forward.ConnectorsCreated += settled.ConnectorsCreated
		result.Forward.ConnectorsDeleted += settled.ConnectorsDeleted
		result.Forward.MetadataUpdated += settled.MetadataUpdated
	}

	// Step 6: Warn about model elements not visible in any view (#183).
	if len(m.Views) > 0 {
		visible := computeVisibleElements(m)
		flat, _ := model.FlattenElements(m)
		var invisible []string
		for id := range flat {
			if visible != nil && !visible[id] {
				invisible = append(invisible, id)
			}
		}
		if len(invisible) > 0 {
			sort.Strings(invisible)
			for _, id := range invisible {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("Element %q exists in the model but is not visible in any view — add it to a view's include list", id))
			}
		}
	}

	// Step 7: Collect all warnings.
	result.Warnings = append(result.Warnings, result.Forward.Warnings...)
	result.Warnings = append(result.Warnings, result.Reverse.Warnings...)
	for _, rc := range result.Conflicts {
		result.Warnings = append(result.Warnings, rc.Warning)
	}

	return result
}

// settleChangeSet turns the field edits that reverse sync took over from draw.io
// into model-side changes, so a forward pass writes them to every page.
func settleChangeSet(cs *ChangeSet) *ChangeSet {
	settle := &ChangeSet{}
	for _, ch := range cs.DrawioElementChanges {
		if ch.Type == Modified && ch.Field != "" {
			settle.ModelElementChanges = append(settle.ModelElementChanges, ch)
		}
	}
	for _, ch := range cs.DrawioRelationshipChanges {
		if ch.Type == Modified && ch.Field == fieldLabel {
			settle.ModelRelationshipChanges = append(settle.ModelRelationshipChanges, ch)
		}
	}
	return settle
}

// hasDrawioDeletions reports whether cs contains an element or relationship
// deleted in draw.io.
func hasDrawioDeletions(cs *ChangeSet) bool {
	for _, ch := range cs.DrawioElementChanges {
		if ch.Type == Deleted {
			return true
		}
	}
	for _, ch := range cs.DrawioRelationshipChanges {
		if ch.Type == Deleted {
			return true
		}
	}
	return false
}

// RemoveOrphanedViewPages removes pages from the draw.io document that were
// created for views that no longer exist in the model. Pages are identified as
// view-managed if their id starts with the "view-" prefix. Pages whose id does
// not start with "view-" are preserved (e.g., default template pages).
func RemoveOrphanedViewPages(doc *drawio.Document, m *model.BausteinsichtModel) {
	// Build the set of expected view page IDs from the model.
	expectedPages := make(map[string]bool, len(m.Views))
	for viewID := range m.Views {
		expectedPages["view-"+viewID] = true
	}

	// Iterate pages and collect orphaned view page IDs.
	var orphans []string
	for _, page := range doc.Pages() {
		pageID := page.ID()
		if !strings.HasPrefix(pageID, "view-") {
			continue // Not a view-managed page; preserve it.
		}
		if !expectedPages[pageID] {
			orphans = append(orphans, pageID)
		}
	}

	// Remove orphaned pages.
	for _, id := range orphans {
		doc.RemovePage(id)
	}
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
