# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

# Bausteinsicht

Architecture-as-code tool with draw.io as visual frontend and bidirectional synchronization.

## Project Conventions

### Documentation
- All documentation in **English**
- Documentation format: **AsciiDoc (.adoc)**
- ADR path: `src/docs/arc42/ADRs/`
- ADR filename: `ADR-NNN-Name.adoc` (e.g., `ADR-001-DSL-Format.adoc`)
- ADR format: Nygard with Weighted Pugh Matrix (-1/0/1 scale)
- PRD path: `src/docs/PRD/`
- Spec path: `src/docs/spec/`
- Security reports: `src/docs/security/` (fortlaufend, mit Changelog)

### Technology Stack
- Implementation language: **Go** (ADR-002)
- DSL format: **JSONC with JSON Schema** (ADR-001)
- CLI framework: Cobra
- XML processing: beevik/etree
- No JavaScript/Node.js for the product itself (security concerns with npm supply chain)

### Quality Goals (Top 3)
1. **Learnability** — new users productive within 30 minutes
2. **IDE Support** — autocompletion/validation via JSON Schema, no plugin needed
3. **LLM Friendliness** — JSON model readable/writable by AI agents, CLI for automation

### Key Design Decisions
- Flexible element hierarchy (not limited to 4 C4 levels)
- Unique variable names as element IDs for synchronization
- Template-based styling (templates are draw.io files)
- Page-based drill-down navigation (one draw.io page per view, cross-page links + back button; see ADR-009)
- CLI + watch mode; CLI commands for LLM-driven workflows

## Development Environment

### Devcontainer (recommended)
A `.devcontainer/` configuration provides a fully reproducible dev environment with all tools pre-installed. Use with VS Code Dev Containers, GitHub Codespaces, or the `devcontainer` CLI.

Start the container and run Claude Code autonomously:
```bash
devcontainer up --workspace-folder .
devcontainer exec --workspace-folder . claude --dangerously-skip-permissions -p "your prompt"
```

Key details:
- Claude Code is installed via **native installer** (not npm) — no Node.js dependency
- draw.io runs headless via `xvfb-run` — use `drawio-export` wrapper for exports
- `COLORTERM=truecolor` is set for correct terminal color rendering

### Headless draw.io Export

The `bausteinsicht export` and `drawio-export` commands require:
1. **`dbus` daemon running** — Electron needs D-Bus for IPC. If export fails with "Export failed" or "input file/directory not found", start dbus:
   ```bash
   sudo mkdir -p /run/dbus && sudo dbus-daemon --system --fork
   ```
2. **`xvfb-run -a`** — the `-a` flag auto-picks a free display (avoids conflicts with existing X servers)
3. **`--no-sandbox`** — required in containers without user namespaces

The devcontainer `postStartCommand` starts dbus automatically. The `drawio-export` wrapper handles xvfb and `--no-sandbox`.

GPU errors in stderr (`"Exiting GPU process due to errors during initialization"`) are **harmless** — draw.io falls back to software rendering.

### Makefile
All build, test, and analysis commands are available via `make`:
- `make build` — build the CLI binary
- `make test` / `make test-race` — run tests (with race detector)
- `make check` — run all analysis tools + race-detected tests
- `make vet` / `make staticcheck` / `make gosec` / `make nilaway` / `make govulncheck` — individual analysis tools
- `make gitleaks` — scan for secrets
- `make golangci-lint` — meta-linter
- `make install-tools` — install Go-based tools

### Installed Tools
- `go vet`, `staticcheck` — static analysis
- `gosec` — security scanner
- `nilaway` — nil pointer analysis
- `govulncheck` — vulnerability scanner
- `golangci-lint` — meta-linter
- `gitleaks` — secret scanner
- `draw.io` CLI (headless via xvfb in devcontainer)
- `claude` (Claude Code CLI)
- `human` (gethuman.sh — AI agent issue tracker integration)

## Code Architecture

### Package Structure

```
cmd/bausteinsicht/     # CLI entry point — Cobra commands, one file per command
internal/model/        # DSL types, loader (JSONC→struct), validation, patch, resolve
internal/drawio/       # draw.io XML document/element/connector/label/template wrappers (beevik/etree)
internal/sync/         # Bidirectional sync engine: diff, forward/reverse apply, conflict resolution, state
internal/diagram/      # Export to C4-PlantUML / Mermaid text formats
internal/watcher/      # File-system watcher (fsnotify) for --watch mode
```

### Data Flow

