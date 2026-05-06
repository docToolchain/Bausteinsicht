# Implementation Plan: Versioned Model Snapshots (#290)

## Overview
Save named snapshots of architecture models at any point in time and compare any two snapshots or a snapshot to the current state. Supports N arbitrary snapshots with timestamps and descriptive messages, stored in `.bausteinsicht-snapshots/`.

## Phases

### Phase 1: Storage & Data Types
Create snapshot storage infrastructure and core types:
- `internal/snapshot/types.go` — Snapshot, SnapshotIndex, SnapshotMetadata types
- `internal/snapshot/storage.go` — Load/save snapshots from .bausteinsicht-snapshots/

### Phase 2: Snapshot Commands (Save, List, Delete)
Implement basic snapshot management:
- `cmd/bausteinsicht/snapshot.go` — Root snapshot command
- `cmd/bausteinsicht/snapshot_save.go` — Save current model as snapshot
- `cmd/bausteinsicht/snapshot_list.go` — List all saved snapshots
- `cmd/bausteinsicht/snapshot_delete.go` — Delete a snapshot

### Phase 3: Diff & Restore
Add comparison and restoration capabilities:
- `cmd/bausteinsicht/snapshot_diff.go` — Diff snapshot vs current or two snapshots
- `cmd/bausteinsicht/snapshot_restore.go` — Restore snapshot to file
- Reuse diff logic from Issue #281 (As-Is/To-Be)

### Phase 4: CLI Integration & Testing
Finalize and test:
- Register snapshot commands in root
- Comprehensive unit tests for all operations
- E2E tests with real snapshot workflows

## Acceptance Criteria
- ✅ CLI commands: save, list, delete, diff, restore
- ✅ Snapshots stored in `.bausteinsicht-snapshots/index.json` + per-snapshot JSON files
- ✅ Timestamps and messages on each snapshot
- ✅ Diff output (text and JSON formats)
- ✅ Restore to file
- ✅ All tests pass
- ✅ Backward compatible

## Key Dependencies
- Requires Issue #281 (As-Is/To-Be Architecture Comparison) for diff logic
- Reuses model loading/validation from existing code
