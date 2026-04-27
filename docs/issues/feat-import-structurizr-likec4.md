---
title: "feat: Import from Structurizr DSL / LikeC4"
labels: enhancement
---

## Beschreibung

Teams using Structurizr or LikeC4 cannot migrate to Bausteinsicht without rewriting their models from scratch. This feature adds a one-time import command that parses these DSL formats and generates a valid `architecture.jsonc` file.

## Motivation

- Structurizr and LikeC4 are widely used architecture-as-code tools
- Migration friction is a significant adoption barrier
- A one-time import does not require ongoing compatibility — it converts the model once
- Reduces the switching cost from other tools to near-zero

## Proposed Implementation

**New command:**
```
bausteinsicht import --from structurizr <input.dsl> [--output architecture.jsonc]
bausteinsicht import --from likec4 <input.c4> [--output architecture.jsonc]
```

**Structurizr mapping:**

| Structurizr | Bausteinsicht |
|-------------|--------------|
| `person` | kind `person` |
| `softwareSystem` | kind `system` |
| `container` | kind `container` (nested) |
| `component` | kind `component` (nested) |
| `->` | relationship |
| `systemContext` view | view with scope |

**LikeC4 mapping:** Direct mapping (both use user-defined kinds).

**Import warnings** for unsupported constructs (deployment diagrams, HTTP `!include`, styles).

**Exit codes:** `0` = success, `1` = parse error, `2` = output file exists (use `--force`)

## Limitations

- Structurizr `!include` resolved only for local files
- Deployment diagrams not imported (no equivalent in Bausteinsicht)
- Styles/themes ignored (use draw.io templates instead)

## Implementation Plan

See the implementation plan embedded in the GitHub issue.

## Affected Components

- `internal/importer/structurizr/` (new package)
- `internal/importer/likec4/` (new package)
- `cmd/bausteinsicht/import.go` (new)
