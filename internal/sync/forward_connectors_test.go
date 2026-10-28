package sync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/drawio"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// liftingModel returns a model with a scoped containers view that shows
// shop.web and shop.lib but not shop.lib's children, so a relationship to
// shop.lib.core is lifted to shop.lib on the page.
func liftingModel(rels ...model.Relationship) *model.BausteinsichtModel {
	m := shopModel(map[string]model.Element{
		"web": {Kind: "container", Title: "Web"},
		"lib": {Kind: "container", Title: "Lib", Children: map[string]model.Element{
			"core": {Kind: "component", Title: "Core"},
			"util": {Kind: "component", Title: "Util"},
		}},
	})
	m.Views["containers"] = model.View{
		Title:   "Container View",
		Scope:   "shop",
		Include: []string{"shop.web", "shop.lib"},
	}
	m.Relationships = rels
	return m
}

// syncTwice runs a sync on m1, persists the state, then syncs m2 against the
// same document and returns the containers page.
func syncTwice(t *testing.T, m1, m2 *model.BausteinsichtModel) *drawio.Page {
	t.Helper()
	ts := minimalTemplates(t)
	doc := shopContainersDoc()

	Run(m1, doc, emptyState(), ts, nil)
	state := buildTestState(t, m1, doc)
	Run(m2, doc, state, ts, nil)
	return requirePage(t, doc, "view-containers")
}

