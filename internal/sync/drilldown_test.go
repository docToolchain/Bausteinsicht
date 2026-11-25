package sync

import (
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// drillDownModel has five views that share the scope "sys", like the arc42
// model of this repository (#661): an overview with all containers and four
// partial views with different numbers of containers.
func drillDownModel() *model.BausteinsichtModel {
	children := map[string]model.Element{}
	for _, id := range []string{"a", "b", "c", "d", "e", "f"} {
		children[id] = model.Element{Kind: "container", Title: id}
	}
	return &model.BausteinsichtModel{
		Specification: model.Specification{Elements: map[string]model.ElementKind{
			"system":    {Notation: "System", Container: true},
			"container": {Notation: "Container"},
		}},
		Model: map[string]model.Element{
			"sys":   {Kind: "system", Title: "Sys", Children: children},
			"other": {Kind: "system", Title: "Other"},
		},
		Views: map[string]model.View{
			"context":              {Title: "Context", Include: []string{"sys", "other"}},
			"containers":           {Title: "Containers", Scope: "sys", Include: []string{"sys.*"}},
			"containers-analysis":  {Title: "Analysis", Scope: "sys", Include: []string{"sys.a", "sys.b"}},
			"containers-evolution": {Title: "Evolution", Scope: "sys", Include: []string{"sys.a", "sys.b", "sys.c"}},
			"containers-io":        {Title: "IO", Scope: "sys", Include: []string{"sys.a", "sys.b", "sys.c"}},
			"containers-sync":      {Title: "Sync", Scope: "sys", Include: []string{"sys.d"}},
		},
	}
}

// The drill-down target of a scope shared by several views must not depend on
// Go's map iteration order (#661): it is the view that shows most of the
// scope's subtree, and the smallest view key on a tie.
func TestBuildDrillDownLinks_IsDeterministicAndPrefersTheOverview(t *testing.T) {
	m := drillDownModel()
	want := "data:page/id,view-containers"
	for i := 0; i < 100; i++ {
		if got := buildDrillDownLinks(m)["sys"]; got != want {
			t.Fatalf("run %d: link = %q, want %q", i, got, want)
		}
	}
}

func TestBuildDrillDownLinks_TieBreaksBySmallestKey(t *testing.T) {
	m := drillDownModel()
	delete(m.Views, "containers") // no overview: evolution and io both show three containers
	for i := 0; i < 100; i++ {
		if got := buildDrillDownLinks(m)["sys"]; got != "data:page/id,view-containers-evolution" {
			t.Fatalf("run %d: link = %q, want the smallest key among the views with most elements", i, got)
		}
	}
}

func TestBuildDrillDownLinks_SingleViewAndNoScope(t *testing.T) {
	m := drillDownModel()
	m.Views = map[string]model.View{
		"only":    {Title: "Only", Scope: "sys", Include: []string{"sys.a"}},
		"context": {Title: "Context", Include: []string{"sys"}},
	}
	links := buildDrillDownLinks(m)
	if links["sys"] != "data:page/id,view-only" || len(links) != 1 {
		t.Errorf("links = %v, want only sys -> view-only", links)
	}
}

// The back-navigation target (the view that shows the scope element) is
// deterministic as well: the smallest view key among the candidates.
func TestFindParentView_IsDeterministic(t *testing.T) {
	m := drillDownModel()
	m.Views["overview"] = model.View{Title: "Overview", Include: []string{"sys"}}
	for i := 0; i < 100; i++ {
		id, title := findParentView(m, "containers-sync", "sys")
		if id != "context" || title != "Context" {
			t.Fatalf("run %d: parent = %q (%q), want context", i, id, title)
		}
	}
	if id, _ := findParentView(m, "containers-sync", "nonexistent"); id != "" {
		t.Errorf("parent of an element no view shows = %q, want none", id)
	}
}
