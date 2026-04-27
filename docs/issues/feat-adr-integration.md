---
title: "feat: ADR Integration — Link Architecture Decisions to Model Elements"
labels: enhancement
---

## Beschreibung

Link Architecture Decision Records (ADRs) directly to elements and relationships in the model. A new `adr` command shows which decisions affect which components. draw.io renders decision badges on affected elements for immediate traceability.

## Motivation

- ADRs and architecture diagrams currently live in separate files with no machine-readable link between them
- Teams lose track of *why* a component exists or *which decision* drove a technology choice
- Traceability from decision → element is a standard requirement in regulated industries
- The `spec.decisions` section makes ADRs part of the living architecture model

## Proposed Implementation

**Spec extension:**
```jsonc
{
  "spec": {
    "decisions": [
      { "id": "ADR-012", "title": "Event-driven communication", "status": "active", "file": "docs/decisions/ADR-012.md" }
    ]
  }
}
```

**Element/Relationship extension:**
```jsonc
{ "id": "order-service", "decisions": ["ADR-012", "ADR-001"] }
```

**New command:**
```
bausteinsicht adr list [--element <id>] [--format text|json]
bausteinsicht adr show <adr-id>
```

**draw.io:** ⚖ badge on elements with linked decisions (blue = active, grey = superseded).

**Validation:**
- Unknown ADR IDs referenced → error
- Superseded ADR still referenced → warning
- ADR with no element references → warning (orphan)

## Implementation Plan

See the implementation plan embedded in the GitHub issue.

## Affected Components

- `internal/model/types.go` — `DecisionRecord` in `Specification`; `Decisions []string` on `Element`/`Relationship`
- `internal/sync/forward.go` — decision badge rendering
- `internal/model/validate.go` — ADR reference validation
- `cmd/bausteinsicht/adr.go` — new command
