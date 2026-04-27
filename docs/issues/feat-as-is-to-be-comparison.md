---
title: "feat: As-Is / To-Be Architecture Comparison"
labels: enhancement
---

## Beschreibung

Architects often need to document both the current state (as-is) and the target state (to-be) of a system architecture. Currently, Bausteinsicht supports only a single model snapshot. This feature adds explicit `asIs` and `toBe` sections to the model and a new `diff` command to visualize changes.

## Motivation

- Teams undergoing migration or modernization need to communicate what changes and what stays
- A `diff` command makes change communication explicit and machine-readable (for CI/CD change reports)
- draw.io overlay views (added/removed/changed with color coding) provide immediate visual clarity for stakeholders

## Proposed Implementation

**Model changes:**
```jsonc
{
  "asIs": { "elements": [...], "relationships": [...] },
  "toBe": { "elements": [...], "relationships": [...] }
}
```

**New command:**
```
bausteinsicht diff [--view <key>] [--format text|json]
```

**draw.io output:** Three new pages per view — `<view>-as-is`, `<view>-to-be`, `<view>-overlay`

**Color coding in overlay:**
- Added elements → green (`#d5e8d4`)
- Removed elements → red (`#f8cecc`)
- Changed elements → orange (`#ffe6cc`)

**Backwards compatible:** Models without `asIs`/`toBe` are unaffected.

## Implementation Plan

See the implementation plan embedded in the GitHub issue.

## Affected Components

- `internal/model/types.go`
- `internal/diff/` (new package)
- `internal/sync/forward.go`
- `cmd/bausteinsicht/diff.go` (new)
