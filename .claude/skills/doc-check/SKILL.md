---
name: doc-check
description: >
  Checks and updates spec/architecture documentation and e2e test coverage at ticket start and at review.
  'start': identifies which docs need updating for a ticket and applies the updates.
  'review': verifies that docs, implementation, tests, and e2e coverage are consistent before merge.
license: MIT
compatibility:
  os: [linux, macos]
  tools: []
metadata:
  author: docToolchain
  version: "1.3"
allowed-tools: Bash Read Write Edit Glob Grep
argument-hint: "start [#<issue-number>] | review"
---

# Doc-Check Skill

You are a documentation consistency guardian for the Bausteinsicht project. You ensure that
spec and architecture documents are always kept in sync with implementation and tests.

**Requires Claude Code.** If running manually (no Claude Code), use the PR template checklist
as the fallback — it covers the same doc gates.

## Project Doc Structure

| Path | Content | Update when… |
|------|---------|--------------|
| `src/docs/spec/01_use_cases.adoc` | User-facing use cases | New CLI command or user-visible behavior added |
| `src/docs/spec/02_cli_specification.adoc` | CLI command reference | Any `--flag`, command name, subcommand rename, or output format changes |
| `src/docs/spec/03_data_models.adoc` | JSONC model schema | `internal/model/types.go` or `schemas/bausteinsicht.schema.json` changes |
| `src/docs/spec/04_acceptance_criteria.adoc` | Acceptance criteria per feature | New feature scope |
| `src/docs/spec/05_sync_specification.adoc` | Sync engine behavior spec | `internal/sync/` changes |
| `src/docs/arc42/chapters/05_building_block_view.adoc` | Package/component structure | New package added or renamed |
| `src/docs/arc42/chapters/06_runtime_view.adoc` | Data flows and runtime behavior | New data flow or runtime path within existing or new packages |
| `src/docs/arc42/chapters/08_concepts.adoc` | Cross-cutting concepts | New cross-cutting pattern (error handling, caching, etc.) |
| `src/docs/arc42/ADRs/` | Architecture decisions | New significant design decision |

## Mode: `start`

**Usage:** `/doc-check start` or `/doc-check start #<issue-number>`

### Steps

1. **Gather scope**
   - If an issue number was given: read it with `gh issue view <N> --json title,body,labels`
     - If the body is empty, infer scope from title + labels + branch name
   - If no issue: use `git branch --show-current` and `git log main...HEAD --oneline` to infer scope from branch name and commits
   - Check if implementation commits already exist on this branch:
     ```bash
     git diff main...HEAD --name-only -- '*.go' ':!*_test.go'
     ```
     If Go files are already changed: note "running start post-implementation (catch-up mode)" — proceed normally but skip the docs-first claim
   - Summarize in one sentence what will change

2. **Map scope to docs** — for each doc in the table above, decide: **needs update / no change needed / unsure**

   Apply **all** rules that match (not just the first):

   | If the scope touches… | Check these docs |
   |----------------------|-----------------|
   | New or changed `cmd/bausteinsicht/*.go` (non-test) | spec/01, spec/02 |
   | `internal/model/types.go` or `schemas/bausteinsicht.schema.json` or `internal/schema/` | spec/03 |
   | `internal/sync/` | spec/05 |
   | New `internal/<pkg>/` directory | arc42/05, arc42/06, **and** `src/docs/arc42/architecture.jsonc` (the self-hosted model itself — run `make arc42-drift-check` to confirm; see #524/#526, where 11 packages went undetected in the model for a long time because this was only a manual judgment call) |
   | New runtime data flow within an existing package | arc42/06 |
   | New cross-cutting pattern (error handling, logging, concurrency model) | arc42/08 |
   | Any other existing `internal/<pkg>/` with user-visible output change | spec/01, spec/02 |
   | New user-facing behavior without an existing acceptance criterion | spec/04 |
   | A significant design tradeoff (new library, new format, new algorithm) | new ADR |
   | New CLI command, new flag, or a fix where one command's output feeds another (import→sync, sync→export, snapshot→restore) | plan an `e2e/*_test.go` scenario chaining producer→consumer (see review mode's "E2E test coverage" check) — flag this now so it's not missed at review |

   **Catch-all:** Any change to an existing `internal/` package that alters CLI-visible behavior,
   output format, or exported API → check spec/01 and spec/02. When in doubt, flag as "unsure"
   rather than "no change needed."

3. **For each doc that needs update:**
   - Read the current content of that doc
   - Write the specific section that needs to change (add/update only the affected section — do not rewrite the whole doc)
   - Keep AsciiDoc format; follow existing heading levels and style
   - If an ADR is needed, create `src/docs/arc42/ADRs/ADR-NNN-Name.adoc` using Nygard format with a Weighted Pugh Matrix

4. **Report**
   ```
   | Doc | Status | Change made |
   |-----|--------|-------------|
   | spec/02_cli_specification.adoc | UPDATED | Added --threshold flag to stale command |
   | arc42/ADRs/ | NO CHANGE | No new design decision |
   ```

5. **Commit** all doc changes with:
   ```
   docs: update spec/architecture for <scope>
   ```
   If an issue number is known, append: `Closes (partial): #<issue>`
   Omit the trailer if no issue number was provided.

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
   - `schema`: changed files under `schemas/` or `internal/schema/`

