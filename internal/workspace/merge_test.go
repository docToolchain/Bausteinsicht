package workspace

import (
	"errors"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

func TestPrefixElementID(t *testing.T) {
	tests := []struct {
		id       string
		prefix   string
		expected string
	}{
		{"backend", "team1", "team1_backend"},
		{"backend.api", "team1", "team1_backend.api"},
		{"system.backend.cache", "team2", "team2_system.backend.cache"},
	}

	for _, tt := range tests {
		t.Run(tt.id+"_"+tt.prefix, func(t *testing.T) {
			result := prefixElementID(tt.id, tt.prefix)
			if result != tt.expected {
				t.Errorf("prefixElementID(%q, %q) = %q, want %q", tt.id, tt.prefix, result, tt.expected)
			}
		})
	}
}

func TestRemapElementIDs(t *testing.T) {
	ids := []string{"backend", "frontend", "db.cache"}
	prefix := "team1"

	result := remapElementIDs(ids, prefix)
	expected := []string{"team1_backend", "team1_frontend", "team1_db.cache"}

	if len(result) != len(expected) {
		t.Fatalf("remapElementIDs result length %d, want %d", len(result), len(expected))
	}

	for i, r := range result {
		if r != expected[i] {
			t.Errorf("remapElementIDs[%d] = %q, want %q", i, r, expected[i])
		}
	}
}

func TestMergeModels(t *testing.T) {
	// Create model 1 (backend team)
	model1 := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"system":  {Notation: "box", Container: true},
				"service": {Notation: "component"},
			},
		},
		Model: map[string]model.Element{
			"backend": {Kind: "system", Title: "Backend System"},
			"cache":   {Kind: "service", Title: "Cache Service"},
		},
		Relationships: []model.Relationship{
			{From: "backend", To: "cache", Label: "uses"},
		},
	}

	// Create model 2 (frontend team)
	model2 := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"system": {Notation: "box", Container: true},
			},
		},
		Model: map[string]model.Element{
			"frontend": {Kind: "system", Title: "Frontend System"},
		},
		Relationships: []model.Relationship{},
	}

	loaded := []LoadedModel{
		{Ref: ModelRef{ID: "backend-team", Path: "", Prefix: "team1"}, Model: model1},
		{Ref: ModelRef{ID: "frontend-team", Path: "", Prefix: "team2"}, Model: model2},
	}

	merged, err := MergeModels(loaded)
	if err != nil {
		t.Fatalf("MergeModels failed: %v", err)
	}

	// Verify merged elements have prefixes
	if _, ok := merged.Model["team1_backend"]; !ok {
		t.Error("expected team1_backend in merged model")
	}
	if _, ok := merged.Model["team1_cache"]; !ok {
		t.Error("expected team1_cache in merged model")
	}
	if _, ok := merged.Model["team2_frontend"]; !ok {
		t.Error("expected team2_frontend in merged model")
	}

	// Verify relationships are remapped
	if len(merged.Relationships) == 0 {
		t.Fatal("expected relationships in merged model")
	}
	rel := merged.Relationships[0]
	if rel.From != "team1_backend" || rel.To != "team1_cache" {
		t.Errorf("relationship not remapped correctly: %q -> %q", rel.From, rel.To)
	}

	// Verify specifications are merged
	if _, ok := merged.Specification.Elements["system"]; !ok {
		t.Error("expected element kind 'system' in specification")
	}
}

func TestMergeModelsWithIDPrefix(t *testing.T) {
	model1 := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"component": {Notation: "box"},
			},
		},
		Model: map[string]model.Element{
			"api": {Kind: "component", Title: "API"},
		},
		Relationships: []model.Relationship{},
	}

	// Without explicit prefix, should use model ID
	loaded := []LoadedModel{
		{Ref: ModelRef{ID: "mymodel", Path: ""}, Model: model1},
	}

	merged, err := MergeModels(loaded)
	if err != nil {
		t.Fatalf("MergeModels failed: %v", err)
	}

	if _, ok := merged.Model["mymodel_api"]; !ok {
		t.Error("expected mymodel_api in merged model when no explicit prefix")
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		slice    []string
		item     string
		expected bool
	}{
		{[]string{"a", "b", "c"}, "b", true},
		{[]string{"a", "b", "c"}, "d", false},
		{[]string{}, "a", false},
	}

	for _, tt := range tests {
		t.Run(tt.item, func(t *testing.T) {
			result := contains(tt.slice, tt.item)
			if result != tt.expected {
				t.Errorf("contains(%v, %q) = %v, want %v", tt.slice, tt.item, result, tt.expected)
			}
		})
	}
}

func specModel(kinds map[string]model.ElementKind, rels map[string]model.RelationshipKind) *model.BausteinsichtModel {
	return &model.BausteinsichtModel{
		Specification: model.Specification{Elements: kinds, Relationships: rels},
		Model:         map[string]model.Element{"x": {Kind: "system", Title: "X"}},
	}
}

