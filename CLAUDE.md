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
- Zoom-based drill-down navigation on single draw.io page
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

### PR Merge Policy
Before merging any PR:
1. **Security review** on the changes
2. **Code review** on the changes

### Security Report
The security report at `src/docs/security/2026-03-01-security-review.adoc` is a living document. Update it (with a Changelog entry) whenever:
- Security findings are fixed or new ones discovered
- Dependencies are updated
- Automated tool results change

### Future Ideas (Out of Scope for v1)
- As-Is / To-Be architecture comparison
- Structurizr/LikeC4 import
- CI/CD validation pipeline

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
| SonarQube Quality Gate | ❌ N/A | Not configured |
| Sampling Review (~20%) | ✅ Present | PR merge policy: security review + code review required |

**Overall Status:** 9/10 measures active (1 N/A)

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
