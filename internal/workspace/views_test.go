package workspace

import (
	"reflect"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

func viewTestModels() []LoadedModel {
	spec := model.Specification{
		Elements: map[string]model.ElementKind{
			"system": {Notation: "System", Container: true},
			"db":     {Notation: "DB"},
			"actor":  {Notation: "Actor"},
		},
		Relationships: map[string]model.RelationshipKind{"uses": {Notation: "uses"}},
	}
	a := &model.BausteinsichtModel{
		Specification: spec,
		Model: map[string]model.Element{
			"shop": {Kind: "system", Title: "Shop", Children: map[string]model.Element{
				"db": {Kind: "db", Title: "Shop DB"},
			}},
			"customer": {Kind: "actor", Title: "Customer"},
		},
		Views: map[string]model.View{"ctx": {Title: "Context", Include: []string{"*"}}},
	}
	b := &model.BausteinsichtModel{
		Specification: spec,
		Model: map[string]model.Element{
			"billing": {Kind: "system", Title: "Billing", Children: map[string]model.Element{
				"ledger": {Kind: "db", Title: "Ledger"},
			}},
		},
	}
	return []LoadedModel{
		{Ref: ModelRef{ID: "team_a", Prefix: "a"}, Model: a},
		{Ref: ModelRef{ID: "team_b"}, Model: b},
	}
}

func mergeWithViews(t *testing.T, views map[string]WorkspaceView) (*model.BausteinsichtModel, error) {
	t.Helper()
	return MergeWorkspace(&Config{Views: views}, viewTestModels())
}

func TestWorkspaceViews_Selection(t *testing.T) {
	tests := []struct {
		name string
		view WorkspaceView
		want []string
	}{
		{"every model, every element, nested ones too", WorkspaceView{Title: "All"},
			[]string{"a_customer", "a_shop", "a_shop.db", "team_b_billing", "team_b_billing.ledger"}},
		{"one model", WorkspaceView{Title: "A", IncludeFrom: []string{"team_a"}},
			[]string{"a_customer", "a_shop", "a_shop.db"}},
		{"models in the given set, uses the prefix of each", WorkspaceView{Title: "B", IncludeFrom: []string{"team_b"}},
			[]string{"team_b_billing", "team_b_billing.ledger"}},
		{"only some kinds", WorkspaceView{Title: "DBs", IncludeKinds: []string{"db"}},
			[]string{"a_shop.db", "team_b_billing.ledger"}},
		{"kinds of one model", WorkspaceView{Title: "A systems", IncludeFrom: []string{"team_a"}, IncludeKinds: []string{"system", "actor"}},
			[]string{"a_customer", "a_shop"}},
		{"everything but a kind", WorkspaceView{Title: "No DB", ExcludeKinds: []string{"db"}},
			[]string{"a_customer", "a_shop", "team_b_billing"}},
		{"include and exclude together", WorkspaceView{Title: "x", IncludeKinds: []string{"db", "system"}, ExcludeKinds: []string{"db"}},
			[]string{"a_shop", "team_b_billing"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merged, err := mergeWithViews(t, map[string]WorkspaceView{"landscape": tt.view})
			if err != nil {
				t.Fatal(err)
			}
			v, ok := merged.Views["landscape"]
			if !ok {
				t.Fatalf("views = %v", merged.Views)
			}
			if !reflect.DeepEqual(v.Include, tt.want) {
				t.Errorf("include = %v, want %v", v.Include, tt.want)
			}
			if v.Title != tt.view.Title {
				t.Errorf("title = %q", v.Title)
			}
			if errs := model.Validate(merged); len(errs) > 0 {
				t.Errorf("the merged model is invalid: %v", errs)
			}
		})
	}
}

func TestWorkspaceViews_CopiesTitleDescriptionLayoutAndKeepsModelViews(t *testing.T) {
	merged, err := mergeWithViews(t, map[string]WorkspaceView{
		"landscape": {Title: "Landscape", Description: "all teams", Layout: "grid"},
	})
	if err != nil {
		t.Fatal(err)
	}
	v := merged.Views["landscape"]
	if v.Description != "all teams" || v.Layout != "grid" {
		t.Errorf("view = %+v", v)
	}
	if _, ok := merged.Views["a_ctx"]; !ok {
		t.Errorf("the prefixed view of team a is gone: %v", merged.Views)
	}
}

func TestWorkspaceViews_PartialImportContributesOnlyWhatItImports(t *testing.T) {
	loaded := viewTestModels()
	loaded[0].Ref.Elements = []string{"shop"}
	loaded[0].Ref.Depth = 1
	sel, err := ApplySelection(loaded[0].Model, loaded[0].Ref)
	if err != nil {
		t.Fatal(err)
	}
	loaded[0].Model = sel
	merged, err := MergeWorkspace(&Config{Views: map[string]WorkspaceView{"a": {Title: "A", IncludeFrom: []string{"team_a"}}}}, loaded)
	if err != nil {
		t.Fatal(err)
	}
	if got := merged.Views["a"].Include; !reflect.DeepEqual(got, []string{"a_shop"}) {
		t.Errorf("include = %v, want only the black box a_shop", got)
	}
}

func TestWorkspaceViews_Errors(t *testing.T) {
	tests := []struct {
		name    string
		views   map[string]WorkspaceView
		wantErr string
	}{
		{"unknown model", map[string]WorkspaceView{"v": {Title: "v", IncludeFrom: []string{"team_z"}}}, `include-from refers to unknown model "team_z"`},
		{"unknown include kind", map[string]WorkspaceView{"v": {Title: "v", IncludeKinds: []string{"queue"}}}, `unknown element kind "queue"`},
		{"unknown exclude kind", map[string]WorkspaceView{"v": {Title: "v", ExcludeKinds: []string{"queue"}}}, `unknown element kind "queue"`},
		{"invalid layout", map[string]WorkspaceView{"v": {Title: "v", Layout: "spiral"}}, `invalid layout "spiral"`},
		{"selects nothing", map[string]WorkspaceView{"v": {Title: "v", IncludeFrom: []string{"team_a"}, IncludeKinds: []string{"system"}, ExcludeKinds: []string{"system"}}}, `view "v" selects no elements`},
		{"key of a prefixed model view", map[string]WorkspaceView{"a_ctx": {Title: "v"}}, `merged view ID "a_ctx" is produced by more than one model`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merged, err := mergeWithViews(t, tt.views)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) || merged != nil {
				t.Fatalf("merged = %v, err = %v, want an error containing %q", merged != nil, err, tt.wantErr)
			}
		})
	}
}

