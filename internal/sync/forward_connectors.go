package sync

import (
	"strings"

	"github.com/docToolchain/Bausteinsicht/internal/drawio"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// viewConnector is one connector a view page should carry: a model
// relationship with its (possibly lifted) endpoints.
type viewConnector struct {
	from, to string
	index    int
	label    string
}

// expectedViewConnectors returns the connectors a view page should carry for
// the model's relationships. Direct relationships are always rendered; a
// lifted relationship is rendered only if no direct relationship and no
// lower-index lifted relationship already covers the same (from,to) pair
// (#142, #197). Every sync derives the same set, independent of which
// relationships changed, so the page converges on it.
func expectedViewConnectors(m *model.BausteinsichtModel, elemSet map[string]bool, scopeID string) []viewConnector {
	var direct, lifted []viewConnector
	directPairs := make(map[string]bool)
	for i, rel := range m.Relationships {
		from, to, ok := liftModelRelationshipEndpoints(rel, elemSet, scopeID)
		if !ok {
			continue
		}
		vc := viewConnector{from: from, to: to, index: i, label: rel.Label}
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