// buildTestState snapshots m and doc as the persisted sync state; BuildState
// hashes real files, so placeholder files are written to a temp dir.
func buildTestState(t *testing.T, m *model.BausteinsichtModel, doc *drawio.Document) *SyncState {
	t.Helper()
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "m.jsonc")
	drawioPath := filepath.Join(dir, "d.drawio")
	for _, p := range []string{modelPath, drawioPath} {
		if err := os.WriteFile(p, []byte("{}"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	state, err := BuildState(m, doc, modelPath, drawioPath)
	if err != nil {
		t.Fatalf("BuildState: %v", err)
	}
	return state
}

// connectorsBetween returns the connector cells on page from src to tgt
// (page-scoped cell IDs).
func connectorsBetween(page *drawio.Page, src, tgt string) []string {
	var ids []string
	for _, c := range page.FindAllConnectors() {
		if c.SelectAttrValue("source", "") == src && c.SelectAttrValue("target", "") == tgt {
			ids = append(ids, c.SelectAttrValue("id", ""))
		}
	}
	return ids
}

// TestSync_LiftedConnectorSurvivesIndexShiftOnce reproduces the duplicate
// connectors seen in the self-hosted arc42 model (UC-SYNC, forward sync):
// inserting a relationship before a lifted one shifts its index; the old
// lifted connector was never removed, so each shift left one more copy.
func TestSync_LiftedConnectorSurvivesIndexShiftOnce(t *testing.T) {
	lifted := model.Relationship{From: "shop.web", To: "shop.lib.core", Label: "calls core"}
	inserted := model.Relationship{From: "shop.lib", To: "shop.web", Label: "notifies"}

	page := syncTwice(t, liftingModel(lifted), liftingModel(inserted, lifted))

	got := connectorsBetween(page, "containers--shop.web", "containers--shop.lib")
	if len(got) != 1 {
		t.Fatalf("want exactly 1 connector shop.web→shop.lib after index shift, got %d: %v", len(got), got)
	}
	if !strings.HasSuffix(got[0], "-1") {
		t.Errorf("connector should carry the relationship's current index 1, got %s", got[0])
	}
}

// TestSync_DeletingLiftedRelationshipRemovesConnector verifies that deleting
// the only relationship behind a lifted connector removes the connector.
func TestSync_DeletingLiftedRelationshipRemovesConnector(t *testing.T) {
	lifted := model.Relationship{From: "shop.web", To: "shop.lib.core", Label: "calls core"}

	page := syncTwice(t, liftingModel(lifted), liftingModel())

	if got := connectorsBetween(page, "containers--shop.web", "containers--shop.lib"); len(got) != 0 {
		t.Fatalf("lifted connector should be removed with its relationship, got %v", got)
	}
}

// TestSync_StaleLiftedConnectorsAreReconciled verifies that connectors left
// behind by earlier syncs (managed IDs whose relationship index no longer
// maps to this pair) are removed, so an affected diagram heals itself.
func TestSync_StaleLiftedConnectorsAreReconciled(t *testing.T) {
	ts := minimalTemplates(t)
	doc := shopContainersDoc()
	lifted := model.Relationship{From: "shop.web", To: "shop.lib.core", Label: "calls core"}
	m := liftingModel(lifted)
	Run(m, doc, emptyState(), ts, nil)
	page := requirePage(t, doc, "view-containers")
	for _, idx := range []int{7, 9} { // stale copies from earlier index positions
		page.CreateConnector(drawio.ConnectorData{
			From: "shop.web", To: "shop.lib", Label: "calls core",
			SourceRef: "containers--shop.web", TargetRef: "containers--shop.lib", Index: idx,
		}, "")
	}
	state := buildTestState(t, m, doc)

	Run(m, doc, state, ts, nil)

	if got := connectorsBetween(page, "containers--shop.web", "containers--shop.lib"); len(got) != 1 {
		t.Fatalf("want 1 connector after reconciliation, got %d: %v", len(got), got)
	}
}

// TestSync_DirectRelationshipWinsOverEarlierLiftedOne verifies the #197 rule
// also for populated connectors: a direct relationship and a lifted one on
// the same pair render as one connector, even if the lifted one comes first.
func TestSync_DirectRelationshipWinsOverEarlierLiftedOne(t *testing.T) {
	m := liftingModel(
		model.Relationship{From: "shop.web", To: "shop.lib.core", Label: "calls core"},
		model.Relationship{From: "shop.web", To: "shop.lib", Label: "uses"},
	)

	page := syncTwice(t, m, m)

	got := connectorsBetween(page, "containers--shop.web", "containers--shop.lib")
	if len(got) != 1 || !strings.HasSuffix(got[0], "-1") {
		t.Fatalf("want only the direct connector (index 1), got %v", got)
	}
}

// TestSync_UserDrawnConnectorIsKept verifies reconciliation only touches
// connectors Bausteinsicht created (rel-<view>--… IDs).
func TestSync_UserDrawnConnectorIsKept(t *testing.T) {
	ts := minimalTemplates(t)
	doc := shopContainersDoc()
	m := liftingModel()
	Run(m, doc, emptyState(), ts, nil)
	page := requirePage(t, doc, "view-containers")
	cell := page.Root().CreateElement("mxCell")
	cell.CreateAttr("id", "user-edge-42")
	cell.CreateAttr("edge", "1")
	cell.CreateAttr("parent", "1")

	if removed := reconcileViewConnectors(page, "containers", nil); removed != 0 {
		t.Fatalf("user-drawn connector must not be removed, removed %d", removed)
	}
}

// addWaypoint gives the connector cell id a manual waypoint, as a user does
// when routing an edge in draw.io.
func addWaypoint(t *testing.T, page *drawio.Page, id string) {
	t.Helper()
	for _, c := range page.FindAllConnectors() {
		if c.SelectAttrValue("id", "") != id {
			continue
		}
		pts := c.SelectElement("mxGeometry").CreateElement("Array")
		pts.CreateAttr("as", "points")
		pt := pts.CreateElement("mxPoint")
		pt.CreateAttr("x", "123")
		pt.CreateAttr("y", "456")
		return
	}
	t.Fatalf("connector %s not found", id)
}

func hasWaypoint(page *drawio.Page, id string) bool {
	for _, c := range page.FindAllConnectors() {
		if c.SelectAttrValue("id", "") == id {
			return c.FindElement("mxGeometry/Array/mxPoint[@x='123']") != nil
		}
	}
	return false
}

// syncTwiceWithWaypoint syncs m1, routes the connector firstID by hand,
// then syncs m2 and returns the page.
func syncTwiceWithWaypoint(t *testing.T, m1, m2 *model.BausteinsichtModel, firstID string) *drawio.Page {
	t.Helper()
	ts := minimalTemplates(t)
	doc := shopContainersDoc()
	Run(m1, doc, emptyState(), ts, nil)
	page := requirePage(t, doc, "view-containers")
	addWaypoint(t, page, firstID)
	Run(m2, doc, buildTestState(t, m1, doc), ts, nil)
	return page
}

// TestSync_IndexShiftKeepsConnectorRouting: connector IDs carry the
// relationship's array index, so inserting a relationship earlier in the
// array renamed every following connector. They were recreated without
// their waypoints; now the existing cell is re-keyed and keeps its routing.
func TestSync_IndexShiftKeepsConnectorRouting(t *testing.T) {
	direct := model.Relationship{From: "shop.web", To: "shop.lib", Label: "uses"}
	inserted := model.Relationship{From: "shop.lib", To: "shop.web", Label: "notifies"}

	page := syncTwiceWithWaypoint(t, liftingModel(direct), liftingModel(inserted, direct),
		"rel-containers--shop.web-containers--shop.lib-0")

	id := "rel-containers--shop.web-containers--shop.lib-1"
	if !hasWaypoint(page, id) {
		t.Fatalf("connector %s should keep its waypoint after the index shift; connectors: %v",
			id, connectorsBetween(page, "containers--shop.web", "containers--shop.lib"))
	}
}

// TestSync_NewLiftedRelationshipKeepsExistingConnector: adding a component
// relationship that lifts onto an already drawn pair (cross-container edge
// on a container view) created a second connector without waypoints next
// to the routed one. The pair keeps exactly one connector, with its routing.
func TestSync_NewLiftedRelationshipKeepsExistingConnector(t *testing.T) {
	notify := model.Relationship{From: "shop.lib", To: "shop.web", Label: "notifies"}
	old := model.Relationship{From: "shop.web", To: "shop.lib.core", Label: "calls core"}
	added := model.Relationship{From: "shop.web", To: "shop.lib.util", Label: "calls util"}

	for name, m2 := range map[string]*model.BausteinsichtModel{
		"appended": liftingModel(notify, old, added),
		"inserted": liftingModel(added, notify, old), // added becomes the pair's lowest index
	} {
		t.Run(name, func(t *testing.T) {
			page := syncTwiceWithWaypoint(t, liftingModel(notify, old), m2,
				"rel-containers--shop.web-containers--shop.lib-1")
			got := connectorsBetween(page, "containers--shop.web", "containers--shop.lib")
			if len(got) != 1 || !hasWaypoint(page, got[0]) {
				t.Fatalf("want 1 routed connector web→lib, got %v", got)
			}
		})
	}
}
