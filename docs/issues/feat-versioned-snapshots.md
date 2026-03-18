---
title: "feat: Versioned Model Snapshots"
labels: enhancement
---

## Beschreibung

Allow architects to save named snapshots of the model at any point and later diff any two snapshots or compare a snapshot to the current model. Extends the planned As-Is/To-Be feature (two fixed versions) to support N arbitrary snapshots stored in `.bausteinsicht-snapshots/`.

## Motivation

- "Show me the architecture from 6 months ago" is a common request during refactoring retrospectives
- Snapshots complement git history: a snapshot captures the logical architecture state, not just a file diff
- Saved alongside code in git, snapshots provide a permanent architectural changelog
- Reuses diff logic from the planned As-Is/To-Be feature — incremental implementation effort

## Proposed Implementation

**New subcommand group:**
```
bausteinsicht snapshot save    --message "before payment service split"
bausteinsicht snapshot list
bausteinsicht snapshot diff    snapshot-001
bausteinsicht snapshot diff    snapshot-001 --to snapshot-002
bausteinsicht snapshot restore snapshot-001 [--output architecture-restored.jsonc]
bausteinsicht snapshot delete  snapshot-001
```

**Storage:** `.bausteinsicht-snapshots/index.json` + one JSON file per snapshot.

**Diff output:**
```
Snapshot Diff: snapshot-001 → current
Added (2): notification-service, audit-log
Removed (0):
Changed (1): payment-service (title, technology)
Relationships added (2)
```

**Integration with As-Is/To-Be:**
```bash
bausteinsicht snapshot restore snapshot-001 --output-section asIs
# → populates the "asIs" section of architecture.jsonc from the snapshot
```

**.gitignore:** `bausteinsicht init` documents that `.bausteinsicht-snapshots/` can optionally be committed or excluded.

## Implementation Plan

See [`docs/plans/2026-03-18-versioned-snapshots.md`](../plans/2026-03-18-versioned-snapshots.md)

## Affected Components

- `internal/snapshot/` — new package (save, list, diff, restore, delete)
- `cmd/bausteinsicht/snapshot.go` — new command with subcommands
- Reuses `internal/diff/` from As-Is/To-Be feature (if implemented first)
