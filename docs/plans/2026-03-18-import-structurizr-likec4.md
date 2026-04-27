# Plan: Import from Structurizr / LikeC4

## Purpose

Enable one-time migration of existing architecture models from Structurizr DSL or LikeC4 into the Bausteinsicht JSON format. Lowers adoption friction for teams already using these tools.

## CLI Interface

```
bausteinsicht import --from structurizr <input-file> [--output <model-file>]
bausteinsicht import --from likec4 <input-file> [--output <model-file>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--from` | (required) | Source format: `structurizr` or `likec4` |
| `--output` | `architecture.jsonc` | Output model file path |
| `--dry-run` | false | Print output to stdout instead of writing file |

Exit codes:
- `0` — import successful
- `1` — parse error (with message)
- `2` — output file already exists (use `--force` to overwrite)

## Structurizr DSL Mapping

Structurizr uses a fixed 4-level hierarchy (Person → SoftwareSystem → Container → Component). Bausteinsicht uses user-defined kinds.

| Structurizr Concept | Bausteinsicht mapping |
|---------------------|----------------------|
| `person` | element kind `person` |
| `softwareSystem` | element kind `system` |
| `container` | element kind `container` |
| `component` | element kind `component` |
| `->` (relationship) | relationship |
| `!include` | resolved before parsing |
| `workspace.description` | model root description |
| `views { ... }` | `views` array |
| `systemContext` view | view with `scope` = system element |
| `container` view | view scoped to container element |

### Example Structurizr Input

```
workspace "My System" {
  model {
    user = person "User" "A customer"
    system = softwareSystem "Order System" {
      web = container "Web App" "React SPA" "TypeScript"
      api = container "API" "REST backend" "Go"
      db  = container "Database" "" "PostgreSQL"
    }
    user -> web "Uses"
    web -> api "Calls"
    api -> db "Reads/Writes"
  }
  views {
    systemContext system "Context" { include * }
    container system "Containers" { include * }
  }
}
```

### Example Generated Output

```jsonc
{
  "spec": {
    "elementKinds": [
      { "id": "person",    "name": "Person" },
      { "id": "system",    "name": "Software System" },
      { "id": "container", "name": "Container", "container": true }
    ],
    "relationshipKinds": [{ "id": "uses", "name": "Uses" }]
  },
  "model": {
    "elements": [
      { "id": "user",   "kind": "person",    "title": "User",      "description": "A customer" },
      { "id": "system", "kind": "system",    "title": "Order System",
        "children": [
          { "id": "web", "kind": "container", "title": "Web App",   "technology": "TypeScript" },
          { "id": "api", "kind": "container", "title": "API",       "technology": "Go" },
          { "id": "db",  "kind": "container", "title": "Database",  "technology": "PostgreSQL" }
        ]
      }
    ],
    "relationships": [
      { "id": "r1", "from": "user", "to": "web", "label": "Uses" },
      { "id": "r2", "from": "web",  "to": "api", "label": "Calls" },
      { "id": "r3", "from": "api",  "to": "db",  "label": "Reads/Writes" }
    ]
  },
  "views": [
    { "key": "Context",    "title": "Context",    "scope": "system", "include": ["*"] },
    { "key": "Containers", "title": "Containers", "scope": "system", "include": ["*"] }
  ]
}
```

## LikeC4 DSL Mapping

LikeC4 already uses user-defined element kinds (similar to Bausteinsicht).

| LikeC4 Concept | Bausteinsicht mapping |
|----------------|----------------------|
| `specification { element ... }` | `spec.elementKinds` |
| `model { ... }` | `model.elements` (preserving hierarchy) |
| `views { view ... }` | `views` array |
| `-> "label"` | `relationships` |
| `include` / `exclude` | view `include` / `exclude` patterns |

## Architecture

### New / Modified Files

| File | Change |
|------|--------|
| `internal/importer/structurizr/parser.go` | New: tokenizer + recursive descent parser for Structurizr DSL |
| `internal/importer/structurizr/mapper.go` | New: maps parsed AST to `BausteinsichtModel` |
| `internal/importer/likec4/parser.go` | New: LikeC4 DSL parser |
| `internal/importer/likec4/mapper.go` | New: maps to `BausteinsichtModel` |
| `internal/importer/types.go` | New: shared `ImportResult{Model, Warnings}` |
| `cmd/bausteinsicht/import.go` | New `import` command with `--from` flag |

## Import Warnings

The import command reports non-fatal issues:

```
Import complete with warnings:
  ⚠ Element 'legacy-system' has no description
  ⚠ View 'Dynamic' uses dynamic view syntax — not imported (use dynamic views feature)
  ⚠ 'styles' block ignored — apply custom styles via draw.io template
```

## Testing

- Unit tests for Structurizr parser with sample DSL files (stored as test fixtures in `testdata/`)
- Unit tests for LikeC4 parser
- Round-trip test: import → validate → compare element/relationship counts
- E2E test: import real-world example → `validate` passes

## Limitations (documented)

- `!include` directives: resolved only if files are local (no HTTP includes)
- Structurizr deployment diagrams: not imported (no equivalent concept)
- LikeC4 custom icon URLs: converted to empty technology field
- Styles/themes: ignored (use draw.io templates instead)