func TestWorkspaceViews_AllProblemsAreReportedInKeyOrder(t *testing.T) {
	_, err := mergeWithViews(t, map[string]WorkspaceView{
		"b_view": {Title: "b", IncludeKinds: []string{"queue"}},
		"a_view": {Title: "a", Layout: "spiral"},
	})
	if err == nil {
		t.Fatal("expected errors")
	}
	msg := err.Error()
	if !strings.Contains(msg, `"a_view"`) || !strings.Contains(msg, `"b_view"`) || strings.Index(msg, `"b_view"`) < strings.Index(msg, `"a_view"`) {
		t.Errorf("problems are not complete or not in key order:\n%s", msg)
	}
}

func TestValidateWorkspace_ReportsViewProblems(t *testing.T) {
	cfg := &Config{Views: map[string]WorkspaceView{
		"ok":  {Title: "ok"},
		"bad": {Title: "bad", IncludeKinds: []string{"queue"}},
	}}
	problems := ValidateWorkspace(cfg, viewTestModels())
	if len(problems) != 1 || !strings.Contains(problems[0].Error(), `view "bad": unknown element kind "queue"`) {
		t.Fatalf("problems = %v", problems)
	}
}

func TestValidateWorkspace_ViewsOfModelsThatCannotBeMerged(t *testing.T) {
	loaded := viewTestModels()
	loaded[1].Ref.Prefix = "a" // same prefix as team a: the merge fails
	cfg := &Config{Views: map[string]WorkspaceView{"v": {Title: "v", IncludeFrom: []string{"team_z"}}}}
	msg := errorsText(ValidateWorkspace(cfg, loaded))
	if !strings.Contains(msg, `include-from refers to unknown model "team_z"`) || !strings.Contains(msg, "more than one model") {
		t.Errorf("problems:\n%s", msg)
	}
}

func errorsText(errs []error) string {
	var b strings.Builder
	for _, e := range errs {
		b.WriteString(e.Error() + "\n")
	}
	return b.String()
}

func TestMergeWorkspace_ViewsDoNotChangeTheLoadedModels(t *testing.T) {
	loaded := viewTestModels()
	cfg := &Config{Views: map[string]WorkspaceView{"all": {Title: "All"}}}
	for i := 0; i < 2; i++ {
		if _, err := MergeWorkspace(cfg, loaded); err != nil {
			t.Fatal(err)
		}
	}
	if len(loaded[0].Model.Views) != 1 {
		t.Errorf("a workspace view was added to the loaded model: %v", loaded[0].Model.Views)
	}
}
