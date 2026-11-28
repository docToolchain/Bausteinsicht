// Package layout arranges the elements of every page of a draw.io diagram.
//
// Each page is laid out on its own, from what is on that page: the element
// cells (nested ones inside their container or boundary) and the connectors
// between them. The model is not needed, and elements keep their size.
package layout

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/beevik/etree"
	"github.com/docToolchain/Bausteinsicht/internal/drawio"
)

// Options controls LayoutDocument.
type Options struct {
	Algorithm      string // "hierarchical" (default; the only one)
	RankDir        string // "TB" (top-to-bottom, default) or "LR" (left-to-right)
	PreservePinned bool   // leave elements marked bausteinsicht-pinned=true (and what is inside them) alone
	View           string // lay out only the page with this ID; empty means every page
}

// Report says what LayoutDocument did.
type Report struct {
	Pages    int // pages laid out
	Elements int // elements positioned
	Pinned   int // pinned elements left where they were
}

const (
	defaultWidth  = 160.0
	defaultHeight = 60.0
	nodeGap       = 40.0 // between elements of one layer
	layerGap      = 80.0 // between layers
	startX        = 40.0
	startY        = 70.0 // leaves room for the back-navigation button
	containerPadX = 20.0 // inside a container or boundary
	containerPadY = 20.0
	containerHead = 80.0 // room for the container's own label
	minContainerW = 300.0
	minContainerH = 200.0
)

// node is an element cell on a page.
type node struct {
	id       string // draw.io cell ID
	geom     *etree.Element
	parent   string
	w, h     float64
	x, y     float64
	pinned   bool
	children []*node
}

type edge struct{ from, to string }

// LayoutDocument lays out the pages of doc.
func LayoutDocument(doc *drawio.Document, opts Options) (Report, error) {
	if opts.RankDir == "" {
		opts.RankDir = "TB"
	}
	if opts.RankDir != "TB" && opts.RankDir != "LR" {
		return Report{}, fmt.Errorf("unknown rank direction %q: use TB or LR", opts.RankDir)
	}

	var report Report
	found := false
	for _, page := range doc.Pages() {
		if opts.View != "" && page.ID() != opts.View {
			continue
		}
		found = true
		elements, pinned := layoutPage(page, opts)
		if elements > 0 {
			report.Pages++
		}
		report.Elements += elements
		report.Pinned += pinned
	}
	if opts.View != "" && !found {
		return report, fmt.Errorf("page %q not found in the diagram", opts.View)
	}
	return report, nil
}

// layoutPage lays out one page and returns the number of positioned and of
// pinned elements.
func layoutPage(page *drawio.Page, opts Options) (positioned, pinned int) {
	root := page.Root()
	if root == nil {
		return 0, 0
	}
	// Make the text cells hold their text first: the shapes may have to grow,
	// and the layout places them by their size (#718).
	for _, obj := range page.FindAllElements() {
		page.FitElementText(obj)
	}
	byID, top := readNodes(root)
	edges := readEdges(page)

	var place func(level []*node, isPage bool) (w, h float64)
	place = func(level []*node, isPage bool) (float64, float64) {
		movable := make([]*node, 0, len(level))
		for _, n := range level {
			if n.pinned && opts.PreservePinned {
				pinned++
				continue
			}
			movable = append(movable, n)
		}
		// Containers first: their size depends on what is inside them.
		for _, n := range movable {
			if len(n.children) > 0 {
				cw, ch := place(n.children, false)
				n.w = maxFloat(cw+2*containerPadX, minContainerW)
				n.h = maxFloat(ch+containerHead+containerPadY, minContainerH)
			}
		}
		originX, originY := containerPadX, containerHead
		if isPage {
			originX, originY = startX, startY
		}
		w, h := arrange(movable, levelEdges(movable, byID, edges), opts.RankDir, originX, originY)
		positioned += len(movable)
		return w, h
	}
	place(top, true)

	for _, n := range byID {
		if !n.pinned || !opts.PreservePinned {
			writeGeometry(n)
		}
	}
	clearWaypoints(page)
	// The connectors of opposite directions lose their separate routes with the
	// waypoints; give them new ones for the new positions (#717).
	page.SeparateOppositeConnectors()
	return positioned, pinned
}

