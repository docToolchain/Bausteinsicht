package workspace

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// MergeWorkspace merges the loaded models like MergeModels and then adds the
// cross-model relationships of the workspace configuration. Each endpoint has
// the form "<model ID>:<element ID>" (for example "team_a:shop.api") and is
// translated to the element's ID in the merged model; an endpoint that does not
// resolve is an error.
func MergeWorkspace(cfg *Config, loaded []LoadedModel) (*model.BausteinsichtModel, error) {
	merged, err := MergeModels(loaded)
	if err != nil {
		return nil, err
	}
	flat, _ := model.FlattenElements(merged)

	var errs []error
	for _, cr := range cfg.CrossRels {
		from, fromErr := resolveEndpoint(cr.From, loaded, flat)
		to, toErr := resolveEndpoint(cr.To, loaded, flat)
		if fromErr != nil || toErr != nil {
			errs = append(errs, fmt.Errorf("cross-model relationship %q: %w", cr.ID, errors.Join(fromErr, toErr)))
			continue
		}
		merged.Relationships = append(merged.Relationships, model.Relationship{
			From: from, To: to, Label: cr.Label, Kind: cr.Kind, Description: cr.Description,
		})
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return merged, nil
}

// resolveEndpoint translates "<model ID>:<element ID>" to the element's ID in
// the merged model and checks that the element exists (unless mergedFlat is
// nil, in which case only the model is checked).
func resolveEndpoint(endpoint string, loaded []LoadedModel, mergedFlat map[string]*model.Element) (string, error) {
	modelID, elemID, ok := strings.Cut(endpoint, ":")
	if !ok || modelID == "" || elemID == "" {
		return "", fmt.Errorf("endpoint %q must have the form <model>:<element>", endpoint)
	}
	for _, lm := range loaded {
		if lm.Ref.ID != modelID {
			continue
		}
		merged := prefixElementID(elemID, resolvePrefix(lm))
		if mergedFlat != nil && mergedFlat[merged] == nil {
			return "", fmt.Errorf("element %q not found in model %q", elemID, modelID)
		}
		return merged, nil
	}
	return "", fmt.Errorf("endpoint %q: unknown model %q", endpoint, modelID)
}

// ApplySelection returns the part of m that ref selects: the listed top-level
// elements (default all of them), each cut off at ref.Depth. Relationships whose
// ends are both still there are kept; views, dynamic views and constraints are
// not imported, because they refer to elements that may be gone. m itself is
// not changed. Without Elements and Depth, m is returned as it is.
func ApplySelection(m *model.BausteinsichtModel, ref ModelRef) (*model.BausteinsichtModel, error) {
	if len(ref.Elements) == 0 && ref.Depth == 0 {
		return m, nil
	}
	if ref.Depth < 0 {
		return nil, fmt.Errorf("depth must not be negative, got %d", ref.Depth)
	}

	ids := ref.Elements
	if len(ids) == 0 {
		for id := range m.Model {
			ids = append(ids, id)
		}
		sort.Strings(ids)
	}

	selected := &model.BausteinsichtModel{
		Schema:        m.Schema,
		Config:        m.Config,
		Meta:          m.Meta,
		Specification: m.Specification,
		Model:         make(map[string]model.Element, len(ids)),
	}
	for _, id := range ids {
		if strings.Contains(id, ".") {
			return nil, fmt.Errorf("element %q: only top-level elements can be selected", id)
		}
		elem, ok := m.Model[id]
		if !ok {
			return nil, fmt.Errorf("element %q not found", id)
		}
		selected.Model[id] = cutOff(elem, ref.Depth)
	}

	flat, _ := model.FlattenElements(selected)
	for _, r := range m.Relationships {
		if flat[r.From] != nil && flat[r.To] != nil {
			selected.Relationships = append(selected.Relationships, r)
		}
	}
	return selected, nil
}

// cutOff returns a copy of e that keeps depth levels (itself included); depth 0
// keeps everything.
func cutOff(e model.Element, depth int) model.Element {
	if depth == 0 || len(e.Children) == 0 {
		return e
	}
	if depth == 1 {
		e.Children = nil
		return e
	}
	children := make(map[string]model.Element, len(e.Children))
	for id, child := range e.Children {
		children[id] = cutOff(child, depth-1)
	}
	e.Children = children
	return e
}

// ValidateWorkspace checks what the individual models cannot: that model IDs
// and prefixes are unique, that cross-model relationships point at elements
// that exist, that their IDs are unique, and that workspace views refer to
// known models. It returns every problem it finds.
func ValidateWorkspace(cfg *Config, loaded []LoadedModel) []error {
	var problems []error

	seenID := make(map[string]bool)
	prefixUsers := make(map[string]int)
	for _, lm := range loaded {
		if seenID[lm.Ref.ID] {
			problems = append(problems, fmt.Errorf("duplicate model ID %q", lm.Ref.ID))
		}
		seenID[lm.Ref.ID] = true
		prefixUsers[resolvePrefix(lm)]++
	}
	prefixes := make([]string, 0, len(prefixUsers))
	for p, n := range prefixUsers {
		if n > 1 {
			prefixes = append(prefixes, p)
		}
	}
	sort.Strings(prefixes)
	for _, p := range prefixes {
		problems = append(problems, fmt.Errorf("prefix %q is used by more than one model", p))
	}

	// If the models cannot be merged, report that and check only the model part
	// of the cross-model endpoints; looking elements up in a merged model that
	// does not exist would report every endpoint as missing.
	var mergedFlat map[string]*model.Element
	if len(loaded) > 0 {
		if merged, err := MergeModels(loaded); err != nil {
			problems = append(problems, err)
		} else {
			mergedFlat, _ = model.FlattenElements(merged)
		}
	}
	problems = append(problems, duplicateConstraintIDs(loaded)...)
	seenRel := make(map[string]bool)
	for _, cr := range cfg.CrossRels {
		if seenRel[cr.ID] {
			problems = append(problems, fmt.Errorf("duplicate cross-model relationship ID %q", cr.ID))
		}
		seenRel[cr.ID] = true
		for _, endpoint := range []string{cr.From, cr.To} {
			if _, err := resolveEndpoint(endpoint, loaded, mergedFlat); err != nil {
				problems = append(problems, fmt.Errorf("cross-model relationship %q: %w", cr.ID, err))
			}
		}
	}

	viewKeys := make([]string, 0, len(cfg.Views))
	for key := range cfg.Views {
		viewKeys = append(viewKeys, key)
	}
	sort.Strings(viewKeys)
	for _, key := range viewKeys {
		for _, modelID := range cfg.Views[key].IncludeFrom {
			if !seenID[modelID] {
				problems = append(problems, fmt.Errorf("view %q: include-from refers to unknown model %q", key, modelID))
			}
		}
	}
	return problems
}

// duplicateConstraintIDs reports constraint IDs that more than one model
// defines: after the merge, lint results could not be told apart.
func duplicateConstraintIDs(loaded []LoadedModel) []error {
	owner := make(map[string]string)
	reported := make(map[string]bool)
	var problems []error
	for _, lm := range loaded {
		for _, c := range lm.Model.Constraints {
			if first, ok := owner[c.ID]; ok && first != lm.Ref.ID && !reported[c.ID] {
				reported[c.ID] = true
				problems = append(problems, fmt.Errorf("constraint ID %q is defined by more than one model (%s, %s)", c.ID, first, lm.Ref.ID))
			}
			if _, ok := owner[c.ID]; !ok {
				owner[c.ID] = lm.Ref.ID
			}
		}
	}
	return problems
}
