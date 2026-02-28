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

### Technology Stack
- Implementation language: **Go** (ADR-002)
- DSL format: **JSONC with JSON Schema** (ADR-001)
- CLI framework: Cobra
- XML processing: beevik/etree or emicklei/mxgraph
- No JavaScript/Node.js (security concerns with npm)

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

### Future Ideas (Out of Scope for v1)
- As-Is / To-Be architecture comparison
- Structurizr/LikeC4 import
- CI/CD validation pipeline
