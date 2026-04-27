---
title: "feat: Stale Element Detection"
labels: enhancement
---

## Beschreibung

Detect elements that have not been referenced in git commits for a configurable period and have no lifecycle status, ADR link, or test coverage. These are likely forgotten components — either undocumented active services or candidates for archiving. A `stale` command reports them with risk assessment.

## Motivation

- Large architecture models accumulate elements that nobody touches or questions
- Without explicit status tracking, it's unclear whether an element is live, abandoned, or removed
- "Stale" detection nudges teams to either document (`status: deployed`, `decisions: [...]`) or clean up (`status: archived`)
- Builds on lifecycle status and ADR features; adds no new model fields
- Risk assessment (incoming vs. outgoing relationships) prevents accidental removal of critical components

## Proposed Implementation

**New command:**
```
bausteinsicht stale [--days 90] [--mark-drawio] [--format text|json]
```

**Staleness criteria (ALL must be true):**
1. Model file not modified in git for `--days` days
2. Element has no `status` field
3. Element has no `decisions` field

**Risk levels:**
- Low: no incoming relationships → safe to archive
- Medium: has incoming relationships → other elements depend on it
- High: referenced in view `include` patterns + has incoming relationships

**Output:**
```
legacy-auth  [service]  Last changed: 187 days ago
             ⚠ No status, no ADR — Medium risk (3 incoming relationships)
             Suggestion: set status "deprecated" or link to ADR
```

**`--mark-drawio`:** adds 💤 badge + grey stripe overlay to stale elements in draw.io (non-destructive).

**Configuration in model:**
```jsonc
{ "meta": { "staleDetection": { "thresholdDays": 90, "excludeKinds": ["database"] } } }
```

## Implementation Plan

See the implementation plan embedded in the GitHub issue.

## Affected Components

- `internal/stale/` — new package (detector, git log, risk assessment)
- `cmd/bausteinsicht/stale.go` — new command
- Composes: git integration, lifecycle status, ADR integration
