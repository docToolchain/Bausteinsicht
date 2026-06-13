---
name: doc-check
description: >
  Checks and updates spec/architecture documentation at ticket start and at review.
  'start': identifies which docs need updating for a ticket and applies the updates.
  'review': verifies that docs, implementation, and tests are consistent before merge.
license: MIT
compatibility:
  os: [linux, macos, windows]
  tools: []
metadata:
  author: docToolchain
  version: "1.0"
allowed-tools: Bash Read Write Edit Glob Grep
argument-hint: "start [#<issue-number>] | review"
---

# Doc-Check Skill

You are a documentation consistency guardian for the Bausteinsicht project. You ensure that
spec and architecture documents are always kept in sync with implementation and tests.

## Project Doc Structure

| Path | Content | Update when… |
|------|---------|--------------|
| `src/docs/spec/01_use_cases.adoc` | User-facing use cases | New CLI command or user-visible behavior added |
| `src/docs/spec/02_cli_specification.adoc` | CLI command reference | Any `--flag`, command name, or output format changes |
| `src/docs/spec/03_data_models.adoc` | JSONC model schema | `internal/model/types.go` changes |
| `src/docs/spec/04_acceptance_criteria.adoc` | Acceptance criteria per feature | New feature scope |
| `src/docs/spec/05_sync_specification.adoc` | Sync engine behavior spec | `internal/sync/` changes |
| `src/docs/arc42/chapters/05_building_block_view.adoc` | Package/component structure | New package added or renamed |
| `src/docs/arc42/chapters/06_runtime_view.adoc` | Data flows and runtime behavior | New data flow or runtime path |
| `src/docs/arc42/chapters/08_concepts.adoc` | Cross-cutting concepts | New cross-cutting pattern |
| `src/docs/arc42/ADRs/` | Architecture decisions | New significant design decision |

## Mode: `start`

**Usage:** `/doc-check start` or `/doc-check start #<issue-number>`

### Steps

1. **Gather scope**
   - If an issue number was given: read it with `gh issue view <N> --json title,body,labels`
   - If no issue: use `git branch --show-current` and `git log main...HEAD --oneline` to infer scope from branch name and commits
   - Summarize in one sentence what will change

2. **Map scope to docs** — for each doc in the table above, decide: **needs update / no change needed / unsure**
   Use these rules:
   - New or changed `cmd/bausteinsicht/*.go` file → check spec/02, spec/01
   - Changes to `internal/model/types.go` or schema → check spec/03
   - Changes to `internal/sync/` → check spec/05
   - New `internal/<pkg>/` directory → check arc42/05 and arc42/06
   - A new user-facing behavior without an existing acceptance criterion → check spec/04
   - A significant design tradeoff (new library, new format, new algorithm) → new ADR

3. **For each doc that needs update:**
   - Read the current content of that doc
   - Write the specific section that needs to change (add/update only the affected section — do not rewrite the whole doc)
   - Keep AsciiDoc format; follow existing heading levels and style
   - Note: if an ADR is needed, create `src/docs/arc42/ADRs/ADR-NNN-Name.adoc` using Nygard format with a Weighted Pugh Matrix

4. **Report**
   Print a table:
   ```
   | Doc | Status | Change made |
   |-----|--------|-------------|
   | spec/02_cli_specification.adoc | UPDATED | Added --threshold flag to stale command |
   | arc42/ADRs/ | NO CHANGE | No new design decision |
   ```

5. **Commit** all doc changes with:
   ```
   docs: update spec/architecture for <scope>

   Closes (partial): #<issue>
   ```

## Mode: `review`

**Usage:** `/doc-check review`

### Steps

1. **Get the diff**
   ```bash
   git diff main...HEAD --name-only
   git diff main...HEAD -- src/docs/
   git diff main...HEAD -- '*.go' ':!*_test.go'
   git diff main...HEAD -- '*_test.go'
   ```

2. **Classify changed files** into four buckets:
   - `impl`: changed `.go` files (non-test)
   - `tests`: changed `_test.go` files
   - `docs`: changed files under `src/docs/`
   - `schema`: changed `schemas/`

3. **Check consistency** — answer each question:

   **A. Implementation coverage in docs**
   For each changed impl file: is there a doc that describes the new/changed behavior?
   - `cmd/bausteinsicht/*.go` → look for matching content in `spec/02_cli_specification.adoc`
   - `internal/model/types.go` → look for matching content in `spec/03_data_models.adoc`
   - `internal/sync/*.go` → look for matching content in `spec/05_sync_specification.adoc`
   - `internal/<new-pkg>/` → look for matching content in `arc42/05_building_block_view.adoc`

   **B. Doc coverage in tests**
   For each new acceptance criterion or spec section added in docs: is there a test exercising it?
   - Grep the test files for the feature name or flag name mentioned in the doc change

   **C. No stale docs**
   For each deleted or renamed symbol (function, flag, struct) in impl: is the old name still referenced in docs?

4. **Output a review report:**

   ```
   ## Doc-Check Review

   ### ✅ Covered
   - `internal/stale/detector.go` → `spec/02_cli_specification.adoc` section "stale" updated
   - New acceptance criterion in spec/04 → test `TestDetect_DeterministicOrder` covers it

   ### ⚠️ Gaps (non-blocking)
   - `internal/graph/analyzer.go` changed but no doc update found — verify if user-visible

   ### ❌ Inconsistencies (blocking)
   - Flag `--threshold` added in `stale.go:45` but missing from `spec/02_cli_specification.adoc`
   - Old name `MarkElements` still referenced in `spec/05` but renamed to `MarkInDrawio` in code

   ### Verdict
   PASS / NEEDS DOCS / FAIL
   ```

   Verdict rules:
   - **PASS**: no ❌ findings
   - **NEEDS DOCS**: only ⚠️ gaps (suggest doc updates but do not block)
   - **FAIL**: any ❌ inconsistency found — list exactly what needs to be fixed

5. If verdict is **NEEDS DOCS** or **FAIL**: offer to fix the gaps immediately.
