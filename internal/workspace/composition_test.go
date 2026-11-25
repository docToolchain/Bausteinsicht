package workspace

import (
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

func teamModel(kind string, titles ...string) *model.BausteinsichtModel {
	m := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements:      map[string]model.ElementKind{"system": {Notation: "System", Container: true}, "container": {Notation: "Container", Container: true}},
			Relationships: map[string]model.RelationshipKind{"uses": {Notation: "uses"}},
		},
		Model: map[string]model.Element{},
		Views: map[string]model.View{"overview": {Title: "Overview", Include: []string{"*"}}},
	}
	for _, id := range titles {
		m.Model[id] = model.Element{Kind: kind, Title: id, Children: map[string]model.Element{
			"api": {Kind: "container", Title: id + " API", Children: map[string]model.Element{
				"handler": {Kind: "container", Title: "Handler"},
			}},
			"db": {Kind: "container", Title: id + " DB"},
		}}
	}
	m.Relationships = []model.Relationship{{From: titles[0] + ".api", To: titles[0] + ".db", Label: "reads", Kind: "uses"}}
	return m
}

func twoTeams() (*Config, []LoadedModel) {
	cfg := &Config{
		Workspace: WorkspaceMetadata{Name: "landscape"},
		Models:    []ModelRef{{ID: "team_a", Path: "a.jsonc"}, {ID: "team_b", Path: "b.jsonc", Prefix: "b"}},
	}
	return cfg, []LoadedModel{
		{Ref: cfg.Models[0], Model: teamModel("system", "shop")},
		{Ref: cfg.Models[1], Model: teamModel("system", "billing")},
	}
}

// Nested elements are merged once, below their parent. They used to be added a
// second time as top-level entries with a dotted key ("team_a_shop.api").
func TestMergeModels_NestedElementsAreNotDuplicated(t *testing.T) {
	_, loaded := twoTeams()
	merged, err := MergeModels(loaded)
	if err != nil {
		t.Fatal(err)
	}
	for id := range merged.Model {
		if strings.Contains(id, ".") {
			t.Errorf("top-level key %q contains a dot: nested elements were added twice", id)
		}
	}
	flat, _ := model.FlattenElements(merged)
	for _, want := range []string{"team_a_shop", "team_a_shop.api", "team_a_shop.api.handler", "b_billing.db"} {
		if flat[want] == nil {
			t.Errorf("merged model lacks %q", want)
		}
	}
	if errs := model.Validate(merged); len(errs) != 0 {
		t.Errorf("merged model is invalid: %v", errs)
	}
}

// crossModelRelationships were read from the configuration but never merged.
func TestMergeWorkspace_AppliesCrossModelRelationships(t *testing.T) {
	cfg, loaded := twoTeams()
	cfg.CrossRels = []CrossModelRelationship{
		{ID: "pay", From: "team_a:shop.api", To: "team_b:billing", Label: "charges via", Kind: "uses", Description: "REST"},
	}
	merged, err := MergeWorkspace(cfg, loaded)
	if err != nil {
		t.Fatal(err)
	}
	var found *model.Relationship
	for i, r := range merged.Relationships {
		if r.Label == "charges via" {
			found = &merged.Relationships[i]
		}
	}
	if found == nil {
		t.Fatalf("cross-model relationship not merged: %+v", merged.Relationships)
	}
	if found.From != "team_a_shop.api" || found.To != "b_billing" || found.Kind != "uses" || found.Description != "REST" {
		t.Errorf("endpoints must be translated to merged IDs (model ID, prefix is used for the ID): %+v", *found)
	}
	if errs := model.Validate(merged); len(errs) != 0 {
		t.Errorf("merged model is invalid: %v", errs)
	}
}