// readNodes collects the element cells of a page (objects with a
// bausteinsicht_id) and builds their nesting. top are the cells that sit
// directly on the page.
func readNodes(root *etree.Element) (byID map[string]*node, top []*node) {
	byID = make(map[string]*node)
	var order []*node
	for _, obj := range root.ChildElements() {
		if obj.Tag != "object" || obj.SelectAttr("bausteinsicht_id") == nil {
			continue
		}
		n := &node{id: obj.SelectAttrValue("id", ""),
			pinned: obj.SelectAttrValue("bausteinsicht-pinned", "") == "true"}
		if cell := obj.SelectElement("mxCell"); cell != nil {
			n.parent = cell.SelectAttrValue("parent", "")
			n.geom = cell.SelectElement("mxGeometry")
		}
		if n.id == "" || n.geom == nil {
			continue
		}
		n.w = attrFloat(n.geom, "width", defaultWidth)
		n.h = attrFloat(n.geom, "height", defaultHeight)
		byID[n.id] = n
		order = append(order, n)
	}
	sort.Slice(order, func(i, j int) bool { return order[i].id < order[j].id })
	for _, n := range order {
		if p, ok := byID[n.parent]; ok {
			p.children = append(p.children, n)
		} else {
			top = append(top, n)
		}
	}
	return byID, top
}

func readEdges(page *drawio.Page) []edge {
	var edges []edge
	for _, c := range page.FindAllConnectors() {
		from, to := c.SelectAttrValue("source", ""), c.SelectAttrValue("target", "")
		if from != "" && to != "" {
			edges = append(edges, edge{from, to})
		}
	}
	return edges
}

// levelEdges maps connector ends to the elements of one level: an end that is
// nested inside one of them counts as that element, so a connector into a
// boundary's child becomes a connector to the boundary on the outer level.
func levelEdges(level []*node, byID map[string]*node, edges []edge) []edge {
	in := make(map[string]bool, len(level))
	for _, n := range level {
		in[n.id] = true
	}
	ancestor := func(id string) string {
		for guard := 0; guard < 64; guard++ {
			if in[id] {
				return id
			}
			n, ok := byID[id]
			if !ok {
				return ""
			}
			id = n.parent
		}
		return ""
	}
	seen := make(map[edge]bool)
	var result []edge
	for _, e := range edges {
		from, to := ancestor(e.from), ancestor(e.to)
		if from == "" || to == "" || from == to || seen[edge{from, to}] {
			continue
		}
		seen[edge{from, to}] = true
		result = append(result, edge{from, to})
	}
	return result
}

// arrange places the nodes of one level in layers and returns the size of the
// area they cover, measured from (originX, originY).
func arrange(nodes []*node, edges []edge, rankDir string, originX, originY float64) (w, h float64) {
	if len(nodes) == 0 {
		return 0, 0
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].id < nodes[j].id })
	layers := orderLayers(assignLayers(nodes, edges), edges)

	horizontal := rankDir == "LR"
	// extent along the layer axis (rank) and across it (position in a layer)
	rankSize := func(n *node) float64 {
		if horizontal {
			return n.w
		}
		return n.h
	}
	crossSize := func(n *node) float64 {
		if horizontal {
			return n.h
		}
		return n.w
	}

	layerCross := make([]float64, len(layers))
	maxCross := 0.0
	for i, layer := range layers {
		for j, n := range layer {
			if j > 0 {
				layerCross[i] += nodeGap
			}
			layerCross[i] += crossSize(n)
		}
		maxCross = maxFloat(maxCross, layerCross[i])
	}

	rank := 0.0
	for i, layer := range layers {
		rankMax := 0.0
		cross := (maxCross - layerCross[i]) / 2
		for _, n := range layer {
			if horizontal {
				n.x, n.y = originX+rank, originY+cross
			} else {
				n.x, n.y = originX+cross, originY+rank
			}
			cross += crossSize(n) + nodeGap
			rankMax = maxFloat(rankMax, rankSize(n))
		}
		rank += rankMax + layerGap
	}
	rank -= layerGap
	if horizontal {
		return rank, maxCross
	}
	return maxCross, rank
}

