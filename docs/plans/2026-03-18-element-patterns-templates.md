# Plan: Element Patterns / Topology Templates

## Purpose

Allow architects to define reusable multi-element topology patterns in the spec (e.g. a "microservice" pattern that creates a service + database + cache with standard relationships). A single CLI command expands a pattern into the model, eliminating repetitive manual element creation.

## CLI Interface

```
bausteinsicht add-from-pattern <pattern-id> --id <base-id> [--title <title>] [--model <file>]
```

| Flag | Required | Description |
|------|----------|-------------|
| `<pattern-id>` | yes | Pattern key from `spec.patterns` |
| `--id` | yes | Base ID prefix for generated elements |
| `--title` | no | Base title prefix (default: `--id`) |

### Example

```bash
bausteinsicht add-from-pattern microservice --id order --title "Order"
```

Output:
```
✅ Pattern 'microservice' applied with base ID 'order':
   + order-service   [service]   "Order Service"
   + order-db        [database]  "Order Database"
   + order-cache     [cache]     "Order Cache"
   + r-order-svc-db  order-service → order-db  "reads/writes"
   + r-order-svc-cache  order-service → order-cache  "caches"
```

## Model Changes

### Pattern Definitions in Spec

```jsonc
{
  "spec": {
    "elementKinds": [...],
    "patterns": {
      "microservice": {
        "description": "Standard microservice with database and cache",
        "elements": [
          {
            "id": "{base}-service",
            "kind": "service",
            "title": "{Title} Service"
          },
          {
            "id": "{base}-db",
            "kind": "database",
            "title": "{Title} Database",
            "technology": "PostgreSQL"
          },
          {
            "id": "{base}-cache",
            "kind": "cache",
            "title": "{Title} Cache",
            "technology": "Redis"
          }
        ],
        "relationships": [
          {
            "id": "r-{base}-svc-db",
            "from": "{base}-service",
            "to": "{base}-db",
            "label": "reads/writes"
          },
          {
            "id": "r-{base}-svc-cache",
            "from": "{base}-service",
            "to": "{base}-cache",
            "label": "caches"
          }
        ]
      },
      "event-driven-pair": {
        "description": "Producer and consumer connected via event broker",
        "elements": [
          { "id": "{base}-producer", "kind": "service", "title": "{Title} Producer" },
          { "id": "{base}-consumer", "kind": "service", "title": "{Title} Consumer" }
        ],
        "relationships": [
          { "id": "r-{base}", "from": "{base}-producer", "to": "{base}-consumer", "label": "events", "dataFlow": "event-streaming" }
        ]
      }
    }
  }
}
```

### Template Variables

| Variable | Value |
|----------|-------|
| `{base}` | Value of `--id` flag |
| `{Title}` | Value of `--title` flag (title-cased) |
| `{BASE}` | Value of `--id` uppercased |

## Pattern Validation

Validated at load time:
1. All element `kind` references must exist in `spec.elementKinds`
2. All relationship `from`/`to` must reference IDs defined within the same pattern (using template variables)
3. Pattern IDs must be unique within spec

## Conflict Detection

Before applying a pattern, `add-from-pattern` checks for ID collisions:

```
⚠ Element 'order-service' already exists in model.
  Use --prefix <prefix> to add a namespace, or --force to overwrite.
```

With `--prefix order-v2`:
- `{base}` → `order-v2-service`, `order-v2-db`, `order-v2-cache`

## Architecture

### New / Modified Files

| File | Change |
|------|--------|
| `internal/model/types.go` | Add `PatternDefinition`, `PatternElement`, `PatternRelationship` types; add `Patterns map[string]PatternDefinition` to `Specification` |
| `internal/model/validate.go` | Validate pattern definitions at load time |
| `internal/model/patterns.go` | New: `ExpandPattern(pattern, baseID, title) ([]Element, []Relationship, error)` |
| `cmd/bausteinsicht/add_from_pattern.go` | New `add-from-pattern` command |

### Expansion Algorithm

```go
func ExpandPattern(p PatternDefinition, baseID, title string) ([]Element, []Relationship, error) {
    vars := map[string]string{
        "{base}":  baseID,
        "{Title}": toTitleCase(title),
        "{BASE}":  strings.ToUpper(baseID),
    }
    elements := make([]Element, len(p.Elements))
    for i, tmpl := range p.Elements {
        elements[i] = Element{
            ID:    replaceVars(tmpl.ID, vars),
            Kind:  tmpl.Kind,
            Title: replaceVars(tmpl.Title, vars),
            // ... other fields
        }
    }
    // same for relationships
    return elements, relationships, nil
}
```

## Pattern Discovery

```bash
bausteinsicht add-from-pattern --list
```

Output:
```
Available patterns:
  microservice        Standard microservice with database and cache (3 elements, 2 relationships)
  event-driven-pair   Producer and consumer connected via event broker (2 elements, 1 relationship)
```

## Testing

- Unit test: `ExpandPattern` with `microservice` → verify generated element/relationship IDs
- Unit test: conflict detection fires on duplicate ID
- Unit test: `--list` shows all patterns with counts
- Validation test: pattern with undefined kind → error at load time
- E2E test: `add-from-pattern microservice --id order` → model file contains 3 new elements + 2 relationships
