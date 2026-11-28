package sync

import (
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/drawio"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// pointsOf returns the waypoint coordinates of the connector cells of a page.
func pointsOf(page *drawio.Page) map[string]string {
	pts := make(map[string]string)
	for _, c := range page.FindAllConnectors() {
		if pt := c.FindElement("mxGeometry/Array[@as='points']/mxPoint"); pt != nil {
			pts[c.SelectAttrValue("id", "")] = pt.SelectAttrValue("x", "") + "," + pt.SelectAttrValue("y", "")
		}
	}
	return pts
}

// #717: A->B and B->A were drawn on top of each other. The forward sync gives
// them waypoints on opposite sides.
func TestSync_OppositeRelationshipsGetSeparateRoutes(t *testing.T) {
	m := liftingModel(
		model.Relationship{From: "shop.web", To: "shop.lib", Label: "submits requests"},
		model.Relationship{From: "shop.lib", To: "shop.web", Label: "sends notifications"},
	)
	ts := minimalTemplates(t)
	doc := shopContainersDoc()

	Run(m, doc, emptyState(), ts, nil)
	page := requirePage(t, doc, "view-containers")

	pts := pointsOf(page)
	if len(pts) != 2 {
		t.Fatalf("both connectors need a waypoint, got %v", pts)
	}
	seen := map[string]bool{}
	for id, p := range pts {
		if seen[p] {
			t.Errorf("connector %s has the same waypoint %s as the other", id, p)
		}
		seen[p] = true
	}

	// A second run with nothing to do keeps the routes.
	Run(m, doc, buildTestState(t, m, doc), ts, nil)
	again := pointsOf(page)
	for id, p := range pts {
		if again[id] != p {
			t.Errorf("connector %s moved from %s to %s", id, p, again[id])
		}
	}
}

func TestSync_OneWayRelationshipKeepsTheAutomaticRoute(t *testing.T) {
	m := liftingModel(model.Relationship{From: "shop.web", To: "shop.lib", Label: "calls"})
	doc := shopContainersDoc()
	Run(m, doc, emptyState(), minimalTemplates(t), nil)
	if pts := pointsOf(requirePage(t, doc, "view-containers")); len(pts) != 0 {
		t.Errorf("unexpected waypoints %v", pts)
	}
}