// assignLayers puts every node on the layer after its latest predecessor
// (longest path). Edges that close a cycle are dropped first: the nodes are
// walked in ID order along the connectors, and an edge back to a node that is
// still being walked is the one that closes the cycle. Because the walk starts
// with the smallest ID, the result does not depend on the order of the cells.
func assignLayers(nodes []*node, edges []edge) [][]*node {
	out := make(map[string][]string)
	for _, e := range edges {
		out[e.from] = append(out[e.from], e.to)
	}
	for id := range out {
		sort.Strings(out[id])
	}

	const (
		walking = 1
		done    = 2
	)
	state := make(map[string]int)
	preds := make(map[string][]string)
	var walk func(id string)
	walk = func(id string) {
		state[id] = walking
		for _, to := range out[id] {
			if state[to] == walking {
				continue // closes a cycle
			}
			preds[to] = append(preds[to], id)
			if state[to] == 0 {
				walk(to)
			}
		}
		state[id] = done
	}
	for _, n := range nodes {
		if state[n.id] == 0 {
			walk(n.id)
		}
	}

	depth := make(map[string]int)
	var depthOf func(id string) int
	depthOf = func(id string) int {
		if d, ok := depth[id]; ok {
			return d
		}
		d := 0
		for _, p := range preds[id] {
			d = maxInt(d, depthOf(p)+1)
		}
		depth[id] = d
		return d
	}
	maxDepth := 0
	for _, n := range nodes {
		maxDepth = maxInt(maxDepth, depthOf(n.id))
	}
	layers := make([][]*node, maxDepth+1)
	for _, n := range nodes { // already sorted by ID
		layers[depth[n.id]] = append(layers[depth[n.id]], n)
	}
	return layers
}

// orderLayers reduces crossings with a few barycenter sweeps. Ties keep the ID
// order, so the result is the same on every run.
func orderLayers(layers [][]*node, edges []edge) [][]*node {
	neighbours := make(map[string][]string)
	for _, e := range edges {
		neighbours[e.from] = append(neighbours[e.from], e.to)
		neighbours[e.to] = append(neighbours[e.to], e.from)
	}
	index := make(map[string]int)
	reindex := func() {
		for _, layer := range layers {
			for i, n := range layer {
				index[n.id] = i
			}
		}
	}
	reindex()
	sweep := func(fixed, moving int) {
		inFixed := make(map[string]bool)
		for _, n := range layers[fixed] {
			inFixed[n.id] = true
		}
		bary := make(map[string]float64)
		for i, n := range layers[moving] {
			sum, cnt := 0.0, 0
			for _, nb := range neighbours[n.id] {
				if inFixed[nb] {
					sum += float64(index[nb])
					cnt++
				}
			}
			bary[n.id] = float64(i)
			if cnt > 0 {
				bary[n.id] = sum / float64(cnt)
			}
		}
		sort.SliceStable(layers[moving], func(a, b int) bool {
			return bary[layers[moving][a].id] < bary[layers[moving][b].id]
		})
		reindex()
	}
	for pass := 0; pass < 2; pass++ {
		for i := 1; i < len(layers); i++ {
			sweep(i-1, i)
		}
		for i := len(layers) - 2; i >= 0; i-- {
			sweep(i+1, i)
		}
	}
	return layers
}

// writeGeometry writes a node's position and size back to the cell. Nested
// cells get coordinates relative to their container, as draw.io expects.
func writeGeometry(n *node) {
	n.geom.CreateAttr("x", fmt.Sprintf("%.0f", n.x))
	n.geom.CreateAttr("y", fmt.Sprintf("%.0f", n.y))
	n.geom.CreateAttr("width", fmt.Sprintf("%.0f", n.w))
	n.geom.CreateAttr("height", fmt.Sprintf("%.0f", n.h))
}

// clearWaypoints drops the stored waypoints of every connector: they belong to
// the old positions, and without them draw.io routes the connectors again.
func clearWaypoints(page *drawio.Page) {
	for _, c := range page.FindAllConnectors() {
		geom := c.SelectElement("mxGeometry")
		if geom == nil {
			continue
		}
		for _, arr := range geom.SelectElements("Array") {
			if arr.SelectAttrValue("as", "") == "points" {
				geom.RemoveChild(arr)
			}
		}
	}
}

func attrFloat(e *etree.Element, name string, fallback float64) float64 {
	if v, err := strconv.ParseFloat(e.SelectAttrValue(name, ""), 64); err == nil && v > 0 {
		return v
	}
	return fallback
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
