---
title: "feat: Relationship Graph Analysis"
labels: enhancement
---

## Beschreibung

Ein `analyze` Befehl berechnet Graphmetriken auf dem Architekturmodell: Degree-Zentralität (welche Elemente haben die meisten Abhängigkeiten?), isolierte Elemente, längste Abhängigkeitskette, zirkuläre Abhängigkeiten, und Fan-in/Fan-out Analyse. JSON-Output für CI-Integration und Architektur-Reviews.

## Motivation

- Architekturbewertungen (ATAM, ADR-Reviews) fragen immer: "Was sind die kritischsten Komponenten?"
- Zirkuläre Abhängigkeiten sind schwer manuell zu erkennen — ein automatischer Cycle-Detector spart Stunden
- Fan-in/Fan-out Analyse identifiziert God-Components und Single-Points-of-Failure
- Längste Abhängigkeitskette = maximale Latenz-Kette bei synchronen Calls

## Proposed Implementation

**New command:**
```
bausteinsicht analyze [--metric all|centrality|cycles|chains|isolation|fanout] [--top 5] [--format text|json]
```

**Metrics:**

| Metrik | Algorithmus | Aussage |
|--------|-------------|---------|
| Centrality | Degree-Count | Wer hat die meisten Verbindungen? |
| Cycles | Tarjan's SCC | Zirkuläre Abhängigkeiten |
| Longest chain | Topo-Sort + DP | Kritischer Pfad |
| Isolation | Degree = 0 | Vergessene Elemente |
| Fan-out | Out-Degree | God-Components |

**Output (text):**
```
── Centrality (Top 5) ─────────────────────────────────
  api-gateway    in:3  out:4  total:7  ← highest
  message-broker in:4  out:0  total:4  ← shared infra

── Circular Dependencies ──────────────────────────────
  ✅ No cycles detected

── Longest Chain ──────────────────────────────────────
  4 hops: web-frontend → api-gateway → order-service → order-db

── Isolated Elements ──────────────────────────────────
  ⚠ report-service — no relationships
```

**Algorithms:** Tarjan's SCC für Cycles (O(V+E)), Kahn's Topo-Sort + DP für Longest Path.

## Implementation Plan

See [`docs/plans/2026-03-18-relationship-graph-analysis.md`](../plans/2026-03-18-relationship-graph-analysis.md)

## Affected Components

- `internal/graph/` — new package (graph builder, centrality, cycles, paths)
- `cmd/bausteinsicht/analyze.go` — new command
