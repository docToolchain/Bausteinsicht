---
title: "feat: Element Lifecycle Status Tracking"
labels: enhancement
---

## Beschreibung

Add a formal `status` field to elements representing where they are in their lifecycle. Status is rendered as a colored badge in draw.io, triggers built-in validation warnings, and is queryable via a new `status` CLI command.

## Motivation

- Teams need to communicate which components are live, under construction, deprecated, or removed
- Currently, lifecycle state is either undocumented or stored as free-text in descriptions
- A machine-readable `status` field enables CI checks ("no new relationships to deprecated components")
- Bridges architecture documentation and project planning without external tool dependency

## Proposed Implementation

**Element field:**
```jsonc
{ "id": "payment-v2", "kind": "service", "title": "Payment Service v2", "status": "implementation" }
```

**Lifecycle values:** `proposed` → `design` → `implementation` → `deployed` → `deprecated` → `archived`

**draw.io:** colored badge (pill) rendered in top-right corner of each element with a status value.

**New command:**
```
bausteinsicht status [--filter deployed] [--format text|json]
```

**Built-in warnings:**
- `no-relationship-from-archived` — archived elements with outgoing relationships
- `deprecated-has-successor` — deprecated element with no deployed successor

**Backwards compatible:** `status` is optional; elements without it render as today.

## Implementation Plan

See [`docs/plans/2026-03-18-element-lifecycle-status.md`](../plans/2026-03-18-element-lifecycle-status.md)

## Affected Components

- `internal/model/types.go` — `Status string` on `Element`
- `internal/sync/forward.go` + `internal/sync/badge.go` — badge rendering
- `internal/model/validate.go` — lifecycle validation rules
- `cmd/bausteinsicht/status.go` — new command
