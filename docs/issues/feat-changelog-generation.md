---
title: "feat: Architecture Changelog Generation"
labels: enhancement
---

## Beschreibung

Automatically generate a human-readable architecture changelog by comparing two model versions from git history or saved snapshots. Outputs Markdown, AsciiDoc, or JSON describing which elements and relationships were added, removed, or changed between two points in time.

## Motivation

- "What changed architecturally in this release?" is asked at every sprint review and architecture board
- Currently, the answer requires manually diffing JSON files or reading git history
- A generated changelog bridges architecture documentation and release notes automatically
- AsciiDoc output integrates directly with docToolchain for architecture documentation sites
- JSON output enables automated PR comments with architecture change summaries in CI

## Proposed Implementation

**New command:**
```
bausteinsicht changelog --since v1.0 --format markdown --output ARCHITECTURE-CHANGELOG.md
bausteinsicht changelog --since snapshot-001 --until snapshot-005
bausteinsicht changelog --since "30 days ago"
```

**Two modes:**
1. **Git-based** (default): uses `git show <ref>:architecture.jsonc` — no additional setup required
2. **Snapshot-based**: uses `.bausteinsicht-snapshots/` from the Versioned Snapshots feature

**Markdown output example:**
```markdown
## v1.0 → v2.0

### Added (3 elements)
- **notification-service** `[service]` — Notification Service

### Removed (1 element)
- ~~**legacy-monolith**~~ `[system]` — Legacy Order System

### Changed (1 element)
- **payment-service** — technology: "Java" → "Go"
```

**CI/CD integration:** Post architecture changes as PR comment automatically.

## Implementation Plan

See the implementation plan embedded in the GitHub issue.

## Affected Components

- `internal/changelog/` — new package (git loader, generator, renderers)
- `cmd/bausteinsicht/changelog.go` — new command
- Reuses `internal/diff/` types from As-Is/To-Be or Snapshot feature
