# Plan: Versioned Model Snapshots

## Purpose

Allow architects to save named snapshots of the model at any point in time and later compare any two snapshots or compare a snapshot to the current model. Goes beyond the planned As-Is/To-Be feature (two fixed versions) by supporting N arbitrary snapshots with timestamps and messages.

## CLI Interface

```
bausteinsicht snapshot save    [--message <msg>] [--model <file>]
bausteinsicht snapshot list    [--model <file>] [--format text|json]
bausteinsicht snapshot diff    <snapshot-id> [--to <snapshot-id>] [--format text|json]
bausteinsicht snapshot restore <snapshot-id> [--output <file>]
bausteinsicht snapshot delete  <snapshot-id>
```

### Examples

```bash
# Save current state before refactoring
bausteinsicht snapshot save --message "before payment service split"
# → Snapshot saved: 2026-03-18T14-32-00Z (snapshot-001)

# List all snapshots
bausteinsicht snapshot list
# → snapshot-001  2026-03-18T14:32  "before payment service split"  (7 elements, 5 relationships)
# → snapshot-002  2026-03-18T16:10  "after extracting notification"  (9 elements, 7 relationships)

# Diff snapshot against current model
bausteinsicht snapshot diff snapshot-001
# → Shows what changed since snapshot-001

# Diff two snapshots against each other
bausteinsicht snapshot diff snapshot-001 --to snapshot-002
```

## Storage

Snapshots are stored in `.bausteinsicht-snapshots/` at the repository root:

```
.bausteinsicht-snapshots/
├── index.json              ← index of all snapshots
├── snapshot-001.json       ← full model snapshot
└── snapshot-002.json
```

### index.json

```json
[
  {
    "id": "snapshot-001",
    "timestamp": "2026-03-18T14:32:00Z",
    "message": "before payment service split",
    "modelPath": "architecture.jsonc",
    "elementCount": 7,
    "relationshipCount": 5
  }
]
```

### Snapshot File

A snapshot is a complete copy of the model at the time of saving — same JSON schema as `architecture.jsonc`, plus a metadata header:

```json
{
  "_snapshot": {
    "id": "snapshot-001",
    "timestamp": "2026-03-18T14:32:00Z",
    "message": "before payment service split",
    "modelPath": "architecture.jsonc"
  },
  "spec": { ... },
  "model": { ... },
  "views": [...]
}
```

## Diff Output

```
Snapshot Diff: snapshot-001 → current
======================================
Snapshot: "before payment service split" (2026-03-18T14:32)
Current:  architecture.jsonc

Added (2):
  + notification-service  [service]   "Notification Service"
  + audit-log             [storage]   "Audit Log Store"

Removed (0):

Changed (1):
  ~ payment-service  title: "Payment Service" → "Payment Service v2"
                     technology: "Java" → "Go"

Relationships added (2):
  + order-service → notification-service  "OrderConfirmed"
  + api-gateway → audit-log               "logs"

Total: 2 added, 0 removed, 1 changed
```

## Architecture

### New / Modified Files

| File | Change |
|------|--------|
| `internal/snapshot/snapshot.go` | New: `Save`, `List`, `Delete`, `Load` operations |
| `internal/snapshot/types.go` | New: `Snapshot`, `SnapshotIndex`, `SnapshotMeta` types |
| `internal/snapshot/diff.go` | New: `DiffSnapshots(a, b Model) DiffResult` (reuses or adapts logic from planned `internal/diff/` package) |
| `cmd/bausteinsicht/snapshot.go` | New `snapshot` command with subcommands |

### Snapshot ID Generation

IDs are auto-incremented (`snapshot-001`, `snapshot-002`, ...) based on the index. Custom IDs are not supported to avoid conflicts.

### `.gitignore` Recommendation

`bausteinsicht init` adds the following comment to `.gitignore` template:

```
# Uncomment to exclude architecture snapshots from git:
# .bausteinsicht-snapshots/
```

Teams can choose to commit snapshots (for shared history) or exclude them (for local-only use).

## Integration with As-Is/To-Be (Planned Feature)

If the As-Is/To-Be feature is implemented, snapshots can populate `asIs`/`toBe`:

```bash
bausteinsicht snapshot restore snapshot-001 --output-section asIs
# → writes snapshot-001 model into the "asIs" section of architecture.jsonc
```

## Testing

- Unit test: `Save` creates snapshot file + updates index
- Unit test: `List` returns snapshots in chronological order
- Unit test: `DiffSnapshots` with known before/after models
- Unit test: `Delete` removes file and index entry
- E2E test: save → modify model → diff → verify changes match
- Test: `restore` writes snapshot back to model file correctly
