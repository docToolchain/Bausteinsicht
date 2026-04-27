# Plan: Architecture Changelog Generation

## Purpose

Automatically generate a human-readable architecture changelog from git history and/or saved snapshots. Answers "what changed architecturally between v1.0 and v2.0?" — as a Markdown or AsciiDoc document suitable for release notes, architecture reviews, and stakeholder communication.

## CLI Interface

```
bausteinsicht changelog [--model <file>] [--since <git-ref|snapshot-id>] [--until <git-ref|snapshot-id>]
                        [--format markdown|asciidoc|json] [--output <file>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--since` | previous git tag | Start ref (git tag, commit SHA, or snapshot ID) |
| `--until` | `HEAD` | End ref |
| `--format` | `markdown` | Output format |
| `--output` | stdout | Output file path |

### Examples

```bash
# Changelog since last git tag
bausteinsicht changelog --since v1.0 --format markdown --output ARCHITECTURE-CHANGELOG.md

# Changelog between two snapshots
bausteinsicht changelog --since snapshot-001 --until snapshot-005

# Changelog for last 30 days (git-based)
bausteinsicht changelog --since "30 days ago"
```

## Two Operating Modes

### Mode 1: Git-Based (default)

Uses `git show <ref>:architecture.jsonc` to retrieve the model at any git ref. No snapshots required.

```go
func loadModelAtRef(modelPath, gitRef string) (*model.BausteinsichtModel, error) {
    out, err := exec.Command("git", "show", gitRef+":"+modelPath).Output()
    // parse JSON from output
}
```

Requires: git available in PATH; model file tracked in git.

### Mode 2: Snapshot-Based

Uses `.bausteinsicht-snapshots/` (from the Versioned Snapshots feature) as the source of comparison points. Works even without git (e.g. in CI environments that do shallow clones).

## Output Format

### Markdown

```markdown
# Architecture Changelog

## v1.0 → v2.0 (2026-01-15 → 2026-03-18)

### Added (3 elements)
- **notification-service** `[service]` — Notification Service _(sends email and push notifications)_
- **audit-log** `[storage]` — Audit Log Store
- **message-broker** `[infra]` — Message Broker _(Kafka cluster)_

### Removed (1 element)
- ~~**legacy-monolith**~~ `[system]` — Legacy Order System _(replaced by microservices)_

### Changed (2 elements)
- **payment-service** — title: "Payment Service" → "Payment Service v2"; technology: "Java" → "Go"
- **api-gateway** — description updated

### New Relationships (3)
- order-service → message-broker _(publishes: OrderPlaced)_
- notification-service → message-broker _(subscribes: OrderPlaced)_
- api-gateway → audit-log _(logs all requests)_

### Removed Relationships (2)
- ~~order-service → payment-service~~ _(now event-driven)_
- ~~web-frontend → order-service~~ _(now via api-gateway)_
```

### AsciiDoc

Same content rendered in AsciiDoc syntax for docToolchain integration.

### JSON

Machine-readable format suitable for automated release notes tools:

```json
{
  "from": { "ref": "v1.0", "date": "2026-01-15" },
  "to":   { "ref": "v2.0", "date": "2026-03-18" },
  "added":   [{ "id": "notification-service", "kind": "service", "title": "..." }],
  "removed": [{ "id": "legacy-monolith", ... }],
  "changed": [{ "id": "payment-service", "changes": { "title": [...], "technology": [...] } }],
  "relationships": { "added": [...], "removed": [...] }
}
```

## Integration with CI/CD

```yaml
- name: Generate Architecture Changelog
  run: |
    bausteinsicht changelog \
      --since ${{ github.event.before }} \
      --until ${{ github.sha }} \
      --format markdown \
      --output reports/architecture-changes.md

- name: Post to PR comment
  uses: actions/github-script@v7
  with:
    script: |
      const body = require('fs').readFileSync('reports/architecture-changes.md', 'utf8')
      github.rest.issues.createComment({ issue_number: context.issue.number, body })
```

## Architecture

### New / Modified Files

| File | Change |
|------|--------|
| `internal/changelog/git.go` | New: `LoadModelAtGitRef(path, ref string) (*model.BausteinsichtModel, error)` |
| `internal/changelog/changelog.go` | New: `Generate(from, to *model.BausteinsichtModel) Changelog` |
| `internal/changelog/render.go` | New: `RenderMarkdown`, `RenderAsciiDoc`, `RenderJSON` |
| `internal/changelog/types.go` | New: `Changelog`, `ElementChange`, `RelationshipChange` |
| `cmd/bausteinsicht/changelog.go` | New `changelog` command |

Reuses `internal/diff/` types from the As-Is/To-Be or Snapshot feature if implemented first.

## Testing

- Unit test: `Generate` with known before/after models → verify added/removed/changed counts
- Unit test: `RenderMarkdown` output matches expected snapshot
- Unit test: `LoadModelAtGitRef` with test git repo fixture
- E2E test: `changelog --since HEAD~1` in a repo with one commit change → output contains change
- Test: `--format json` output is valid JSON with expected structure
