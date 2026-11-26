package sync

import (
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// danglingRefsModel has a cache that the user deletes in draw.io, and the other
// places of a model that refer to elements by ID.
func danglingRefsModel() *model.BausteinsichtModel {
	return &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"actor":     {Notation: "Actor"},
				"container": {Notation: "Container", Container: true},
			},
			Relationships: map[string]model.RelationshipKind{"uses": {Notation: "uses"}},
		},
		Model: map[string]model.Element{
			"user": {Kind: "actor", Title: "User"},
			"api":  {Kind: "container", Title: "API"},
			"cache": {Kind: "container", Title: "Cache", Children: map[string]model.Element{
				"store": {Kind: "container", Title: "Store"},
			}},
		},
		Relationships: []model.Relationship{
			{From: "user", To: "api", Kind: "uses", Label: "calls"},
			{From: "api", To: "cache", Kind: "uses", Label: "reads"},
		},
		Views: map[string]model.View{
			"all":  {Title: "All", Include: []string{"user", "api", "cache", "cache.store", "api.*"}, Exclude: []string{"cache.store"}},
			"wild": {Title: "Wild", Include: []string{"*"}},
		},
		DynamicViews: []model.DynamicView{
			{Key: "mixed", Title: "Mixed", Steps: []model.SequenceStep{
				{Index: 1, From: "user", To: "api", Label: "go"},
				{Index: 2, From: "api", To: "cache", Label: "read"},
				{Index: 3, From: "cache", To: "api", Label: "data", Type: "return"},
			}},
			{Key: "cache_only", Title: "Cache only", Steps: []model.SequenceStep{
				{Index: 1, From: "api", To: "cache.store", Label: "put"},
			}},
			{Key: "blocks", Title: "Blocks", Steps: []model.SequenceStep{
				{Index: 1, From: "user", To: "api", Label: "go"},
				{Index: 2, Type: model.StepAlt, Label: "cached"},
				{Index: 3, From: "api", To: "cache", Label: "get"},
				{Index: 4, Type: model.StepEnd},
			}},
		},
	}
}

func validationErrors(m *model.BausteinsichtModel) []string {
	var out []string
	for _, e := range model.Validate(m) {
		out = append(out, e.Error())
	}
	return out
}

// TestApplyReverse_DeletedElementLeavesAValidModel (#720): deleting an element
// in draw.io removed it and its relationships, but not the dynamic-view steps
// that referred to it, so the model no longer validated and every command that
// validates (validate, export, the next sync) failed.
func TestApplyReverse_DeletedElementLeavesAValidModel(t *testing.T) {
	m := danglingRefsModel()
	if errs := validationErrors(m); len(errs) > 0 {
		t.Fatalf("setup: the model is invalid: %v", errs)
	}

	r := ApplyReverse(elemChangeSet("cache", Deleted, "", "", ""), m, nil)

	if _, ok := m.Model["cache"]; ok || r.ElementsDeleted != 1 {
		t.Fatalf("the element must be removed (deleted=%d)", r.ElementsDeleted)
	}
	if errs := validationErrors(m); len(errs) > 0 {
		t.Fatalf("the model is invalid after the deletion:\n%s", strings.Join(errs, "\n"))
	}
}

func TestApplyReverse_DeletedElementDropsItsDynamicViewSteps(t *testing.T) {
	m := danglingRefsModel()

	r := ApplyReverse(elemChangeSet("cache", Deleted, "", "", ""), m, nil)

	byKey := map[string]model.DynamicView{}
	for _, dv := range m.DynamicViews {
		byKey[dv.Key] = dv
	}
	if got := byKey["mixed"].Steps; len(got) != 1 || got[0].Label != "go" {
		t.Errorf("only the step that does not involve the cache may remain: %+v", got)
	}
	if _, ok := byKey["cache_only"]; ok {
		t.Error("a dynamic view that only involved the cache and its child must be removed")
	}
	// Block steps stay, so the alt/end of the third view still balance.
	if got := byKey["blocks"].Steps; len(got) != 3 || got[1].Type != model.StepAlt || got[2].Type != model.StepEnd {
		t.Errorf("block steps must be kept: %+v", got)
	}
	if r.RelationshipsDeleted != 1 {
		t.Errorf("RelationshipsDeleted = %d, want 1", r.RelationshipsDeleted)
	}
	if got := strings.Join(r.Warnings, "\n"); !strings.Contains(got, "dynamicViews[") || !strings.Contains(got, `"cache"`) {
		t.Errorf("the warning must name the references that were dropped:\n%s", got)
	}
}

func TestApplyReverse_DeletedElementCleansViewListsButKeepsWildcards(t *testing.T) {
	m := danglingRefsModel()

	ApplyReverse(elemChangeSet("cache", Deleted, "", "", ""), m, nil)

	all := m.Views["all"]
	for _, id := range append(append([]string{}, all.Include...), all.Exclude...) {
		if id == "cache" || strings.HasPrefix(id, "cache.") {
			t.Errorf("the view still names the deleted element: %v / %v", all.Include, all.Exclude)
		}
	}
	if len(all.Include) != 3 || all.Include[2] != "api.*" {
		t.Errorf("other entries, wildcards included, must stay: %v", all.Include)
	}
	if got := m.Views["wild"].Include; len(got) != 1 || got[0] != "*" {
		t.Errorf("a wildcard view must be unchanged: %v", got)
	}
}

// An element that a view uses as its scope, or a constraint names, cannot be
// removed without breaking the model; like "remove element" refuses, reverse
// sync keeps it (the model wins) and says why.
func TestApplyReverse_DeletedElementThatIsAScopeOrConstrainedIsKept(t *testing.T) {
	tests := map[string]func(*model.BausteinsichtModel){
		"scope of a view": func(m *model.BausteinsichtModel) {
			m.Views["inside"] = model.View{Title: "Inside", Scope: "cache", Include: []string{"cache.*"}}
		},
		"constraint": func(m *model.BausteinsichtModel) {
			m.Constraints = []model.Constraint{{ID: "c1", Rule: "no-relationship", From: "cache", To: "user"}}
		},
	}
	for name, setup := range tests {
		t.Run(name, func(t *testing.T) {
			m := danglingRefsModel()
			setup(m)

			r := ApplyReverse(elemChangeSet("cache", Deleted, "", "", ""), m, nil)

			if _, ok := m.Model["cache"]; !ok || r.ElementsDeleted != 0 {
				t.Fatalf("the element must stay in the model (deleted=%d)", r.ElementsDeleted)
			}
			if len(m.Relationships) != 2 || len(m.DynamicViews) != 3 {
				t.Errorf("nothing else may be removed: %d relationships, %d dynamic views", len(m.Relationships), len(m.DynamicViews))
			}
			if errs := validationErrors(m); len(errs) > 0 {
				t.Errorf("the model must stay valid: %v", errs)
			}
			if got := strings.Join(r.Warnings, "\n"); !strings.Contains(got, `"cache"`) || !strings.Contains(strings.ToLower(got), "kept") {
				t.Errorf("the warning must say that the element was kept and why:\n%s", got)
			}
		})
	}
}
