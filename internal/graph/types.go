package graph

// Cycle represents a circular dependency in the relationship graph.
type Cycle struct {
	Elements []string `json:"elements"` // ordered elements forming the cycle
	Length   int      `json:"length"`   // number of elements in cycle
}

// Centrality metrics for an element.
type Centrality struct {
	ID        string  `json:"id"`
	InDegree  int     `json:"inDegree"`  // number of incoming relationships
	OutDegree int     `json:"outDegree"` // number of outgoing relationships
	Betweenness float64 `json:"betweenness"` // how many shortest paths pass through this node
	Closeness float64 `json:"closeness"` // average distance to all other nodes
}

// Component represents a strongly connected component in the graph.
type Component struct {
	ID       int      `json:"id"`
	Elements []string `json:"elements"`
	IsCycle  bool     `json:"isCycle"` // true if it forms a cycle (SCC with len > 1)
}

// GraphAnalysis contains results from relationship graph analysis.
type GraphAnalysis struct {
	ElementCount    int           `json:"elementCount"`
	RelationshipCount int         `json:"relationshipCount"`
	Cycles          []Cycle       `json:"cycles"`
	Centrality      []Centrality  `json:"centrality"`
	Components      []Component   `json:"components"`
	IDAGValid       bool          `json:"idagValid"` // true if graph is acyclic
	MaxDepth        int           `json:"maxDepth"`  // longest dependency path
}

// NodeInfo holds temporary computation data for graph algorithms.
type NodeInfo struct {
	index     int
	lowlink   int
	onStack   bool
	inDegree  int
	outDegree int
}
