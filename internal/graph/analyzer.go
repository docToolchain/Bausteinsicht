package graph

import (
	"sort"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// Analyzer performs graph analysis on model relationships.
type Analyzer struct {
	model *model.BausteinsichtModel
	graph map[string][]string // element ID → list of outgoing relationship targets
	reverse map[string][]string // reverse graph: target → list of incoming sources
}

// NewAnalyzer creates a new graph analyzer.
func NewAnalyzer(m *model.BausteinsichtModel) *Analyzer {
	a := &Analyzer{
		model:   m,
		graph:   make(map[string][]string),
		reverse: make(map[string][]string),
	}

	// Build adjacency lists from relationships
	flatElems, _ := model.FlattenElements(m)
	for id := range flatElems {
		a.graph[id] = []string{}
		a.reverse[id] = []string{}
	}

	for _, rel := range m.Relationships {
		if _, ok := flatElems[rel.From]; ok {
			if _, ok := flatElems[rel.To]; ok {
				a.graph[rel.From] = append(a.graph[rel.From], rel.To)
				a.reverse[rel.To] = append(a.reverse[rel.To], rel.From)
			}
		}
	}

	return a
}

// Analyze performs comprehensive graph analysis.
func (a *Analyzer) Analyze() *GraphAnalysis {
	flatElems, _ := model.FlattenElements(a.model)

	result := &GraphAnalysis{
		ElementCount:      len(flatElems),
		RelationshipCount: len(a.model.Relationships),
	}

	// Find all cycles
	result.Cycles = a.findCycles()
	result.IDAGValid = len(result.Cycles) == 0

	// Calculate centrality metrics
	result.Centrality = a.calculateCentrality()

	// Find strongly connected components
	result.Components = a.findStronglyConnectedComponents()

	// Calculate maximum depth (longest path)
	result.MaxDepth = a.calculateMaxDepth()

	return result
}

// tarjanSCCs runs Tarjan's strongly-connected-components algorithm over graph,
// calling onSCC once per discovered SCC. seeds controls which vertices to start
// from; passing every key in graph gives full coverage.
func tarjanSCCs(graph map[string][]string, seeds []string, onSCC func([]string)) {
	index := 0
	stack := []string{}
	nodeInfo := make(map[string]*NodeInfo)

	var strongconnect func(string)
	strongconnect = func(v string) {
		nodeInfo[v] = &NodeInfo{index: index, lowlink: index, onStack: true}
		index++
		stack = append(stack, v)

		for _, w := range graph[v] {
			if _, ok := nodeInfo[w]; !ok {
				strongconnect(w)
				nodeInfo[v].lowlink = min(nodeInfo[v].lowlink, nodeInfo[w].lowlink)
			} else if nodeInfo[w].onStack {
				nodeInfo[v].lowlink = min(nodeInfo[v].lowlink, nodeInfo[w].index)
			}
		}

		if nodeInfo[v].lowlink == nodeInfo[v].index {
			var component []string
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				nodeInfo[w].onStack = false
				component = append(component, w)
				if w == v {
					break
				}
			}
			onSCC(component)
		}
	}

	for _, v := range seeds {
		if _, ok := nodeInfo[v]; !ok {
			strongconnect(v)
		}
	}
}

// findCycles detects all cycles using Tarjan's algorithm.
func (a *Analyzer) findCycles() []Cycle {
	var cycles []Cycle
	seeds := make([]string, 0, len(a.graph))
	for v := range a.graph {
		seeds = append(seeds, v)
	}
	tarjanSCCs(a.graph, seeds, func(component []string) {
		if len(component) > 1 {
			cycles = append(cycles, Cycle{Elements: component, Length: len(component)})
		}
	})
	return cycles
}

// betweennessScore returns the simplified betweenness: fraction of other
// elements reachable from id via any path.
func (a *Analyzer) betweennessScore(id string, flatElems map[string]*model.Element) float64 {
	count := 0
	for target := range flatElems {
		if target != id && a.hasPath(id, target) {
			count++
		}
	}
	if n := len(flatElems) - 1; n > 0 {
		return float64(count) / float64(n)
	}
	return 0
}

// closenessScore returns the simplified closeness: inverse of average shortest
// path distance to all reachable elements.
func (a *Analyzer) closenessScore(id string, flatElems map[string]*model.Element) float64 {
	totalDist, reachable := 0, 0
	for target := range flatElems {
		if target != id {
			if dist := a.shortestPath(id, target); dist > 0 {
				totalDist += dist
				reachable++
			}
		}
	}
	if reachable > 0 {
		return 1.0 / (1.0 + float64(totalDist)/float64(reachable))
	}
	return 0
}

