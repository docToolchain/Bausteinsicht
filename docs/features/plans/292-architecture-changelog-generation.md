# Implementation Plan: Architecture Changelog Generation (#292)

## Overview
Generate human-readable architecture changelogs from git history or snapshots, showing which elements and relationships were added, removed, or changed between two points in time.

## Phases

### Phase 1: Types and Git Loading
Create the core data structures and git integration:
- `internal/changelog/types.go` — Changelog, Change data structures
- `internal/changelog/git.go` — LoadModelAtGitRef function

### Phase 2: Changelog Generation
Implement the diff logic to generate changelogs:
- `internal/changelog/changelog.go` — Generate function
- Reuse `internal/diff/` logic from Issue #281

### Phase 3: Output Rendering
Implement all three output formats:
- `internal/changelog/render.go` — RenderMarkdown, RenderAsciiDoc, RenderJSON

### Phase 4: CLI Command
Wire up the user-facing command:
- `cmd/bausteinsicht/changelog.go` — changelog command
- Update `cmd/bausteinsicht/root.go` — register command

### Phase 5: Testing
Comprehensive test coverage:
- Unit tests for types, git loading, generation
- Snapshot tests for output rendering
- E2E tests with real git repos

## Acceptance Criteria
- ✅ CLI command works with `--since`, `--until`, `--format`, `--output` flags
- ✅ Git-based mode retrieves models from any git ref
- ✅ All three output formats (Markdown, AsciiDoc, JSON) work correctly
- ✅ Changelog shows added/removed/changed elements and relationships
- ✅ All tests pass
- ✅ Backward compatible with existing code

## Key Dependencies
- Requires Issue #281 (As-Is/To-Be Architecture Comparison) for diff logic
- Uses git command-line interface for model retrieval
