---
title: "feat: JSON Schema Auto-Update from Spec"
labels: enhancement
---

## Beschreibung

Generate a project-specific JSON Schema from the model's `spec` section (element kinds, tags, patterns). The schema extends the base Bausteinsicht schema with project-specific enum values, enabling IDE autocompletion for kind names, tag IDs, and status values — without a VS Code extension.

## Motivation

- The base JSON Schema only validates structure; it cannot know which element kinds a project defines
- Teams currently get no autocompletion for `"kind": "s..."` — they must remember kind IDs manually
- A generated project schema enables `"kind": "<TAB>"` to show `["service", "database", "frontend"]` in any JSON Schema-aware editor (VS Code, IntelliJ, Neovim LSP)
- Regeneration on spec change keeps the schema in sync automatically

## Proposed Implementation

**New command:**
```
bausteinsicht schema [--output .bausteinsicht-schema.json]
```

**Generated schema adds enums for:**
- `element.kind` → valid values from `spec.elementKinds[*].id`
- `element.tags[]` → valid values from `spec.tags[*].id`
- `element.status` → fixed lifecycle enum
- `relationship.dataFlow` → fixed data flow enum

**Usage in `architecture.jsonc`:**
```jsonc
{
  "$schema": "./.bausteinsicht-schema.json",   // ← add this line
  "spec": { ... },
  "model": {
    "elements": [
      { "id": "svc", "kind": "<TAB>" }   // ← IDE now shows: service, database, frontend
    ]
  }
}
```

**Watch integration:**
```
bausteinsicht watch --schema
# → Regenerates .bausteinsicht-schema.json when spec changes
```

**`init` integration:** `bausteinsicht init` auto-generates the schema and adds `$schema` reference.

## Implementation Plan

See the implementation plan embedded in the GitHub issue.

## Affected Components

- `internal/schema/generator.go` — new
- `cmd/bausteinsicht/schema.go` — new command
- `cmd/bausteinsicht/watch.go` — `--schema` flag
- `cmd/bausteinsicht/init.go` — auto-generate on init
