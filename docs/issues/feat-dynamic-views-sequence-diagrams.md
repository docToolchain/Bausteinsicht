---
title: "feat: Dynamic Views / Sequence Diagrams"
labels: enhancement
---

## Beschreibung

Bausteinsicht currently covers structural architecture (elements, relationships, views). It lacks support for **behavioral architecture** — showing the order and nature of interactions between elements at runtime. This feature adds `dynamicViews` to the model and a new `export-sequence` command.

## Motivation

- Structural diagrams show *what* exists; sequence diagrams show *how* the system behaves
- Architects need both to fully describe a system
- PlantUML and Mermaid are widely used and render in GitHub, Confluence, AsciiDoc tools
- ADR-004 validated that dynamic views are useful but should be explicit (not auto-derived)

## Proposed Implementation

**Model extension:**
```jsonc
{
  "dynamicViews": [
    {
      "key": "checkout-flow",
      "title": "Checkout Flow",
      "steps": [
        { "index": 1, "from": "web-frontend", "to": "api-gateway", "label": "POST /orders", "type": "sync" },
        { "index": 2, "from": "api-gateway", "to": "order-service", "label": "createOrder()", "type": "sync" }
      ]
    }
  ]
}
```

**New command:**
```
bausteinsicht export-sequence [--view <key>] [--format plantuml|mermaid] [--output <dir>]
```

**Step types:** `sync` (→), `async` (->>), `return` (-->)

**Validation:** `from`/`to` must reference existing element IDs; `index` values unique per view.

## Implementation Plan

See [`docs/plans/2026-03-18-dynamic-views-sequence-diagrams.md`](../plans/2026-03-18-dynamic-views-sequence-diagrams.md)

## Affected Components

- `internal/model/types.go`
- `internal/model/validate.go`
- `internal/diagram/sequence.go` (new)
- `cmd/bausteinsicht/export_sequence.go` (new)
