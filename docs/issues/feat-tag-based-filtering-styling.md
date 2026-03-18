---
title: "feat: Tag-Based View Filtering & Batch Styling"
labels: enhancement
---

## Beschreibung

The `tags` field already exists on elements but has no effect beyond storage. This feature makes tags a first-class mechanism: views can filter elements by tag, and tag definitions in the spec can define visual styles applied automatically during draw.io sync.

## Motivation

- Teams need to show only `deployed` elements in a production view without duplicating the model
- `deprecated` elements should visually stand out (red border, strikethrough) — currently requires manual draw.io editing that gets overwritten on next sync
- Tag-based constraints (`no-relationship-across-tags`) fill a gap in the planned constraints feature
- Tags are already in the schema — this activates existing data

## Proposed Implementation

**Spec extension:**
```jsonc
{
  "spec": {
    "tags": [
      { "id": "deprecated", "style": { "fillColor": "#f8cecc", "fontStyle": "strikethrough" } },
      { "id": "deployed" },
      { "id": "external" }
    ]
  }
}
```

**View filter:**
```jsonc
{ "key": "prod", "include": ["*"], "filter-tags": ["deployed"], "exclude-tags": ["deprecated"] }
```

**Constraint:**
```jsonc
{ "rule": "no-relationship-across-tags", "from-tags": ["frontend"], "to-tags": ["internal-only"] }
```

**Backwards compatible:** Models without `spec.tags` behave identically to today.

## Implementation Plan

See [`docs/plans/2026-03-18-tag-based-filtering-styling.md`](../plans/2026-03-18-tag-based-filtering-styling.md)

## Affected Components

- `internal/model/types.go` — `TagDefinition`, `filter-tags`/`exclude-tags` on `View`
- `internal/model/resolve.go` — `FilterElementsByTags()`
- `internal/sync/forward.go` — apply tag styles during sync
- `internal/constraints/rules.go` — `no-relationship-across-tags` rule
