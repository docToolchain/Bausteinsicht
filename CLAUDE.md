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
