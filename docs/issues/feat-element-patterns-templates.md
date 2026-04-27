---
title: "feat: Element Patterns / Topology Templates"
labels: enhancement
---

## Beschreibung

Allow teams to define reusable multi-element topology patterns in `spec.patterns`. A single `add-from-pattern` command expands a pattern into the model — e.g. a "microservice" pattern creates a service, database, and cache with standard relationships in one command.

## Motivation

- Teams repeatedly define the same element topologies (microservice = service + db + cache) — this is tedious and error-prone
- Patterns make architectural standards explicit and enforceable
- Highly LLM-friendly: an AI agent can apply patterns without knowing the full JSON schema
- `bausteinsicht add-from-pattern microservice --id order` is faster and less error-prone than 3 `add element` + 2 `add relationship` calls

## Proposed Implementation

**Spec definition:**
```jsonc
{
  "spec": {
    "patterns": {
      "microservice": {
        "description": "Service with database and cache",
        "elements": [
          { "id": "{base}-service", "kind": "service",   "title": "{Title} Service" },
          { "id": "{base}-db",      "kind": "database",  "title": "{Title} Database" },
          { "id": "{base}-cache",   "kind": "cache",     "title": "{Title} Cache" }
        ],
        "relationships": [
          { "id": "r-{base}-svc-db",    "from": "{base}-service", "to": "{base}-db",    "label": "reads/writes" },
          { "id": "r-{base}-svc-cache", "from": "{base}-service", "to": "{base}-cache", "label": "caches" }
        ]
      }
    }
  }
}
```

**New command:**
```
bausteinsicht add-from-pattern microservice --id order --title "Order"
# → creates order-service, order-db, order-cache + 2 relationships

bausteinsicht add-from-pattern --list
# → lists all available patterns with element/relationship counts
```

**Variables:** `{base}` = `--id`, `{Title}` = `--title` (title-cased), `{BASE}` = uppercased id.

**Conflict detection:** warns if generated IDs already exist; `--prefix` adds a namespace.

## Implementation Plan

See [`docs/plans/2026-03-18-element-patterns-templates.md`](../plans/2026-03-18-element-patterns-templates.md)

## Affected Components

- `internal/model/types.go` — `PatternDefinition` in `Specification`
- `internal/model/patterns.go` — `ExpandPattern()` (new)
- `internal/model/validate.go` — validate pattern definitions at load
- `cmd/bausteinsicht/add_from_pattern.go` — new command
