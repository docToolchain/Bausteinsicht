package workspace

import (
	"fmt"
	"sort"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// validWorkspaceLayouts are the layout values a view may have (see model.View).
var validWorkspaceLayouts = map[string]bool{"": true, "layered": true, "grid": true, "none": true}

// applyWorkspaceViews adds the workspace views of cfg to merged as ordinary
// views. A workspace view selects elements across models:
//
//   - include-from lists model IDs; empty means every model,
//   - include-kinds keeps only elements of these kinds (empty: all kinds),
//   - exclude-kinds removes elements of these kinds.
//
// Every element of the imported models counts, nested ones too (a model that is
// imported partially contributes only what it imports). The merged view lists
// the selected elements by their exact merged IDs, so it does not depend on the
// wildcard rules. A view whose key is taken by a prefixed model view, that
// selects nothing, or that names an unknown model, kind or layout is an error;
// all problems are returned.
func applyWorkspaceViews(cfg *Config, loaded []LoadedModel, merged *model.BausteinsichtModel) []error {
	keys := make([]string, 0, len(cfg.Views))
	for key := range cfg.Views {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var problems []error
	for _, key := range keys {
		view := cfg.Views[key]
		ids, viewProblems := selectViewElements(key, view, loaded, merged)
		problems = append(problems, viewProblems...)
		if len(viewProblems) > 0 {
			continue
		}
		if !validWorkspaceLayouts[view.Layout] {
			problems = append(problems, fmt.Errorf("view %q: invalid layout %q (must be \"layered\", \"grid\", \"none\", or empty)", key, view.Layout))
			continue
		}
		if len(ids) == 0 {
			problems = append(problems, fmt.Errorf("view %q selects no elements", key))
			continue
		}
		if _, taken := merged.Views[key]; taken {
			problems = append(problems, collisionError("view", key))
			continue
		}
		merged.Views[key] = model.View{
			Title:       view.Title,
			Description: view.Description,
			Layout:      view.Layout,
			Include:     ids,
		}
	}
	return problems
}

// selectViewElements returns the sorted merged IDs a workspace view selects.
func selectViewElements(key string, view WorkspaceView, loaded []LoadedModel, merged *model.BausteinsichtModel) ([]string, []error) {
	var problems []error
	byID := make(map[string]LoadedModel, len(loaded))
	for _, lm := range loaded {
		byID[lm.Ref.ID] = lm
	}
	for _, kind := range append(append([]string{}, view.IncludeKinds...), view.ExcludeKinds...) {
		if _, ok := merged.Specification.Elements[kind]; !ok {
			problems = append(problems, fmt.Errorf("view %q: unknown element kind %q", key, kind))
		}
	}

	from := loaded
	if len(view.IncludeFrom) > 0 {
		from = nil
		for _, modelID := range view.IncludeFrom {
			lm, ok := byID[modelID]
			if !ok {
				problems = append(problems, fmt.Errorf("view %q: include-from refers to unknown model %q", key, modelID))
				continue
			}
			from = append(from, lm)
		}
	}
	if len(problems) > 0 {
		return nil, problems
	}

	selected := make(map[string]bool)
	for _, lm := range from {
		flat, err := model.FlattenElements(lm.Model)
		if err != nil {
			return nil, []error{fmt.Errorf("view %q: model %q: %w", key, lm.Ref.ID, err)}
		}
		prefix := resolvePrefix(lm)
		for id, elem := range flat {
			if len(view.IncludeKinds) > 0 && !contains(view.IncludeKinds, elem.Kind) {
				continue
			}
			if contains(view.ExcludeKinds, elem.Kind) {
				continue
			}
			selected[prefixElementID(id, prefix)] = true
		}
	}
	ids := make([]string, 0, len(selected))
	for id := range selected {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids, nil
}
