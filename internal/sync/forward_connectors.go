package sync

import (
	"sort"
	"strings"

	"github.com/beevik/etree"

	"github.com/docToolchain/Bausteinsicht/internal/drawio"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// viewConnector is one connector a view page should carry: a model
// relationship with its (possibly lifted) endpoints.
type viewConnector struct {
	from, to string
	index    int
	label    string
	dashed   bool // any relationship rendered by this connector is dashed (#518)
}

// expectedViewConnectors returns the connectors a view page should carry for
// the model's relationships. Direct relationships are always rendered; a
// lifted relationship is rendered only if no direct relationship and no
// lower-index lifted relationship already covers the same (from,to) pair
// (#142, #197). Every sync derives the same set, independent of which
// relationships changed, so the page converges on it.
func expectedViewConnectors(m *model.BausteinsichtModel, elemSet map[string]bool, scopeID string) []viewConnector {
	dashedForPair := computeDashedPairs(m, elemSet, scopeID)
	var direct, lifted []viewConnector
	directPairs := make(map[string]bool)
	for i, rel := range m.Relationships {
		from, to, ok := liftModelRelationshipEndpoints(rel, elemSet, scopeID)
		if !ok {
			continue
		}
		vc := viewConnector{from: from, to: to, index: i, label: rel.Label, dashed: dashedForPair[from+"->"+to]}
		if from == rel.From && to == rel.To {
			direct = append(direct, vc)
			directPairs[from+"->"+to] = true
		} else {
			lifted = append(lifted, vc)
		}
	}

	result := direct
	liftedSeen := make(map[string]bool)
	for _, vc := range lifted {
		pair := vc.from + "->" + vc.to
		if directPairs[pair] || liftedSeen[pair] {
			continue
		}
		liftedSeen[pair] = true
		result = append(result, vc)
	}
	return result
}

// expectedConnectorIDs returns the page-scoped cell IDs of conns on viewID.
func expectedConnectorIDs(viewID string, conns []viewConnector) map[string]bool {
	ids := make(map[string]bool, len(conns))
	for _, vc := range conns {
		ids[drawio.ConnectorID(scopedCellID(viewID, vc.from), scopedCellID(viewID, vc.to), vc.index)] = true
	}
	return ids
}

// reconcileViewConnectors removes connectors Bausteinsicht created on the
// view page (IDs "rel-<viewID>--…") that are not in expected, and returns
// how many it removed. Such stale connectors arise when a relationship's
// index shifts or a lifted relationship is deleted: deletion looks the
// connector up by the unlifted endpoints and misses the lifted one, so
// each shift left another copy on the page. User-drawn connectors carry
// other IDs and are kept.
func reconcileViewConnectors(page *drawio.Page, viewID string, expected map[string]bool) int {
	root := page.Root()
	if root == nil {
		return 0
	}
	prefix := "rel-" + viewID + "--"
	removed := 0
	for _, cell := range page.FindAllConnectors() {
		id := cell.SelectAttrValue("id", "")
		if !strings.HasPrefix(id, prefix) || expected[id] {
			continue
		}
		root.RemoveChild(cell)
		removed++
	}
	return removed
}

// rekeyStaleConnectors gives existing connectors the cell ID of their
// relationship's current index before changes are applied. Connector IDs
// carry the index into the relationships array, so inserting or removing a
// relationship renames every following connector, and a new lower-index
// relationship can take over a lifted pair. Without re-keying, the old cell
// would be deleted and a new one created, losing the user's waypoints and
// label position. For each expected connector missing on the page, a
// managed connector with the same source and target whose ID is no longer
// expected is renamed, relabelled and restyled (dashed) instead. Returns
// how many it re-keyed.
func rekeyStaleConnectors(page *drawio.Page, viewID string, conns []viewConnector) int {
	expected := expectedConnectorIDs(viewID, conns)
	prefix := "rel-" + viewID + "--"
	stale := make(map[string][]*etree.Element) // "src->tgt" → stale cells
	present := make(map[string]bool)
	for _, cell := range page.FindAllConnectors() {
		id := cell.SelectAttrValue("id", "")
		present[id] = true
		if strings.HasPrefix(id, prefix) && !expected[id] {
			pair := cell.SelectAttrValue("source", "") + "->" + cell.SelectAttrValue("target", "")
			stale[pair] = append(stale[pair], cell)
		}
	}

	// Match parallel stale cells in old-index order so each relationship
	// keeps the routing of its predecessor, independent of document order.
	for _, cells := range stale {
		sort.SliceStable(cells, func(i, j int) bool {
			return parseConnectorIndex(cells[i].SelectAttrValue("id", "")) < parseConnectorIndex(cells[j].SelectAttrValue("id", ""))
		})
	}

	rekeyed := 0
	for _, vc := range conns {
		src, tgt := scopedCellID(viewID, vc.from), scopedCellID(viewID, vc.to)
		id := drawio.ConnectorID(src, tgt, vc.index)
		cells := stale[src+"->"+tgt]
		if present[id] || len(cells) == 0 {
			continue
		}
		cell := cells[0]
		stale[src+"->"+tgt] = cells[1:]
		setAttrOn(cell, "id", id)
		setAttrOn(cell, "value", vc.label)
		dashed := "dashed=0;"
		if vc.dashed {
			dashed = "dashed=1;"
		}
		setAttrOn(cell, "style", mergeStyles(cell.SelectAttrValue("style", ""), dashed))
		present[id] = true
		rekeyed++
	}
	return rekeyed
}