1. **Model** — JSONC file parsed by `internal/model.Load()` into `BausteinsichtModel` (elements keyed by dot-path variable names, e.g. `system.backend.api`)
2. **Sync cycle** (`internal/sync.Run`) — pure function; no I/O:
   - `DetectChanges` diffs model+drawio against stored `SyncState` (`.bausteinsicht-sync` JSON file)
   - Conflict resolution: model always wins
   - `ApplyForward` writes model changes → draw.io XML
   - `ApplyReverse` writes draw.io label edits → model struct
3. **State** persisted atomically to `.bausteinsicht-sync` (SHA-256 checksummed JSON) so next sync can detect what changed on either side
4. **Export** — `export diagram` renders views to PlantUML/Mermaid; `export table` produces CSV/Markdown; `export` calls headless draw.io for PNG/PDF

### Key Conventions

- **Element IDs are dot-separated variable paths** — `parent.child.grandchild`. `model.FlattenElements` recursively expands the nested map to a flat `map[string]*Element`.
- **draw.io elements carry `bausteinsicht_id` attribute** — this is the synchronization anchor between the two file formats.
- **Views filter what is rendered** — each view has `include`/`exclude` lists; `model.ResolveView` expands them to a flat element ID set.
- **Templates are `.drawio` files** — visual styles come from template pages, not hardcoded; `internal/drawio.TemplateSet` loads and clones them.
- **Run a single package's tests:** `go test ./internal/sync/` (or any other package path)
- **Run a single test:** `go test -run TestName ./internal/sync/`

## Workflow Rules

### Ticket Start
When starting work on a ticket:
1. Run `/doc-check start #<issue>` — identifies which spec/architecture docs need updating and applies them before implementation begins (docs-first approach)

> `/doc-check` is a Claude Code skill. Without Claude Code, use the Documentation checklist in the PR template as the manual fallback.

### PR Merge Policy
Before merging any PR:
1. **Doc check** — run `/doc-check review` to verify spec/architecture is consistent with implementation and tests
2. **Security review** on the changes
3. **Code review** on the changes

#### Quality Gate Override Policy
SonarCloud's **New Code Coverage** gate (threshold: **65%**) may trip on PRs that touch previously-untested legacy code without adding new logic. A verified **behavior-preserving refactor** may be admin-merged despite the gate if ALL of these hold:
- All functional CI checks are green (build, unit tests, lint, golangci-lint)
- An independent code review confirms no new logic was added
- The merge reason is noted in the PR description (e.g. "admin merge: refactor only, gate tripped by legacy untested lines")

Pads and exclusion-based workarounds are **not** acceptable alternatives — they hide real coverage gaps.

### Security Report
The security report at `src/docs/security/2026-03-01-security-review.adoc` is a living document. Update it (with a Changelog entry) whenever:
- Security findings are fixed or new ones discovered
- Dependencies are updated
- Automated tool results change

