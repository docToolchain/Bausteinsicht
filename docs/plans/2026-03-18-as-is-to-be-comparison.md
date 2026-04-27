# Plan: As-Is / To-Be Architecture Comparison

## Purpose

Allow modeling current (as-is) and target (to-be) architectures in the same model file with automated change detection and visual diff output in draw.io.

## CLI Interface

```
bausteinsicht diff [--model <file>] [--view <key>] [--format text|json]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--model` | `architecture.jsonc` | Model file path |
| `--view` | (all) | Show diff for one view only |
| `--format` | `text` | Output format: `text` or `json` |

The `sync` and `watch` commands are extended to render both model versions into draw.io:
- draw.io pages: `<view-key>-as-is`, `<view-key>-to-be`, `<view-key>-overlay`
- Overlay page: added elements green, removed elements red, unchanged grey

## Model Changes

```jsonc
{
  "spec": { ... },
  "model": {
    // existing current-state model
  },
  "asIs": {
    // snapshot of current architecture (same schema as "model")
    "elements": [...],
    "relationships": [...]
  },
  "toBe": {
    // target architecture (same schema as "model")
    "elements": [...],
    "relationships": [...]
  },
  "views": [...]
}
```

When `asIs`/`toBe` are absent, existing behaviour is unchanged.

## Diff Algorithm

1. Collect element IDs from `asIs.elements` and `toBe.elements`
2. Compute three sets:
   - `added` = in `toBe` but not in `asIs`
   - `removed` = in `asIs` but not in `toBe`
   - `changed` = in both, but title/description/kind/technology differ
3. Render overlay view:
   - `added` → green fill (`#d5e8d4`, stroke `#82b366`)
   - `removed` → red fill (`#f8cecc`, stroke `#b85450`) with strikethrough label
   - `changed` → orange fill (`#ffe6cc`, stroke `#d6b656`)
   - `unchanged` → default style (grey)

## Architecture

### New / Modified Files

| File | Change |
|------|--------|
| `internal/model/types.go` | Add `AsIsSection`, `ToBeSection` fields to `BausteinsichtModel` |
| `internal/model/validate.go` | Validate `asIs`/`toBe` consistency (element kind references) |
| `internal/diff/diff.go` | New package: `ComputeDiff(asIs, toBe) DiffResult` |
| `internal/diff/types.go` | `DiffResult`, `ElementChange`, `ChangeType` (added/removed/changed) |
| `internal/sync/forward.go` | Extend to render overlay pages when `asIs`/`toBe` present |
| `cmd/bausteinsicht/diff.go` | New `diff` command using `diff.ComputeDiff` + text/json printer |

### Data Types

```go
type DiffResult struct {
    Added   []ElementChange
    Removed []ElementChange
    Changed []ElementChange
}

type ElementChange struct {
    ID       string
    Kind     string
    Title    string
    OldValue map[string]string // for "changed"
    NewValue map[string]string
}
```

## Sync Behaviour

- `asIs` and `toBe` sections are **not** reverse-synced from draw.io (model wins)
- Overlay pages are regenerated on every forward sync
- As-is/to-be pages use the same layout engine as regular views

## Output (text format)

```
Architecture Diff
=================
Added (3):
  + payment-service  [service]  "New Payment Microservice"
  + message-broker   [infra]    "Kafka Cluster"
  + audit-log        [storage]  "Compliance Audit Store"

Removed (1):
  - legacy-monolith  [system]   "Legacy Order System"

Changed (2):
  ~ api-gateway      title: "API Gateway" → "Edge Gateway"
  ~ user-db          technology: "PostgreSQL 13" → "PostgreSQL 16"
```

## Testing

- Unit tests for `diff.ComputeDiff` with table-driven cases (add/remove/change/no-change)
- E2E test: model with `asIs`/`toBe` → `diff` → verify JSON output matches expected changes
- Property-based test: ComputeDiff is symmetric (swap asIs/toBe inverts add/remove)

## Migration

Fully backwards-compatible. Existing models without `asIs`/`toBe` are unaffected.
