package workspace

import (
	"fmt"
	"strings"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// MergeModels combines multiple models into a single unified model.
// Element IDs are prefixed to avoid collisions:
// - If ModelRef.Prefix is set, use it as prefix
// - Otherwise, use ModelRef.ID as prefix
// Cross-model relationships are resolved using prefixed IDs.
func MergeModels(loaded []LoadedModel) (*model.BausteinsichtModel, error) {
	if len(loaded) == 0 {
		return nil, fmt.Errorf("no models to merge")
	}

	merged := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements:      make(map[string]model.ElementKind),
			Relationships: make(map[string]model.RelationshipKind),
		},
		Model:         make(map[string]model.Element),
		Relationships: []model.Relationship{},
		Views:         make(map[string]model.View),
		DynamicViews:  []model.DynamicView{},
		Constraints:   []model.Constraint{},
	}

	// Merge specifications (element and relationship kinds)
	for _, lm := range loaded {
		for kind, def := range lm.Model.Specification.Elements {
			if _, exists := merged.Specification.Elements[kind]; !exists {
				merged.Specification.Elements[kind] = def
			}
		}
		for kind, def := range lm.Model.Specification.Relationships {
			if _, exists := merged.Specification.Relationships[kind]; !exists {
				merged.Specification.Relationships[kind] = def
			}
		}
	}

	// Map to track ID transformations for relationship resolution
	idMap := make(map[string]string) // original ID → prefixed ID

	// Merge elements with prefixing
	for _, lm := range loaded {
		prefix := lm.Ref.Prefix
		if prefix == "" {
			prefix = lm.Ref.ID
		}

		flatElems, _ := model.FlattenElements(lm.Model)
		for id, elemPtr := range flatElems {
			prefixedID := prefixElementID(id, prefix)
			idMap[id] = prefixedID

			merged.Model[prefixedID] = *elemPtr
		}
	}

	// Merge relationships with ID remapping
	for _, lm := range loaded {
		prefix := lm.Ref.Prefix
		if prefix == "" {
			prefix = lm.Ref.ID
		}

		for _, rel := range lm.Model.Relationships {
			remappedRel := rel
			remappedRel.From = prefixElementID(rel.From, prefix)
			remappedRel.To = prefixElementID(rel.To, prefix)
			merged.Relationships = append(merged.Relationships, remappedRel)
		}
	}

	// Merge views (each model's views are prefixed with model ID)
	for _, lm := range loaded {
		prefix := lm.Ref.Prefix
		if prefix == "" {
			prefix = lm.Ref.ID
		}

		for viewID, view := range lm.Model.Views {
			viewKey := prefix + "_" + viewID
			remappedView := view
			remappedView.Include = remapElementIDs(view.Include, prefix)
			remappedView.Exclude = remapElementIDs(view.Exclude, prefix)
			if view.Scope != "" {
				remappedView.Scope = prefixElementID(view.Scope, prefix)
			}
			merged.Views[viewKey] = remappedView
		}
	}

	// Merge dynamic views
	for _, lm := range loaded {
		prefix := lm.Ref.Prefix
		if prefix == "" {
			prefix = lm.Ref.ID
		}

		for _, dv := range lm.Model.DynamicViews {
			remappedDV := dv
			remappedDV.Key = prefix + "_" + dv.Key
			for i := range remappedDV.Steps {
				remappedDV.Steps[i].From = prefixElementID(remappedDV.Steps[i].From, prefix)
				remappedDV.Steps[i].To = prefixElementID(remappedDV.Steps[i].To, prefix)
			}
			merged.DynamicViews = append(merged.DynamicViews, remappedDV)
		}
	}

	// Merge constraints
	for _, lm := range loaded {
		prefix := lm.Ref.Prefix
		if prefix == "" {
			prefix = lm.Ref.ID
		}

		for _, constraint := range lm.Model.Constraints {
			remappedConstraint := constraint
			if constraint.FromKind != "" {
				remappedConstraint.FromKind = constraint.FromKind
			}
			merged.Constraints = append(merged.Constraints, remappedConstraint)
		}
	}

	return merged, nil
}

// prefixElementID adds a prefix to an element ID.
// Dot-notation paths like "a.b.c" become "prefix_a.b.c".
func prefixElementID(id, prefix string) string {
	parts := strings.SplitN(id, ".", 2)
	return prefix + "_" + parts[0] + func() string {
		if len(parts) > 1 {
			return "." + parts[1]
		}
		return ""
	}()
}

// remapElementIDs applies prefixing to a list of element IDs.
func remapElementIDs(ids []string, prefix string) []string {
	var result []string
	for _, id := range ids {
		result = append(result, prefixElementID(id, prefix))
	}
	return result
}

// ResolveWorkspaceView resolves a workspace view by expanding element filters
// across all loaded models and returning a unified element set.
func ResolveWorkspaceView(cfg *Config, loaded []LoadedModel, view *WorkspaceView) (map[string]*model.Element, error) {
	merged, err := MergeModels(loaded)
	if err != nil {
		return nil, err
	}

	flatElems, _ := model.FlattenElements(merged)
	result := make(map[string]*model.Element)

	// Start with includes
	if len(view.IncludeFrom) > 0 {
		// Include from specific models
		for _, modelID := range view.IncludeFrom {
			for _, lm := range loaded {
				if lm.Ref.ID == modelID {
					prefix := lm.Ref.Prefix
					if prefix == "" {
						prefix = lm.Ref.ID
					}
					flatLM, _ := model.FlattenElements(lm.Model)
					for id, elemPtr := range flatLM {
						prefixedID := prefixElementID(id, prefix)
						if len(view.IncludeKinds) == 0 || contains(view.IncludeKinds, elemPtr.Kind) {
							result[prefixedID] = elemPtr
						}
					}
					break
				}
			}
		}
	} else if len(view.IncludeKinds) > 0 {
		// Include by kinds across all models
		for id, elemPtr := range flatElems {
			if contains(view.IncludeKinds, elemPtr.Kind) && !contains(view.ExcludeKinds, elemPtr.Kind) {
				result[id] = elemPtr
			}
		}
	} else {
		// Include all
		result = flatElems
	}

	// Apply excludes
	for _, kind := range view.ExcludeKinds {
		for id, elemPtr := range result {
			if elemPtr.Kind == kind {
				delete(result, id)
			}
		}
	}

	return result, nil
}

func contains(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