// calculateCentrality computes centrality metrics for all elements.
func (a *Analyzer) calculateCentrality() []Centrality {
	flatElems, _ := model.FlattenElements(a.model)
	var results []Centrality

	for id := range flatElems {
		results = append(results, Centrality{
			ID:          id,
			InDegree:    len(a.reverse[id]),
			OutDegree:   len(a.graph[id]),
			Betweenness: a.betweennessScore(id, flatElems),
			Closeness:   a.closenessScore(id, flatElems),
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].ID < results[j].ID
	})
	return results
}

// findStronglyConnectedComponents finds all SCCs in the graph.
func (a *Analyzer) findStronglyConnectedComponents() []Component {
	var components []Component
	componentID := 0
	flatElems, _ := model.FlattenElements(a.model)
	seeds := make([]string, 0, len(flatElems))
	for v := range flatElems {
		seeds = append(seeds, v)
	}
	tarjanSCCs(a.graph, seeds, func(component []string) {
		sort.Strings(component)
		components = append(components, Component{
			ID:       componentID,
			Elements: component,
			IsCycle:  len(component) > 1,
		})
		componentID++
	})
	return components
}

// calculateMaxDepth finds the longest dependency path in the graph.
// In cyclic graphs, returns 0 since there's no defined maximum.
func (a *Analyzer) calculateMaxDepth() int {
	if !a.isDAG() {
		return 0 // Cyclic graph has undefined max depth
	}

	flatElems, _ := model.FlattenElements(a.model)
	maxDepth := 0

	for start := range flatElems {
		depth := a.longestPathDAG(start)
		if depth > maxDepth {
			maxDepth = depth
		}
	}

	return maxDepth
}

// isDAG checks if the graph is acyclic.
func (a *Analyzer) isDAG() bool {
	visited := make(map[string]int) // 0=unvisited, 1=visiting, 2=visited
	var hasCycle bool

	var visit func(string)
	visit = func(node string) {
		if visited[node] == 1 {
			hasCycle = true
			return
		}
		if visited[node] == 2 {
			return
		}

		visited[node] = 1
		for _, neighbor := range a.graph[node] {
			visit(neighbor)
		}
		visited[node] = 2
	}

	flatElems, _ := model.FlattenElements(a.model)
	for node := range flatElems {
		if visited[node] == 0 {
			visit(node)
		}
	}

	return !hasCycle
}

// hasPath checks if there is a path from src to dst (BFS with limit).
func (a *Analyzer) hasPath(src, dst string) bool {
	if src == dst {
		return true
	}

	visited := make(map[string]bool)
	queue := []string{src}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current] {
			continue
		}
		visited[current] = true

		if current == dst {
			return true
		}

		for _, neighbor := range a.graph[current] {
			if !visited[neighbor] {
				queue = append(queue, neighbor)
			}
		}
	}

	return false
}

// shortestPath finds the shortest path from src to dst (BFS distance).
func (a *Analyzer) shortestPath(src, dst string) int {
	if src == dst {
		return 0
	}

	visited := make(map[string]bool)
	queue := []string{src}
	distances := map[string]int{src: 0}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current] {
			continue
		}
		visited[current] = true

		if current == dst {
			return distances[current]
		}

		for _, neighbor := range a.graph[current] {
			if !visited[neighbor] {
				if _, ok := distances[neighbor]; !ok {
					distances[neighbor] = distances[current] + 1
					queue = append(queue, neighbor)
				}
			}
		}
	}

	return -1 // no path found
}

// longestPathDAG finds the longest path starting from a node (DFS with memoization).
// Only valid for DAGs; cyclic graphs will have max depth 0.
func (a *Analyzer) longestPathDAG(start string) int {
	memo := make(map[string]int)
	return a.dfsLongestPath(start, memo)
}

func (a *Analyzer) dfsLongestPath(node string, memo map[string]int) int {
	if depth, ok := memo[node]; ok {
		return depth
	}

	maxDepth := 0
	for _, neighbor := range a.graph[node] {
		depth := 1 + a.dfsLongestPath(neighbor, memo)
		if depth > maxDepth {
			maxDepth = depth
		}
	}

	memo[node] = maxDepth
	return maxDepth
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
