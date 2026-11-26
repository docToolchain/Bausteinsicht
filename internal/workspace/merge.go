package workspace

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// resolvePrefix returns the effective prefix for a loaded model:
// ModelRef.Prefix if set, otherwise ModelRef.ID.
func resolvePrefix(lm LoadedModel) string {
	if lm.Ref.Prefix != "" {
		return lm.Ref.Prefix
	}
	return lm.Ref.ID
}

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

	// Merge specifications and elements in a single pass per model. A kind that
	// two models define differently is a conflict (ADR-012), not a silent win
	// for the first model.
	specs := newSpecMerge()
	for _, lm := range loaded {
		specs.add(lm, merged)
		prefix := resolvePrefix(lm)
		// Only the top-level elements are copied: their children come along.
		// (Copying the flattened elements as well added every nested element a
		// second time, as a top-level entry with a dotted key.)
		for id, elem := range lm.Model.Model {
			prefixed := prefixElementID(id, prefix)
			if _, taken := merged.Model[prefixed]; taken {
				return nil, collisionError("element", prefixed)
			}
			merged.Model[prefixed] = elem
		}
	}
	// Report conflicting kinds before anything else is remapped.
	if err := specs.err(); err != nil {
		return nil, err
	}

	// Merge relationships, views, dynamic views, and constraints in one pass.
	for _, lm := range loaded {
		prefix := resolvePrefix(lm)

		for _, rel := range lm.Model.Relationships {
			remappedRel := rel
			remappedRel.From = prefixElementID(rel.From, prefix)
			remappedRel.To = prefixElementID(rel.To, prefix)
			merged.Relationships = append(merged.Relationships, remappedRel)
		}

		for viewID, view := range lm.Model.Views {
			remappedView := view
			remappedView.Include = remapElementIDs(view.Include, prefix)
			remappedView.Exclude = remapElementIDs(view.Exclude, prefix)
			if view.Scope != "" {
				remappedView.Scope = prefixElementID(view.Scope, prefix)
			}
			if _, taken := merged.Views[prefix+"_"+viewID]; taken {
				return nil, collisionError("view", prefix+"_"+viewID)
			}
			merged.Views[prefix+"_"+viewID] = remappedView
		}

		for _, dv := range lm.Model.DynamicViews {
			remappedDV := dv
			// dv shares its Steps with the loaded model; remap a copy.
			remappedDV.Steps = append([]model.SequenceStep(nil), dv.Steps...)
			remappedDV.Key = prefix + "_" + dv.Key
			for i := range remappedDV.Steps {
				remappedDV.Steps[i].From = prefixElementID(remappedDV.Steps[i].From, prefix)
				remappedDV.Steps[i].To = prefixElementID(remappedDV.Steps[i].To, prefix)
			}
			merged.DynamicViews = append(merged.DynamicViews, remappedDV)
		}

		for _, c := range lm.Model.Constraints {
			// Selectors by element ID follow the prefix of the elements; by kind
			// or tag they apply to the merged model as they are.
			if c.From != "" {
				c.From = prefixElementID(c.From, prefix)
			}
			if c.To != "" {
				c.To = prefixElementID(c.To, prefix)
			}
			merged.Constraints = append(merged.Constraints, c)
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

// collisionError reports that two models produce the same merged ID. The prefix
// and the ID are joined with "_", so different pairs can give the same result,
// for example ("a", "b_c") and ("a_b", "c"); one would silently replace the other.
func collisionError(what, id string) error {
	return fmt.Errorf("merged %s ID %q is produced by more than one model; use different prefixes", what, id)
}

// specMerge unions the element and relationship kinds of the models and records
// the kinds that two models define differently.
type specMerge struct {
	elementOwner      map[string]string // kind -> ID of the model that defined it first
	relationshipOwner map[string]string
	conflicts         []specConflict
}

// specConflict is one conflicting kind; key orders the report (element kinds
// before relationship kinds, then by kind name).
type specConflict struct {
	key string
	err error
}

func newSpecMerge() *specMerge {
	return &specMerge{elementOwner: map[string]string{}, relationshipOwner: map[string]string{}}
}

// add merges the kinds of one model into merged. Two definitions of a kind are
// the same if they agree on everything that affects rendering and validation
// (notation, container, C4 macro / dashed); a different description is not a
// conflict, the first one is kept.
func (m *specMerge) add(lm LoadedModel, merged *model.BausteinsichtModel) {
	for _, kind := range sortedKeys(lm.Model.Specification.Elements) {
		def := lm.Model.Specification.Elements[kind]
		have, exists := merged.Specification.Elements[kind]
		switch {
		case !exists:
			merged.Specification.Elements[kind] = def
			m.elementOwner[kind] = lm.Ref.ID
		case have.Notation != def.Notation || have.Container != def.Container || have.C4 != def.C4:
			m.conflicts = append(m.conflicts, specConflict{"1 " + kind, fmt.Errorf(
				"element kind %q is defined differently: model %q has notation %q, container %t, c4 %q; model %q has notation %q, container %t, c4 %q",
				kind, m.elementOwner[kind], have.Notation, have.Container, have.C4, lm.Ref.ID, def.Notation, def.Container, def.C4)})
		}
	}
	for _, kind := range sortedKeys(lm.Model.Specification.Relationships) {
		def := lm.Model.Specification.Relationships[kind]
		have, exists := merged.Specification.Relationships[kind]
		switch {
		case !exists:
			merged.Specification.Relationships[kind] = def
			m.relationshipOwner[kind] = lm.Ref.ID
		case have.Notation != def.Notation || have.Dashed != def.Dashed:
			m.conflicts = append(m.conflicts, specConflict{"2 " + kind, fmt.Errorf(
				"relationship kind %q is defined differently: model %q has notation %q, dashed %t; model %q has notation %q, dashed %t",
				kind, m.relationshipOwner[kind], have.Notation, have.Dashed, lm.Ref.ID, def.Notation, def.Dashed)})
		}
	}
}

// err returns all conflicts, sorted by kind (stable for equal kinds, so a kind
// that two later models both change keeps the model order).
func (m *specMerge) err() error {
	sort.SliceStable(m.conflicts, func(i, j int) bool { return m.conflicts[i].key < m.conflicts[j].key })
	errs := make([]error, len(m.conflicts))
	for i, c := range m.conflicts {
		errs[i] = c.err
	}
	return errors.Join(errs...)
}

func sortedKeys[V any](in map[string]V) []string {
	keys := make([]string, 0, len(in))
	for k := range in {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
