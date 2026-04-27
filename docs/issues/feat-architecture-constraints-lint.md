---
title: "feat: Architecture Constraints / Lint Command"
labels: enhancement
---

## Beschreibung

Architecture rules (e.g. "frontends must not call databases directly", "all services must have a description") are currently undocumented and unenforced. This feature adds a `constraints` section to the model and a `lint` command that evaluates rules and fails with a non-zero exit code in CI/CD pipelines.

## Motivation

- Architecture drift is a common problem in growing systems
- Enforcing rules in CI/CD catches violations before code review
- JSON output (`--format json`) integrates seamlessly with GitHub Actions, GitLab CI, and other pipelines
- Rules are documented in the model, making architectural decisions explicit and auditable

## Proposed Implementation

**Model extension:**
```jsonc
{
  "constraints": [
    { "id": "C-001", "description": "Frontend must not call database directly",
      "rule": "no-relationship", "from-kind": "frontend", "to-kind": "database" },
    { "id": "C-002", "description": "All services must have a description",
      "rule": "required-field", "element-kind": "service", "field": "description" },
    { "id": "C-003", "description": "Max nesting depth is 3",
      "rule": "max-depth", "max": 3 }
  ]
}
```

**New command:**
```
bausteinsicht lint [--format text|json]
```

**Exit codes:** `0` = clean, `1` = violations found, `2` = model parse error

**Supported rules:** `no-relationship`, `allowed-relationship`, `required-field`, `max-depth`, `no-circular-dependency`, `technology-allowed`

**GitHub Actions example:**
```yaml
- run: bausteinsicht lint --format json
  # non-zero exit fails the build automatically
```

## Implementation Plan

See the implementation plan embedded in the GitHub issue.

## Affected Components

- `internal/model/types.go`
- `internal/constraints/` (new package)
- `cmd/bausteinsicht/lint.go` (new)
