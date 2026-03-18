# Plan: Element Lifecycle Status Tracking

## Purpose

Add a formal `status` field to elements representing where they are in their lifecycle. Status is displayed visually in draw.io (colored badges), enforced via validation rules, and queryable via a new `status` CLI command.

## Lifecycle Values

| Status | Meaning | draw.io Badge Color |
|--------|---------|---------------------|
| `proposed` | Being discussed, not yet designed | yellow `#fff2cc` |
| `design` | Architecture approved, not yet built | blue `#dae8fc` |
| `implementation` | Currently being built | orange `#ffe6cc` |
| `deployed` | Live in production | green `#d5e8d4` |
| `deprecated` | Scheduled for removal, still running | red `#f8cecc` |
| `archived` | Removed, kept for historical reference | grey `#f5f5f5` |

Default (if omitted): no badge rendered (backwards compatible).

## CLI Interface

```
bausteinsicht status [--model <file>] [--filter <status>] [--format text|json]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--filter` | (all) | Show only elements with this status |
| `--format` | `text` | Output format |

### Example Output (text)

```
Element Lifecycle Status
========================

proposed (1):
  notification-service  [service]  "Notification Service"

design (0):

implementation (2):
  payment-v2     [service]   "Payment Service v2"
  audit-log      [storage]   "Audit Log Store"

deployed (4):
  web-frontend   [frontend]  "Web Frontend"
  api-gateway    [service]   "API Gateway"
  order-service  [service]   "Order Service"
  user-db        [database]  "User Database"

deprecated (1):
  legacy-monolith [system]   "Legacy Order System"

archived (0):
```

### Example Output (JSON)

```json
{
  "summary": {
    "proposed": 1, "design": 0, "implementation": 2,
    "deployed": 4, "deprecated": 1, "archived": 0
  },
  "elements": [
    { "id": "notification-service", "kind": "service", "title": "Notification Service", "status": "proposed" }
  ]
}
```

## Model Changes

```jsonc
{
  "model": {
    "elements": [
      {
        "id": "payment-v2",
        "kind": "service",
        "title": "Payment Service v2",
        "status": "implementation"   // new optional field
      }
    ]
  }
}
```

## draw.io Rendering

Status badges are rendered as small overlay labels on element shapes in draw.io:
- Position: top-right corner of the element bounding box
- Format: rounded pill shape with status text
- Color: from the lifecycle color table above
- Sync: badge is regenerated on every forward sync; not reverse-synced from draw.io

Implementation: draw.io supports child cells within a parent cell. The badge is added as a child label cell with absolute position offset.

## Validation Rules

Built-in rules (always active when status field is used):

| Rule | Description |
|------|-------------|
| `no-relationship-from-archived` | Warn if an `archived` element has outgoing relationships |
| `deprecated-has-successor` | Warn if a `deprecated` element has no relationship to a `deployed` element of the same kind |
| `proposed-linked-to-design` | Info: `proposed` elements should have at least one relationship defined |

These are warnings (non-fatal), not errors. Can be suppressed per-element via `"suppress-warnings": [...]`.

## Architecture

### New / Modified Files

| File | Change |
|------|--------|
| `internal/model/types.go` | Add `Status string` field to `Element`; add `ElementStatus` constants |
| `internal/model/validate.go` | Add status value validation; add lifecycle rules |
| `internal/sync/forward.go` | Render status badge as child cell in draw.io |
| `internal/sync/badge.go` | New: badge cell creation helpers |
| `cmd/bausteinsicht/status.go` | New `status` command |

## Testing

- Unit test: `status` command output with mixed-status model
- Unit test: `no-relationship-from-archived` warning fires correctly
- Unit test: badge cell XML generation (check position, color, text)
- E2E test: model with status fields → sync → draw.io file contains badge cells
- Test: omitted status field → no badge rendered, no validation error