func TestMergeWorkspace_RejectsDanglingCrossModelRelationships(t *testing.T) {
	tests := []struct {
		name string
		rel  CrossModelRelationship
		want string
	}{
		{"unknown model", CrossModelRelationship{ID: "x", From: "nope:shop", To: "team_b:billing"}, `unknown model "nope"`},
		{"unknown element", CrossModelRelationship{ID: "x", From: "team_a:shop.cache", To: "team_b:billing"}, `element "shop.cache" not found in model "team_a"`},
		{"missing model prefix", CrossModelRelationship{ID: "x", From: "shop", To: "team_b:billing"}, `must have the form <model>:<element>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, loaded := twoTeams()
			cfg.CrossRels = []CrossModelRelationship{tt.rel}
			_, err := MergeWorkspace(cfg, loaded)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %v, want it to contain %q", err, tt.want)
			}
		})
	}
}

// A model can contribute only part of itself: selected top-level elements, cut
// off at a depth ("black box" = depth 1).
func TestApplySelection(t *testing.T) {
	t.Run("selected elements as a black box", func(t *testing.T) {
		src := teamModel("system", "shop", "billing")
		got, err := ApplySelection(src, ModelRef{ID: "t", Elements: []string{"shop"}, Depth: 1})
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Model) != 1 || len(got.Model["shop"].Children) != 0 {
			t.Errorf("want only shop without children, got %+v", got.Model)
		}
		if len(got.Relationships) != 0 {
			t.Errorf("relationships to cut-off elements must go: %+v", got.Relationships)
		}
		if len(got.Views) != 0 {
			t.Errorf("views of a partial import must not be imported: %+v", got.Views)
		}
		if len(got.Specification.Elements) != 2 {
			t.Errorf("the specification is kept whole: %+v", got.Specification)
		}
		if len(src.Model["shop"].Children) == 0 {
			t.Error("the loaded model must not be changed")
		}
	})

	t.Run("depth 2 keeps one level of children and the relationships inside", func(t *testing.T) {
		got, err := ApplySelection(teamModel("system", "shop"), ModelRef{ID: "t", Depth: 2})
		if err != nil {
			t.Fatal(err)
		}
		flat, _ := model.FlattenElements(got)
		if flat["shop.api"] == nil || flat["shop.db"] == nil || flat["shop.api.handler"] != nil {
			t.Errorf("depth 2 must keep shop and its children only: %v", keysOf(flat))
		}
		if len(got.Relationships) != 1 {
			t.Errorf("shop.api -> shop.db survives: %+v", got.Relationships)
		}
	})

	t.Run("no selection keeps everything", func(t *testing.T) {
		src := teamModel("system", "shop")
		got, err := ApplySelection(src, ModelRef{ID: "t"})
		if err != nil || got != src {
			t.Errorf("an empty selection must return the model unchanged, got %v, %v", got, err)
		}
	})

	t.Run("unknown or nested elements are errors", func(t *testing.T) {
		src := teamModel("system", "shop")
		if _, err := ApplySelection(src, ModelRef{ID: "t", Elements: []string{"nope"}}); err == nil || !strings.Contains(err.Error(), `"nope" not found`) {
			t.Errorf("unknown element: %v", err)
		}
		if _, err := ApplySelection(src, ModelRef{ID: "t", Elements: []string{"shop.api"}}); err == nil || !strings.Contains(err.Error(), "top-level") {
			t.Errorf("nested element: %v", err)
		}
		if _, err := ApplySelection(src, ModelRef{ID: "t", Depth: -1}); err == nil {
			t.Error("negative depth must be rejected")
		}
	})
}

func keysOf(m map[string]*model.Element) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func TestValidateWorkspace(t *testing.T) {
	t.Run("a consistent workspace has no problems", func(t *testing.T) {
		cfg, loaded := twoTeams()
		cfg.CrossRels = []CrossModelRelationship{{ID: "pay", From: "team_a:shop.api", To: "team_b:billing"}}
		if problems := ValidateWorkspace(cfg, loaded); len(problems) != 0 {
			t.Errorf("unexpected problems: %v", problems)
		}
	})

	t.Run("duplicate model IDs and prefixes, dangling cross-model relationships and views", func(t *testing.T) {
		cfg, loaded := twoTeams()
		cfg.Models[1].ID = "team_a"
		cfg.Models[1].Prefix = "team_a"
		loaded[1].Ref = cfg.Models[1]
		cfg.CrossRels = []CrossModelRelationship{
			{ID: "dup", From: "team_a:shop", To: "team_a:ghost"},
			{ID: "dup", From: "team_a:shop", To: "team_a:shop.api"},
		}
		cfg.Views = map[string]WorkspaceView{"all": {Title: "All", IncludeFrom: []string{"team_z"}}}

		text := strings.Join(problems(ValidateWorkspace(cfg, loaded)), "\n")
		for _, want := range []string{
			`duplicate model ID "team_a"`,
			`prefix "team_a" is used by more than one model`,
			`element "ghost" not found in model "team_a"`,
			`duplicate cross-model relationship ID "dup"`,
			`include-from refers to unknown model "team_z"`,
		} {
			if !strings.Contains(text, want) {
				t.Errorf("missing %q in:\n%s", want, text)
			}
		}
	})
}

func problems(errs []error) []string {
	out := make([]string, len(errs))
	for i, e := range errs {
		out[i] = e.Error()
	}
	return out
}

// Constraints name elements by ID, so they must follow the prefix of the
// elements. Before, they were merged unchanged and matched nothing (found in
// the review of #227: lint passed on a model that violated the rule).
func TestMergeModels_RemapsConstraintSelectors(t *testing.T) {
	cfg, loaded := twoTeams()
	_ = cfg
	loaded[0].Model.Constraints = []model.Constraint{
		{ID: "c1", Rule: "no-relationship", From: "shop.api", To: "shop.db"},
		{ID: "c2", Rule: "no-relationship", FromKind: "container", ToKind: "container"},
	}
	merged, err := MergeModels(loaded)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged.Constraints) != 2 {
		t.Fatalf("want 2 constraints, got %+v", merged.Constraints)
	}
	if c := merged.Constraints[0]; c.From != "team_a_shop.api" || c.To != "team_a_shop.db" {
		t.Errorf("selectors by ID must get the prefix: %+v", c)
	}
	if c := merged.Constraints[1]; c.From != "" || c.To != "" || c.FromKind != "container" {
		t.Errorf("selectors by kind or tag are left alone: %+v", c)
	}
	if loaded[0].Model.Constraints[0].From != "shop.api" {
		t.Error("the loaded model must not be changed")
	}
}

// Two different (prefix, element) pairs can give the same merged ID, because
// the prefix and the element are joined with "_": ("a", "b_c") and ("a_b", "c").
// That used to overwrite one element silently.
func TestMergeModels_RejectsCollidingMergedIDs(t *testing.T) {
	a := &model.BausteinsichtModel{Model: map[string]model.Element{"b_c": {Kind: "system", Title: "one"}}}
	b := &model.BausteinsichtModel{Model: map[string]model.Element{"c": {Kind: "system", Title: "two"}}}
	loaded := []LoadedModel{
		{Ref: ModelRef{ID: "m1", Prefix: "a"}, Model: a},
		{Ref: ModelRef{ID: "m2", Prefix: "a_b"}, Model: b},
	}
	_, err := MergeModels(loaded)
	if err == nil || !strings.Contains(err.Error(), `"a_b_c"`) {
		t.Errorf("err = %v, want a collision on a_b_c", err)
	}

	cfg := &Config{Models: []ModelRef{loaded[0].Ref, loaded[1].Ref}}
	text := strings.Join(problems(ValidateWorkspace(cfg, loaded)), "\n")
	if !strings.Contains(text, `"a_b_c"`) {
		t.Errorf("validate must report the collision, got:\n%s", text)
	}
}

func TestValidateWorkspace_ReportsDuplicateConstraintIDs(t *testing.T) {
	cfg, loaded := twoTeams()
	loaded[0].Model.Constraints = []model.Constraint{{ID: "no-cycles", Rule: "no-relationship", From: "shop", To: "shop"}}
	loaded[1].Model.Constraints = []model.Constraint{{ID: "no-cycles", Rule: "no-relationship", From: "billing", To: "billing"}}
	text := strings.Join(problems(ValidateWorkspace(cfg, loaded)), "\n")
	if !strings.Contains(text, `constraint ID "no-cycles" is defined by more than one model`) {
		t.Errorf("missing duplicate constraint ID problem:\n%s", text)
	}
}