func TestMergeModels_SpecificationConflicts(t *testing.T) {
	a := LoadedModel{Ref: ModelRef{ID: "a"}, Model: specModel(
		map[string]model.ElementKind{"system": {Notation: "System", Container: true, Description: "A"}, "db": {Notation: "DB"}, "ui": {Notation: "UI", C4: "Container"}},
		map[string]model.RelationshipKind{"uses": {Notation: "uses"}, "async": {Notation: "async", Dashed: true}})}

	t.Run("equal definitions merge, the first description wins", func(t *testing.T) {
		b := LoadedModel{Ref: ModelRef{ID: "b"}, Model: specModel(
			map[string]model.ElementKind{"system": {Notation: "System", Container: true, Description: "B"}, "cache": {Notation: "Cache"}},
			map[string]model.RelationshipKind{"uses": {Notation: "uses"}})}
		merged, err := MergeModels([]LoadedModel{a, b})
		if err != nil {
			t.Fatal(err)
		}
		if got := merged.Specification.Elements["system"].Description; got != "A" {
			t.Errorf("description = %q, want the first model's", got)
		}
		if _, ok := merged.Specification.Elements["cache"]; !ok {
			t.Error("kinds that only the second model defines must be added")
		}
	})

	t.Run("every conflict is reported, in a stable order", func(t *testing.T) {
		b := LoadedModel{Ref: ModelRef{ID: "b"}, Model: specModel(
			map[string]model.ElementKind{
				"system": {Notation: "Software System", Container: true},
				"db":     {Notation: "DB", Container: true},
				"ui":     {Notation: "UI", C4: "Component"},
			},
			map[string]model.RelationshipKind{"uses": {Notation: "uses", Dashed: true}, "async": {Notation: "async", Dashed: true}})}
		for i := 0; i < 5; i++ { // map iteration must not change the message
			_, err := MergeModels([]LoadedModel{a, b})
			if err == nil {
				t.Fatal("conflicting kinds must fail")
			}
			msg := err.Error()
			for _, want := range []string{
				`element kind "db" is defined differently`, `element kind "system" is defined differently`, `element kind "ui" is defined differently`,
				`relationship kind "uses" is defined differently`, `model "a"`, `model "b"`,
			} {
				if !strings.Contains(msg, want) {
					t.Fatalf("message lacks %q:\n%s", want, msg)
				}
			}
			if strings.Contains(msg, `relationship kind "async"`) {
				t.Errorf("equal relationship kind reported:\n%s", msg)
			}
			if strings.Index(msg, `"db"`) > strings.Index(msg, `"system"`) || strings.Index(msg, `"system"`) > strings.Index(msg, `"ui"`) {
				t.Errorf("conflicts are not sorted by kind:\n%s", msg)
			}
		}
	})

	t.Run("ValidateWorkspace reports the conflict", func(t *testing.T) {
		b := LoadedModel{Ref: ModelRef{ID: "b"}, Model: specModel(map[string]model.ElementKind{"system": {Notation: "Other"}}, nil)}
		problems := ValidateWorkspace(&Config{}, []LoadedModel{a, b})
		if len(problems) == 0 || !strings.Contains(errors.Join(problems...).Error(), "defined differently") {
			t.Errorf("problems = %v", problems)
		}
	})
}

// Merging must not change the loaded models: the dynamic-view steps of a model
// were remapped in place, so a second merge (validate and merge both do one)
// prefixed the endpoints twice (a_x became a_a_x).
func TestMergeModels_DoesNotModifyLoadedModels(t *testing.T) {
	m := specModel(map[string]model.ElementKind{"system": {Notation: "System"}}, nil)
	m.Model = map[string]model.Element{"x": {Kind: "system", Title: "X"}, "y": {Kind: "system", Title: "Y"}}
	m.DynamicViews = []model.DynamicView{{Key: "flow", Title: "Flow", Steps: []model.SequenceStep{{Index: 1, From: "x", To: "y", Label: "go"}}}}
	loaded := []LoadedModel{{Ref: ModelRef{ID: "a"}, Model: m}}

	first, err := MergeModels(loaded)
	if err != nil {
		t.Fatal(err)
	}
	second, err := MergeModels(loaded)
	if err != nil {
		t.Fatal(err)
	}
	if got := m.DynamicViews[0].Steps[0]; got.From != "x" || got.To != "y" {
		t.Errorf("the loaded model's step was changed to %q -> %q", got.From, got.To)
	}
	for name, merged := range map[string]*model.BausteinsichtModel{"first": first, "second": second} {
		if got := merged.DynamicViews[0].Steps[0]; got.From != "a_x" || got.To != "a_y" {
			t.Errorf("%s merge: step is %q -> %q, want a_x -> a_y", name, got.From, got.To)
		}
	}
}

func TestMergeModels_ConflictLeavesLoadedModelsUntouched(t *testing.T) {
	a := specModel(map[string]model.ElementKind{"system": {Notation: "System"}}, nil)
	b := specModel(map[string]model.ElementKind{"system": {Notation: "Other"}}, nil)
	b.DynamicViews = []model.DynamicView{{Key: "flow", Steps: []model.SequenceStep{{Index: 1, From: "x", To: "x"}}}}
	if _, err := MergeModels([]LoadedModel{{Ref: ModelRef{ID: "a"}, Model: a}, {Ref: ModelRef{ID: "b"}, Model: b}}); err == nil {
		t.Fatal("the conflict must fail")
	}
	if got := b.DynamicViews[0].Steps[0]; got.From != "x" || got.To != "x" {
		t.Errorf("a failed merge changed the loaded model: %q -> %q", got.From, got.To)
	}
}
