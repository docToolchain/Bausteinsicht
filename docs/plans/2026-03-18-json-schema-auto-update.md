# Plan: JSON Schema Auto-Update from Spec

## Purpose

Generate an updated JSON Schema from the project's own `spec` section (element kinds, relationship kinds, tags, patterns). The generated schema extends the base Bausteinsicht schema with project-specific valid values, enabling IDE autocompletion for element kinds, tags, and status values specific to this project — without requiring a VS Code extension.

## CLI Interface

```
bausteinsicht schema [--model <file>] [--output <schema-file>] [--base-schema <url>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--output` | `.bausteinsicht-schema.json` | Output schema file |
| `--base-schema` | (bundled) | URL or path to base Bausteinsicht JSON Schema |

### Example

```bash
bausteinsicht schema --output .bausteinsicht-schema.json
# → Generated project-specific JSON Schema

# In architecture.jsonc, add:
# { "$schema": "./.bausteinsicht-schema.json", ... }
# → IDE now autocompletes kind values, tag IDs, pattern IDs
```

## What the Generated Schema Adds

Given this spec:

```jsonc
{
  "spec": {
    "elementKinds": [
      { "id": "service" },
      { "id": "database" },
      { "id": "frontend" }
    ],
    "tags": [
      { "id": "deprecated" },
      { "id": "deployed" }
    ]
  }
}
```

The generated schema adds enum constraints:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$ref": "https://bausteinsicht.example.com/schema/v1.json",
  "$defs": {
    "ElementKindId": {
      "enum": ["service", "database", "frontend"]
    },
    "TagId": {
      "enum": ["deprecated", "deployed"]
    },
    "ElementStatus": {
      "enum": ["proposed", "design", "implementation", "deployed", "deprecated", "archived"]
    }
  },
  "properties": {
    "model": {
      "properties": {
        "elements": {
          "items": {
            "properties": {
              "kind": { "$ref": "#/$defs/ElementKindId" },
              "tags": { "items": { "$ref": "#/$defs/TagId" } },
              "status": { "$ref": "#/$defs/ElementStatus" }
            }
          }
        }
      }
    }
  }
}
```

## IDE Integration

In `architecture.jsonc`:

```jsonc
{
  "$schema": "./.bausteinsicht-schema.json",
  "spec": { ... },
  "model": {
    "elements": [
      {
        "id": "order-service",
        "kind": "s<TAB>",   // ← IDE completes: "service"
        "tags": ["d<TAB>"], // ← IDE completes: "deprecated" or "deployed"
        "status": "<TAB>"   // ← IDE completes lifecycle values
      }
    ]
  }
}
```

Works in VS Code (with JSON Language Server), IntelliJ IDEA, Neovim (via LSP), and any editor with JSON Schema support.

## Auto-Update in Watch Mode

When `--schema` flag is passed to `watch`, the schema is regenerated automatically when `spec` changes:

```
bausteinsicht watch --schema
```

```
[14:32:01] spec.elementKinds changed — regenerating schema...
[14:32:01] Schema updated: .bausteinsicht-schema.json
```

## `init` Integration

`bausteinsicht init` creates the initial schema file and adds the `$schema` reference to `architecture.jsonc`:

```bash
bausteinsicht init
# → Creates architecture.jsonc with "$schema": "./.bausteinsicht-schema.json"
# → Creates .bausteinsicht-schema.json (generated from initial spec)
```

## Architecture

### New / Modified Files

| File | Change |
|------|--------|
| `internal/schema/generator.go` | New: `Generate(spec Specification) ([]byte, error)` |
| `internal/schema/base.go` | New: embedded base schema (or URL reference) |
| `cmd/bausteinsicht/schema.go` | New `schema` command |
| `cmd/bausteinsicht/watch.go` | Add `--schema` flag; trigger regeneration on spec changes |
| `cmd/bausteinsicht/init.go` | Auto-generate schema on `init` |

## Testing

- Unit test: generated schema contains all spec-defined kind IDs in enum
- Unit test: adding a new kind → schema enum includes it
- Unit test: generated JSON is valid JSON Schema (validate with JSON Schema meta-schema)
- E2E test: `schema` → output file parseable and contains expected enum values
- Test: `watch --schema` regenerates on spec change
