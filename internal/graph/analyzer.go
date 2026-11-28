package graph

import (
	"sort"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// Analyzer performs graph analysis on model relationships.
type Analyzer struct {
	model   *model.BausteinsichtModel
	graph   map[string][]string // element ID → list of outgoing relationship targets
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

// popSCC pops elements from stack until v is reached and returns them as one SCC.
func popSCC(stack *[]string, nodeInfo map[string]*NodeInfo, v string) []string {
	var component []string
	for {
		w := (*stack)[len(*stack)-1]
		*stack = (*stack)[:len(*stack)-1]
		nodeInfo[w].onStack = false
		component = append(component, w)
		if w == v {
			break
		}
	}
	return component
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
			onSCC(popSCC(&stack, nodeInfo, v))
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

// reachability is one breadth-first search from id: the number of other
// elements that can be reached from it and the sum of their shortest distances.
// Betweenness and closeness both follow from it, so the search runs once per
// element, not once per pair of elements (#730).
func (a *Analyzer) reachability(id string) (reachable, totalDistance int) {
	dist := map[string]int{id: 0}
	queue := []string{id}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, next := range a.graph[current] {
			if _, seen := dist[next]; !seen {
				dist[next] = dist[current] + 1
				totalDistance += dist[next]
				reachable++
				queue = append(queue, next)
			}
		}
	}
	return reachable, totalDistance
}

// betweennessScore returns the simplified betweenness: the fraction of the other
// elements that are reachable from id via any path.
func betweennessScore(reachable, elements int) float64 {
	if n := elements - 1; n > 0 {
		return float64(reachable) / float64(n)
	}
	return 0
}

// closenessScore returns the simplified closeness: inverse of one plus the mean
// shortest distance to the reachable elements.
func closenessScore(reachable, totalDistance int) float64 {
	if reachable > 0 {
		return 1.0 / (1.0 + float64(totalDistance)/float64(reachable))
	}
	return 0
}

// calculateCentrality computes centrality metrics for all elements.
func (a *Analyzer) calculateCentrality() []Centrality {
	flatElems, _ := model.FlattenElements(a.model)
	var results []Centrality

	for id := range flatElems {
		reachable, total := a.reachability(id)
		results = append(results, Centrality{
			ID:          id,
			InDegree:    len(a.reverse[id]),
			OutDegree:   len(a.graph[id]),
			Betweenness: betweennessScore(reachable, len(flatElems)),
			Closeness:   closenessScore(reachable, total),
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

	memo := make(map[string]int)
	for start := range flatElems {
		depth := a.dfsLongestPath(start, memo)
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

// dfsLongestPath finds the longest path starting from a node (DFS with
// memoization). Only valid for DAGs; cyclic graphs have max depth 0.
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