3. **Check consistency** — answer each question:

   **A. Implementation coverage in docs**
   Apply the same routing table as `start` step 2 to each changed impl/schema file:
   - `cmd/bausteinsicht/*.go` → `spec/02` and `spec/01`
   - `internal/model/types.go` / `schemas/` / `internal/schema/` → `spec/03`
   - `internal/sync/*.go` → `spec/05`
   - New `internal/<pkg>/` directory → `arc42/05`, `arc42/06`, **and** `architecture.jsonc`
   - New runtime data flow in existing package → `arc42/06`
   - New cross-cutting pattern → `arc42/08`
   - Any other `internal/<pkg>/` with user-visible change → `spec/01`, `spec/02`
   - Significant design tradeoff → new ADR in `src/docs/arc42/ADRs/`

   Always run this concrete check (not just judgment) when any `internal/` or `cmd/` directory was added, renamed, or removed in the diff:
   ```bash
   make arc42-drift-check
   ```
   Non-zero exit → **❌ blocking** (a real package has no `container` element in `architecture.jsonc`, or vice versa). This is a scripted version of the routing-table rule above — it exists because that rule was manual-judgment-only for a long time and 11 packages drifted out of the model undetected (#524).

   Also run this when a new `cmd/bausteinsicht/*.go` command or subcommand was added:
   ```bash
   make arc42-process-coverage-check
   make arc42-runtime-coverage-check
   ```
   Non-zero exit → **⚠️ non-blocking** (advisory only — a textual mention check, not a semantic one; see the script headers). Flags commands not yet mentioned in §3.1.1's process diagram (`chapters/03_context_and_scope.adoc`) or chapter 6's runtime scenarios (`chapters/06_runtime_view.adoc`). This exists because §3.1.1 and chapter 6 both drifted badly behind the CLI's real command set before being caught (#535).

   **B. Doc coverage in tests**
   For each new acceptance criterion or spec section added: is there a test covering it?
   ```bash
   grep -rn "<feature-name>" --include="*_test.go" .
   ```

   **C. E2E test coverage** — every user-visible new feature or cross-command bug fix needs an
   end-to-end scenario, not just a unit test. Unit tests catch a wrong return value; they do not
   catch "command A writes a value command B then rejects" — that class of bug only shows up when
   the pipeline runs for real. (See [Issue #512](https://github.com/docToolchain/Bausteinsicht/issues/512):
   `import --from structurizr` wrote a `layout` value that `sync`'s own validation rejected — shipped
   in v1.2.0 with zero test coverage anywhere in the repo, unit or e2e, because no test ever ran
   `import` output through `sync`.)

   For each changed/added file, check whether it falls into one of these triggers:
   - New `cmd/bausteinsicht/*.go` command or subcommand
   - New or changed `--flag` with user-visible effect
   - A fix or feature where the input of one command is produced by another (import→sync,
     sync→export, snapshot→restore, etc.) — i.e. anything that only breaks when chained
   - A bug fix whose root cause is "component X assumes something about component Y's output/format"

   If any trigger matches, search for a scenario that actually chains the relevant commands:
   ```bash
   grep -rln "<command-name>\|<scenario-keyword>" e2e/*_test.go
   ```
   A unit test on the producing side alone (e.g. only testing the importer's output struct) does
   **not** satisfy this — the check must confirm the *consuming* command (sync/export/etc.) actually
   runs against that output in the test, the same way a real user's shell pipeline would.

   - Trigger matched, no e2e scenario chaining producer→consumer → ❌ (blocking)
   - Trigger matched, existing e2e test covers it but doesn't assert on the specific new behavior → ⚠️
   - Pure internal refactor with no user-visible or cross-command effect → no check needed

   **D. No stale docs** — check for renamed/deleted exported symbols still referenced in docs.

   Two-pass extraction (handles both top-level funcs and methods):
   ```bash
   # Pass 1: top-level func/type/const (e.g. "func ParseV2(")
   git diff main...HEAD -- '*.go' ':!*_test.go' \
     | grep '^-' \
     | grep -E '^-\s*(func|type|const)\s+[A-Z]' \
     | grep -oE '\b[A-Z][A-Za-z0-9_]+' | sort -u

   # Pass 2: methods (e.g. "func (r *Receiver) MethodName(")
   git diff main...HEAD -- '*.go' ':!*_test.go' \
     | grep '^-' \
     | grep -E '^-\s*func\s+\(' \
     | grep -oE '\)\s+[A-Z][A-Za-z0-9_]+' \
     | grep -oE '[A-Z][A-Za-z0-9_]+' | sort -u
   ```
   Then for each extracted symbol: `grep -rn "<Symbol>" src/docs/`

   - Stale reference to a removed/renamed symbol in docs → always ❌
   - Changed-but-undocumented symbol → ⚠️, **except**: CLI flag, command name, public schema field → ❌

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
   - `import --from structurizr` writes `layout: "auto"` but no e2e test runs the imported model
     through `sync` — the producer→consumer chain is untested (check C)

   ### Verdict
   PASS / NEEDS DOCS / FAIL
   ```

   Verdict rules:
   - **PASS**: no ❌ findings
   - **NEEDS DOCS**: only ⚠️ gaps (suggest updates but do not block)
   - **FAIL**: any ❌ found — list exactly what needs to be fixed

5. If verdict is **NEEDS DOCS** or **FAIL**: offer to fix the gaps immediately.
