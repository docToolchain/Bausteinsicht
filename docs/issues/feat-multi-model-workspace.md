---
title: "feat: Multi-Model Workspace — Merge Team Models into Platform View"
labels: enhancement
---

## Beschreibung

Allow merging multiple `architecture.jsonc` files (one per team, bounded context, or domain) into a unified workspace view. Each team owns their local model; a `workspace.jsonc` config references all models and generates cross-team views showing inter-domain dependencies.

## Motivation

- Large organizations have multiple teams, each with their own architecture model
- There is currently no way to generate a platform-level view spanning multiple models
- Teams should not need to copy elements from other teams' models into their own
- Cross-model relationships (e.g. Orders → Payments) need a central place to be defined

## Proposed Implementation

**Workspace config:**
```jsonc
// workspace.jsonc
{
  "workspace": {
    "name": "E-Commerce Platform",
    "models": [
      { "id": "orders",   "path": "teams/orders/architecture.jsonc",   "prefix": "orders" },
      { "id": "payments", "path": "teams/payments/architecture.jsonc", "prefix": "payments" }
    ]
  },
  "crossModelRelationships": [
    { "id": "xr-001", "from": "orders/order-service", "to": "payments/payment-service", "label": "charges via" }
  ],
  "views": [
    { "key": "platform", "title": "Platform Overview", "include-from": ["orders", "payments"], "include-kinds": ["system", "service"] }
  ]
}
```

**New commands:**
```
bausteinsicht workspace init
bausteinsicht workspace merge    [--workspace workspace.jsonc]
bausteinsicht workspace validate [--workspace workspace.jsonc]
bausteinsicht workspace list     [--workspace workspace.jsonc]
```

**ID namespacing:** elements are prefixed with model `id` → `orders/order-service`, `payments/payment-service` — no ID collisions.

**Validation:** missing model paths, conflicting kind definitions, unresolved cross-model references.

## Implementation Plan

See the implementation plan embedded in the GitHub issue.

## Affected Components

- `internal/workspace/` — new package (loader, merge, validate)
- `cmd/bausteinsicht/workspace.go` — new command with subcommands
