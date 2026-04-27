---
title: "feat: Metric Overlays — Heatmap on Architecture Diagram"
labels: enhancement
---

## Beschreibung

Load external metrics (error rate, response time, test coverage, deployment frequency) from a JSON file and overlay them as a heatmap on draw.io elements. Elements are colored green→yellow→orange→red based on normalized metric values. A `remove` command restores original styling.

## Motivation

- Architects need to correlate runtime problems with architecture — "which components are most at risk?"
- Currently, draw.io diagrams are purely structural; operational context requires a separate tool
- A heatmap overlay bridges architecture and observability in a single diagram
- Works with any monitoring source (Prometheus, Datadog, custom CI reports) via a simple JSON intermediary format
- Non-destructive: original styles are preserved and restorable

## Proposed Implementation

**Metrics file (team-generated, any source):**
```json
{
  "metrics": [
    { "elementId": "order-service",   "error_rate": 4.2, "coverage": 78 },
    { "elementId": "legacy-monolith", "error_rate": 12.5, "coverage": 23 }
  ]
}
```

**New command:**
```
bausteinsicht overlay apply metrics.json --metric error_rate
bausteinsicht overlay remove
bausteinsicht overlay list metrics.json
```

**Color scale:** normalized across all elements:
- 0–25% → green, 25–50% → yellow, 50–75% → orange, 75–100% → red
- Direction auto-detected: `error_rate` → higher=worse; `coverage` → higher=better

**Non-destructive:** original fill colors saved in draw.io metadata; restored with `overlay remove`.

**CI/CD integration:** generate `metrics.json` from Prometheus/Datadog + `overlay apply` in report job.

## Implementation Plan

See the implementation plan embedded in the GitHub issue.

## Affected Components

- `internal/overlay/` — new package (normalize, apply, remove)
- `cmd/bausteinsicht/overlay.go` — new command
