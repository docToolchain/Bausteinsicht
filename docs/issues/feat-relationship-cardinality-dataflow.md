---
title: "feat: Relationship Cardinality & Data Flow Annotations"
labels: enhancement
---

## Beschreibung

Extend the Relationship struct with three optional fields: `cardinality` (1:N), `dataFlow` (request-response / event-streaming / batch), and `protocol` (gRPC, AMQP, …). These affect draw.io connector styling (dashed for async, dotted for batch) and enrich PlantUML/Mermaid C4 exports with technology parameters.

## Motivation

- Structural diagrams currently cannot distinguish between synchronous HTTP calls and async event streams — both render as identical solid arrows
- Teams need to document data flow types for integration architecture reviews
- Cardinality is frequently asked for in architecture reviews and currently has no representation
- All fields are optional → zero breaking changes for existing models

## Proposed Implementation

**Model extension (all fields optional):**
```jsonc
{
  "from": "order-service", "to": "message-broker",
  "label": "publishes",
  "cardinality": "1:N",
  "dataFlow": "event-streaming",
  "protocol": "AMQP"
}
```

**draw.io connector styles:**
| dataFlow | Style |
|----------|-------|
| `request-response` | solid arrow |
| `event-streaming` | dashed arrow |
| `batch` | dotted arrow |

**PlantUML C4:** `Rel(order_service, broker, "publishes [1:N]", "AMQP")`

**`add relationship` prompts** extended with optional cardinality/dataFlow/protocol fields.

## Implementation Plan

See [`docs/plans/2026-03-18-relationship-cardinality-dataflow.md`](../plans/2026-03-18-relationship-cardinality-dataflow.md)

## Affected Components

- `internal/model/types.go` — 3 new optional fields on `Relationship`
- `internal/sync/forward.go` — data-flow-based edge style + cardinality label
- `internal/diagram/diagram.go` — PlantUML/Mermaid export
- `cmd/bausteinsicht/add_relationship.go` — optional prompts
