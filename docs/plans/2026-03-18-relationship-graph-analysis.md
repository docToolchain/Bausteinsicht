# Plan: Relationship Graph Analysis

## Purpose

Compute graph-theoretic metrics on the architecture model: degree centrality (which elements have the most dependencies?), isolated elements, longest dependency chain, circular dependency detection, and fan-in/fan-out ratios. Outputs a structured report useful for architecture reviews and refactoring prioritization.

## CLI Interface

```
bausteinsicht analyze [--model <file>] [--metric all|centrality|cycles|chains|isolation|fanout]
                      [--format text|json] [--top <n>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--metric` | `all` | Which analysis to run |
| `--top` | `5` | Show top N results per metric |
| `--format` | `text` | Output format |

## Metrics

### 1. Degree Centrality

**In-degree** (fan-in): how many other elements depend on this element.
**Out-degree** (fan-out): how many elements this element depends on.
**Total degree**: in + out.

High fan-in → critical shared component (single point of failure risk).
High fan-out → god element (too many responsibilities).

### 2. Circular Dependency Detection

Finds all cycles in the directed relationship graph using DFS with back-edge detection.

```
⚠ Circular dependency detected (3 cycles):
  Cycle 1: order-service → payment-service → fraud-check → order-service
  Cycle 2: auth-service → user-service → auth-service
```

### 3. Longest Dependency Chain

Finds the longest path (by hop count) in the DAG (after cycle removal).

```
Longest chain (7 hops):
  user → web-frontend → api-gateway → order-service → payment-service
       → payment-gateway → bank-api → confirmation
```

### 4. Isolated Elements

Elements with no relationships (in or out). May indicate:
- Forgotten elements (candidates for stale detection)
- Elements that should be connected but aren't documented

### 5. Fan-In / Fan-Out Ratio

Elements with high fan-in and low fan-out are likely core infrastructure (shared services, databases).
Elements with high fan-out and low fan-in are likely orchestrators (API gateways, BFFs).

## Output (text format)

```
Architecture Graph Analysis
===========================
Total: 12 elements, 15 relationships

── Centrality (Top 5) ─────────────────────────────────────────────────
  api-gateway      in: 3  out: 4  total: 7  ← highest total
  order-service    in: 2  out: 3  total: 5
  message-broker   in: 4  out: 0  total: 4  ← high fan-in (shared infra)
  user-db          in: 3  out: 0  total: 3
  payment-service  in: 1  out: 2  total: 3

── Circular Dependencies ──────────────────────────────────────────────
  ✅ No cycles detected

── Longest Chain ──────────────────────────────────────────────────────
  4 hops: web-frontend → api-gateway → order-service → order-db

── Isolated Elements ──────────────────────────────────────────────────
  ⚠ report-service — no relationships (consider: stale or missing links?)

── Fan-In / Fan-Out Analysis ──────────────────────────────────────────
  High fan-in (shared):      message-broker (4), user-db (3), api-gateway (3)
  High fan-out (orchestrator): api-gateway (4), order-service (3)
  Balanced:                  payment-service (1:2), auth-service (2:1)
```

## Output (JSON format)

```json
{
  "summary": { "elements": 12, "relationships": 15, "cycles": 0, "isolated": 1 },
  "centrality": [
    { "id": "api-gateway", "in": 3, "out": 4, "total": 7 }
  ],
  "cycles": [],
  "longestChain": {
    "length": 4,
    "path": ["web-frontend", "api-gateway", "order-service", "order-db"]
  },
  "isolated": ["report-service"],
  "fanAnalysis": {
    "highFanIn":  ["message-broker", "user-db"],
    "highFanOut": ["api-gateway", "order-service"]
  }
}
```

## Algorithm Details

### Cycle Detection (Tarjan's SCC)

Uses Tarjan's Strongly Connected Components algorithm — O(V+E) — to find all cycles.

```go
func FindCycles(elements []Element, relationships []Relationship) [][]string {
    // Tarjan's SCC: returns each SCC with size > 1 as a cycle
}
```

### Longest Path (DAG)

Applied after cycle contraction. Uses topological sort + dynamic programming.

```go
func LongestPath(dag Graph) []string {
    // Kahn's algorithm for topo sort + DP for longest path
}
```

## Architecture

### New / Modified Files

| File | Change |
|------|--------|
| `internal/graph/graph.go` | New: `BuildGraph(model) Graph` |
| `internal/graph/centrality.go` | New: `ComputeCentrality(graph) []CentralityResult` |
| `internal/graph/cycles.go` | New: `FindCycles(graph) []Cycle` (Tarjan's SCC) |
| `internal/graph/paths.go` | New: `LongestPath(graph) Path` |
| `internal/graph/types.go` | New: `Graph`, `Node`, `Edge`, analysis result types |
| `cmd/bausteinsicht/analyze.go` | New `analyze` command |

## Testing

- Unit test: centrality on known graph (star topology → center has highest degree)
- Unit test: cycle detection finds A→B→C→A but not A→B→C
- Unit test: longest path on chain A→B→C→D = 3 hops
- Unit test: isolated element detection
- Property-based test: centrality total = sum of all in/out degrees / 2
- E2E test: `analyze --metric cycles --format json` on model with known cycle
