# Plan: Relationship Cardinality & Data Flow Annotations

## Purpose

Extend the Relationship struct with optional fields for cardinality (`1:N`), data flow type (`event-streaming`, `request-response`, ...), and protocol (`gRPC`, `HTTP`, ...). These are displayed on draw.io connectors as labels and affect arrow styling. All fields are optional — no breaking changes.

## Model Changes

```jsonc
{
  "model": {
    "relationships": [
      {
        "id": "r1",
        "from": "web-frontend",
        "to": "api-gateway",
        "label": "REST calls",
        "cardinality": "N:1",
        "dataFlow": "request-response",
        "protocol": "HTTPS"
      },
      {
        "id": "r2",
        "from": "order-service",
        "to": "message-broker",
        "label": "publishes",
        "cardinality": "1:N",
        "dataFlow": "event-streaming",
        "protocol": "AMQP"
      },
      {
        "id": "r3",
        "from": "report-job",
        "to": "order-db",
        "label": "reads",
        "dataFlow": "batch"
      }
    ]
  }
}
```

## Field Definitions

### `cardinality`

Free-form string. Common values: `"1:1"`, `"1:N"`, `"N:1"`, `"N:N"`. Displayed as a small label at the midpoint of the connector in draw.io.

### `dataFlow`

| Value | draw.io Arrow Style | Description |
|-------|---------------------|-------------|
| `request-response` | solid arrow (→) | Synchronous HTTP/RPC call |
| `event-streaming` | dashed arrow (- ->) | Async event/message |
| `batch` | dotted arrow (···>) | Scheduled batch transfer |
| `file-transfer` | solid with document icon | File-based data exchange |
| `database-read` | arrow with cylinder | Query to data store |

Default (omitted): solid arrow, current behavior unchanged.

### `protocol`

Free-form string. Common values: `"HTTP"`, `"HTTPS"`, `"gRPC"`, `"AMQP"`, `"MQTT"`, `"WebSocket"`, `"TCP"`. Displayed as a tooltip on the connector. Rendered in PlantUML/Mermaid C4 macros as the technology parameter.

## draw.io Connector Styling

Each `dataFlow` value maps to a draw.io edge style string:

```go
var DataFlowStyles = map[string]string{
    "request-response": "edgeStyle=orthogonalEdgeStyle;dashed=0;",
    "event-streaming":  "edgeStyle=orthogonalEdgeStyle;dashed=1;dashPattern=8 4;",
    "batch":            "edgeStyle=orthogonalEdgeStyle;dashed=1;dashPattern=2 2;",
    "file-transfer":    "edgeStyle=orthogonalEdgeStyle;dashed=0;endArrow=open;",
    "database-read":    "edgeStyle=orthogonalEdgeStyle;dashed=0;endArrow=ERmany;",
}
```

Cardinality renders as a draw.io edge label at position `exitX=0.5;exitY=0.5` (midpoint).

## PlantUML / Mermaid C4 Export

Extended PlantUML C4 relationship macro:

```plantuml
' Before (current):
Rel(web_frontend, api_gateway, "REST calls")

' After (with new fields):
Rel(web_frontend, api_gateway, "REST calls", "HTTPS")
' Cardinality added as part of the label:
Rel(order_service, message_broker, "publishes [1:N]", "AMQP")
```

Mermaid does not have a C4 cardinality macro — cardinality is appended to the label.

## `add relationship` CLI Extension

```
bausteinsicht add relationship

From element ID : order-service
To element ID   : message-broker
Label           : publishes
Cardinality     : 1:N  (optional, press Enter to skip)
Data Flow       : event-streaming  (optional: request-response/event-streaming/batch/file-transfer/database-read)
Protocol        : AMQP  (optional, press Enter to skip)
```

## Architecture

### New / Modified Files

| File | Change |
|------|--------|
| `internal/model/types.go` | Add `Cardinality`, `DataFlow`, `Protocol string` to `Relationship` |
| `internal/model/validate.go` | Validate `dataFlow` is one of the known values (warn on unknown, don't error) |
| `internal/sync/forward.go` | Apply data-flow-based edge styles; add cardinality mid-label |
| `internal/diagram/diagram.go` | Extend PlantUML/Mermaid rendering with protocol and cardinality |
| `cmd/bausteinsicht/add_relationship.go` | Add optional prompts for new fields |

## Backwards Compatibility

- All new fields are optional with zero-value semantics
- Omitted `dataFlow` → current solid arrow style (unchanged)
- Omitted `cardinality` → no mid-label rendered
- Existing models work without modification

## Testing

- Unit test: `DataFlowStyles` map covers all defined values
- Unit test: cardinality mid-label XML generation
- Unit test: PlantUML export with/without cardinality and protocol
- E2E test: relationship with `dataFlow: "event-streaming"` → sync → connector is dashed in draw.io
