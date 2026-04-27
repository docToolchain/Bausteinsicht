# Plan: Architecture Constraints / Lint

## Purpose

Allow teams to codify architectural rules as constraints in the model. A new `lint` command evaluates these rules and fails with a non-zero exit code when violations are found — enabling enforcement in CI/CD pipelines.

## CLI Interface

```
bausteinsicht lint [--model <file>] [--format text|json]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--model` | `architecture.jsonc` | Model file path |
| `--format` | `text` | Output format (`json` for CI/CD integration) |

Exit codes:
- `0` — no violations
- `1` — one or more violations found
- `2` — model could not be loaded/parsed

## Model Changes

```jsonc
{
  "spec": { ... },
  "model": { ... },
  "views": [...],
  "constraints": [
    {
      "id": "C-001",
      "description": "Frontend must not call database directly",
      "rule": "no-relationship",
      "from-kind": "frontend",
      "to-kind": "database"
    },
    {
      "id": "C-002",
      "description": "All services must have a description",
      "rule": "required-field",
      "element-kind": "service",
      "field": "description"
    },
    {
      "id": "C-003",
      "description": "Maximum nesting depth is 3",
      "rule": "max-depth",
      "max": 3
    },
    {
      "id": "C-004",
      "description": "Only services may depend on the message broker",
      "rule": "allowed-relationship",
      "to-kind": "infra",
      "from-kinds": ["service"]
    }
  ]
}
```

## Supported Rule Types

| Rule | Parameters | Description |
|------|-----------|-------------|
| `no-relationship` | `from-kind`, `to-kind` | Forbids any relationship from elements of `from-kind` to elements of `to-kind` |
| `allowed-relationship` | `to-kind`, `from-kinds` | Only elements of listed kinds may have relationships to elements of `to-kind` |
| `required-field` | `element-kind`, `field` | All elements of `element-kind` must have the specified field set (non-empty) |
| `max-depth` | `max` | No element may be nested deeper than `max` levels |
| `no-circular-dependency` | — | Detects cycles in the relationship graph |
| `technology-allowed` | `element-kind`, `technologies` | Elements of `element-kind` may only use technologies from the given list |

## Architecture

### New / Modified Files

| File | Change |
|------|--------|
| `internal/model/types.go` | Add `Constraint` type, `Constraints []Constraint` to `BausteinsichtModel` |
| `internal/constraints/engine.go` | New: `Evaluate(model, constraints) []Violation` |
| `internal/constraints/rules.go` | New: one function per rule type |
| `internal/constraints/types.go` | New: `Violation{ConstraintID, Message, ElementIDs}` |
| `cmd/bausteinsicht/lint.go` | New `lint` command |

### Data Types

```go
type Constraint struct {
    ID          string   `json:"id"`
    Description string   `json:"description"`
    Rule        string   `json:"rule"`
    // rule-specific fields (flexible via map or dedicated structs)
    Params      map[string]json.RawMessage `json:"params,omitempty"`
}

type Violation struct {
    ConstraintID string
    Message      string
    ElementIDs   []string
}
```

## Output (text format)

```
Architecture Lint
=================
✅ C-001  Frontend must not call database directly
❌ C-002  All services must have a description
          → order-service: missing description
          → payment-service: missing description
✅ C-003  Maximum nesting depth is 3
❌ C-004  Only services may depend on the message broker
          → web-frontend → message-broker: frontend kind not allowed

2 violations found. Exit code: 1
```

## Output (JSON format, for CI)

```json
{
  "violations": [
    {
      "constraint_id": "C-002",
      "message": "All services must have a description",
      "elements": ["order-service", "payment-service"]
    },
    {
      "constraint_id": "C-004",
      "message": "Only services may depend on the message broker",
      "elements": ["web-frontend"]
    }
  ],
  "total": 2
}
```

## GitHub Actions Integration

```yaml
- name: Architecture Lint
  run: bausteinsicht lint --format json | tee lint-result.json
  # Exit code 1 fails the build automatically

- name: Upload lint results
  if: failure()
  uses: actions/upload-artifact@v4
  with:
    name: architecture-lint
    path: lint-result.json
```

## Testing

- Unit tests per rule type with valid/violating model fixtures
- E2E test: model with known violations → `lint --format json` → verify violation count and IDs
- Test: exit code 0 on clean model, exit code 1 on violations
- Property-based test: `no-circular-dependency` rule on randomly generated graphs