### Future Ideas (Out of Scope for v1)
- CI/CD validation pipeline
- XMI import from Enterprise Architect (issue #401)

(As-Is/To-Be comparison and Structurizr/LikeC4 import are implemented — see the `diff` and `import` commands.)

## Release Process

Releases are **fully automated via GitHub Actions** — do not build or upload artifacts by hand.

### How to cut a release
1. **Pick the version** with semver against the last tag: features → minor (`v1.1.0` → `v1.2.0`), fixes only → patch, breaking changes → major. Check with `git log <last-tag>..HEAD` for `feat:`/`fix:` and any `BREAKING`.
2. **Make sure `main` is green and current**, then tag from `main` and push:
   ```bash
   git tag -a vX.Y.Z -m "Release vX.Y.Z

   <short summary + highlights>"
   git push origin vX.Y.Z
   ```
   The tag push triggers `.github/workflows/release.yml`. Tags are **not** branch-protected, so the push is the only trigger needed (no PR).
3. **Watch the run:** `gh run watch <id> --exit-status`.

### What the Action produces
`release.yml` runs **goreleaser** (`.goreleaser.yml`), which:
- cross-compiles for **linux/darwin/windows × amd64/arm64/arm**,
- injects the version via `-X main.version={{.Version}}` (so `bausteinsicht --version` prints the tag),
- emits **SPDX + CycloneDX SBOMs** per archive (SLSA L2) and `checksums.txt`,
- creates the GitHub Release with a changelog grouped by Conventional-Commit prefix.

### Two prerequisites that must stay intact
- **syft must be installed before goreleaser** — `release.yml` does this via `anchore/sbom-action/download-syft` (SHA-pinned). goreleaser shells out to syft for SBOMs and does **not** bundle it; without this step the SBOM stage fails with `exec: "syft": executable file not found`.
- **`.goreleaser.yml` `sboms` uses the `cmd`/`args`/`documents` form** (not `format:`/`output:`, which are not in the goreleaser v2 schema). Validate config changes locally with `goreleaser check` before tagging.

### After the release: write good release notes
goreleaser's auto-changelog is raw (one line per commit). **Always enrich it** so a reader understands the release at a glance — prepend a curated intro **above** the auto-generated changelog (keep the changelog):
```bash
gh release edit vX.Y.Z --notes-file notes.md   # curated intro + appended auto-changelog
```
The curated intro should cover: a one-paragraph summary (counts, no-breaking-changes note), **Highlights** grouped by theme (a table for new commands), **Install** (download + `go install …@vX.Y.Z`), and **Verify** (SBOMs + `sha256sum -c checksums.txt`). See the [v1.2.0 release](https://github.com/docToolchain/Bausteinsicht/releases/tag/v1.2.0) as the reference shape.

### If a tagged release fails
Fix the cause on `main` via PR (config/workflow changes are review-gated), then **move the tag** onto the fixed commit and re-push:
```bash
git tag -d vX.Y.Z && git push origin :refs/tags/vX.Y.Z
git tag -a vX.Y.Z -m "…" && git push origin vX.Y.Z
```

## Risk Radar Assessment

_Generated by `/risk-assess` on 2026-03-04 — Architecture Decision: See [ADR-003](src/docs/arc42/ADRs/ADR-003-Risk-Classification.adoc)_

### Module: bausteinsicht
| Dimension | Score | Level | Evidence |
|-----------|-------|-------|----------|
| Code Type | 2 | Business Logic | Architecture model processing, XML sync engine, template rendering — no auth/API/DB |
| Language | 1 | Statically typed | 69 `.go` files (Go) |
| Deployment | 1 | Internal tool | Open-source CLI, primary use company-internal for architecture diagrams |
| Data Sensitivity | 0 | Public data | Processes architecture model definitions (JSONC/XML), no personal data |
| Blast Radius | 0 | Cosmetic / Tech debt | Incorrect diagram output; data loss theoretically possible but trivially recoverable from git |

**Tier: 2 — Extended Assurance** (determined by Code Type = 2)

### Mitigations: bausteinsicht (Tier 2)

_Updated by `/risk-mitigate` on 2026-03-04_

#### Tier 1 — Automated Gates
| Measure | Status | Details |
|---------|--------|---------|
| Linter & Formatter | ✅ Present | `golangci-lint` in CI (`go.yml`), `go vet`, `staticcheck` via Makefile |
| Type Checking | ✅ Present | Go is statically typed; `go build` enforces types |
| Pre-Commit Hooks | ✅ Set up | `scripts/pre-commit` — gofmt, go vet, golangci-lint, gitleaks; install via `make install-hooks` |
| Dependency Check | ✅ Present | `govulncheck` via Makefile; `gosec` for security scanning |
| CI Build & Unit Tests | ✅ Present | GitHub Actions `go.yml`: build + test + golangci-lint |

#### Tier 2 — Extended Assurance
| Measure | Status | Details |
|---------|--------|---------|
| SAST | ✅ Present | `gosec` (security scanner), `nilaway` (nil pointer analysis), `staticcheck` |
| AI Code Review | ✅ Present | Claude Code with code-review plugin; PR merge policy requires review |
| Property-Based Tests | ✅ Set up | `pgregory.net/rapid` — label roundtrip + escapeHTML + trimBrackets property tests |
| SonarQube Quality Gate | ✅ Present | SonarCloud on all PRs; new-code-coverage threshold set to 65% (see override policy in PR Merge Policy section); `sonar.qualitygate.wait=true` in `sonar-project.properties` |
| Sampling Review (~20%) | ✅ Present | PR merge policy: security review + code review required |

**Overall Status:** 10/10 measures active

## Branch & PR Management

### Duplicate PR Prevention
To prevent duplicate/parallel branch development (as happened with PR #332 vs #361), we've implemented automated checks:

**See:** [Issue #362](https://github.com/docToolchain/Bausteinsicht/issues/362) for detailed implementation plan

Key automation layers:
1. **GitHub Actions**: Stale branch detection, duplicate PR detection, branch freshness checks
2. **Pre-Commit Hooks**: Warn if branch is behind main before committing
3. **PR Template**: Developer checklist for branch status and duplicate checks
4. **Branch Protection Rules**: Require branch to be up-to-date with main
5. **Periodic Cleanup**: Weekly automation to identify merged branches and stale branches
6. **Local CLI Tools**: `scripts/check-duplicate-branches.sh` for manual verification

**Best Practice:**
- Always rebase your branch on main before opening a PR: `git rebase origin/main`
- Check for overlapping work before PR creation
- Delete merged branches promptly: `git branch -d <branch>` and `git push origin --delete <branch>`
