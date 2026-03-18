# Plan: ADR Integration

## Purpose

Link Architecture Decision Records (ADRs) directly to model elements and relationships. A new `adr` command shows which decisions affect which components, and draw.io displays ADR badges on affected elements. Provides traceability between architectural decisions and the components they shape.

## CLI Interface

```
bausteinsicht adr list   [--model <file>] [--element <id>] [--format text|json]
bausteinsicht adr show   <adr-id> [--model <file>]
```

| Flag | Description |
|------|-------------|
| `--element <id>` | Show only ADRs affecting a specific element |
| `--format` | `text` (default) or `json` |

### Example Output

```
Architecture Decision Records
=============================

ADR-001  "Use Go for HAL layer"           [active]    affects: api-gateway, order-service (2 elements)
ADR-004  "Reject auto-derived dynamic views" [active] affects: (no elements linked)
ADR-007  "Migrate from monolith to services" [superseded by ADR-012]
                                              affects: legacy-monolith (deprecated)
ADR-012  "Event-driven communication"      [active]    affects: order-service, payment-service, message-broker

bausteinsicht adr show ADR-012
  Title:    Event-driven communication
  Status:   active
  Date:     2026-01-15
  File:     docs/decisions/ADR-012.md
  Affects:  order-service, payment-service, message-broker
  Decides:  All inter-service communication uses async events via message broker
```

## Model Changes

### ADR References on Elements and Relationships

```jsonc
{
  "model": {
    "elements": [
      {
        "id": "order-service",
        "kind": "service",
        "title": "Order Service",
        "decisions": ["ADR-012", "ADR-001"]   // new optional field
      }
    ],
    "relationships": [
      {
        "id": "r1",
        "from": "order-service",
        "to": "message-broker",
        "label": "publishes",
        "decisions": ["ADR-012"]   // new optional field
      }
    ]
  }
}
```

### ADR Definitions in Spec

```jsonc
{
  "spec": {
    "decisions": [
      {
        "id": "ADR-001",
        "title": "Use Go for HAL layer",
        "status": "active",
        "date": "2026-01-10",
        "file": "docs/decisions/ADR-001.md"   // optional: path to ADR file in repo
      },
      {
        "id": "ADR-007",
        "title": "Migrate from monolith to services",
        "status": "superseded",
        "supersededBy": "ADR-012"
      }
    ]
  }
}
```

ADR status values: `"proposed"`, `"active"`, `"deprecated"`, `"superseded"`.

## draw.io Rendering

Elements with linked decisions show a small decision badge:
- Icon: ⚖ (scales) symbol in bottom-left corner
- Tooltip: comma-separated list of ADR IDs
- Superseded ADRs render with grey badge; active ADRs render with blue badge

Implementation: draw.io child cell at bottom-left position, similar to lifecycle status badge.

## Validation

- `decisions` array on elements/relationships must reference IDs defined in `spec.decisions`
- Warn if an ADR with status `"superseded"` is still referenced (use the superseding ADR instead)
- Warn if a decision has no elements or relationships referencing it (`orphan ADR`)

## Architecture

### New / Modified Files

| File | Change |
|------|--------|
| `internal/model/types.go` | Add `Decisions []string` to `Element` and `Relationship`; add `DecisionRecord` struct and `Decisions []DecisionRecord` to `Specification` |
| `internal/model/validate.go` | Validate decision ID references; orphan ADR warnings |
| `internal/sync/forward.go` | Render decision badges on elements |
| `cmd/bausteinsicht/adr.go` | New `adr` command with `list` and `show` subcommands |

## Testing

- Unit test: `adr list` output with mixed-status ADRs
- Unit test: `adr list --element order-service` filters correctly
- Unit test: superseded ADR warning fires when still referenced
- Unit test: decision badge XML generation
- E2E test: model with decisions → sync → draw.io contains badge cells on affected elements
