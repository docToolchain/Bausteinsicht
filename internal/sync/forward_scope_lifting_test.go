package sync

import (
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// scopeLiftingModel returns a model with a partial container view ("io")
// of system shop that shows shop.web but not shop.lib. A relationship to
// shop.lib.core would lift to the scope boundary shop on that page.
func scopeLiftingModel(rels ...model.Relationship) *model.BausteinsichtModel {
	m := shopModel(map[string]model.Element{
		"web": {Kind: "container", Title: "Web"},
		"lib": {Kind: "container", Title: "Lib", Children: map[string]model.Element{
			"core": {Kind: "component", Title: "Core"},
		}},
	})
	m.Views["containers"] = model.View{Title: "IO", Scope: "shop", Include: []string{"shop.web"}}
	m.Relationships = rels
	return m
}

// connectorsTouching returns the IDs of connectors on the containers page
// with an endpoint on cellID.
func connectorsTouching(t *testing.T, m *model.BausteinsichtModel, cellID string) []string {
	t.Helper()
	doc := shopContainersDoc()
	Run(m, doc, emptyState(), minimalTemplates(t), nil)
	page := requirePage(t, doc, "view-containers")
	var ids []string
	for _, c := range page.FindAllConnectors() {
		if c.SelectAttrValue("source", "") == cellID || c.SelectAttrValue("target", "") == cellID {
			ids = append(ids, c.SelectAttrValue("id", ""))
		}
	}
	return ids
}

// TestSync_NoConnectorLiftedOntoScopeBoundary reproduces the boundary edges
// in the self-hosted arc42 partial container views (e.g. "exports Mermaid
// diagrams" cli→bausteinsicht on containers-sync): an endpoint missing from
// the view was lifted to its parent, which is the view's own scope boundary.
// The boundary encloses every element on the page, so such an edge says
// nothing; the relationship belongs to a view that shows both endpoints.
func TestSync_NoConnectorLiftedOntoScopeBoundary(t *testing.T) {
	m := scopeLiftingModel(
		model.Relationship{From: "shop.web", To: "shop.lib.core", Label: "calls core"},
		model.Relationship{From: "shop.lib", To: "shop.web", Label: "notifies"},
	)

	if got := connectorsTouching(t, m, "containers--shop"); len(got) != 0 {
		t.Fatalf("no connector may be lifted onto the scope boundary, got %v", got)
	}
}

// TestSync_DirectRelationshipToScopeStillRendered keeps AC-9.4 (#217): a
// relationship that targets the scope element in the model is drawn to the
// boundary.
func TestSync_DirectRelationshipToScopeStillRendered(t *testing.T) {
	m := scopeLiftingModel(model.Relationship{From: "shop.web", To: "shop", Label: "part of"})

	got := connectorsTouching(t, m, "containers--shop")
	if len(got) != 1 || !strings.HasSuffix(got[0], "-0") {
		t.Fatalf("want the direct web→shop connector, got %v", got)
	}
}

// TestComputeVisibleRelationships_ScopeLiftingIsNotVisible: a relationship
// that only lifts onto a view's scope boundary has no connector anywhere,
// so it must not count as visible — otherwise reverse sync would read the
// missing connector as a user deletion (#167).
func TestComputeVisibleRelationships_ScopeLiftingIsNotVisible(t *testing.T) {
	m := scopeLiftingModel(model.Relationship{From: "shop.web", To: "shop.lib.core", Label: "calls core"})

	if visible := computeVisibleRelationships(m); visible[relKey("shop.web", "shop.lib.core", 0)] {
		t.Fatal("relationship lifted only onto the scope boundary must not be visible")
	}
}
