# Plan: Tag-Based View Filtering & Batch Styling

## Purpose

The `tags` field already exists on elements but is unused beyond storage. This feature activates tags as a first-class filtering and styling mechanism: views can include/exclude elements by tag, and elements with specific tags get automatic visual styles in draw.io (e.g. `deprecated` → red border, strikethrough).

## CLI Interface

No new top-level commands. Extensions to existing commands:

```
bausteinsicht sync   [--model <file>]          # unchanged, but now respects tag styles
bausteinsicht export-diagram --tags deprecated  # filter exported view by tags
```

The `add element` guided prompts and REPL (if implemented) suggest known tags from spec.

## Model Changes

### Tag Definitions in Spec

```jsonc
{
  "spec": {
    "elementKinds": [...],
    "tags": [
      {
        "id": "deprecated",
        "description": "Component is deprecated and will be removed",
        "style": {
          "fillColor": "#f8cecc",
          "strokeColor": "#b85450",
          "fontStyle": "strikethrough"
        }
      },
      {
        "id": "deployed",
        "description": "Currently deployed to production"
      },
      {
        "id": "external",
        "description": "Owned by an external team"
      }
    ]
  }
}
```

### View Filter

```jsonc
{
  "views": [
    {
      "key": "production",
      "title": "Production Components",
      "include": ["*"],
      "filter-tags": ["deployed"],       // only elements WITH these tags
      "exclude-tags": ["deprecated"]     // elements WITH these tags are excluded
    }
  ]
}
```

### Tag-Based Constraints

```jsonc
{
  "constraints": [
    {
      "id": "C-T01",
      "description": "Frontends must not call internal-only services",
      "rule": "no-relationship-across-tags",
      "from-tags": ["frontend"],
      "to-tags": ["internal-only"]
    }
  ]
}
```

## Architecture

### New / Modified Files

| File | Change |
|------|--------|
| `internal/model/types.go` | Add `TagDefinition` struct; add `Tags []TagDefinition` to `Specification`; add `FilterTags []string`, `ExcludeTags []string` to `View` |
| `internal/model/validate.go` | Validate that tags used on elements are defined in spec; validate filter-tag references |
| `internal/model/resolve.go` | New: `FilterElementsByTags(elements, filterTags, excludeTags) []Element` |
| `internal/sync/forward.go` | Apply `TagDefinition.Style` overrides to elements during forward sync |
| `internal/sync/reverse.go` | Preserve tags from draw.io custom properties on roundtrip |
| `internal/constraints/rules.go` | Add `no-relationship-across-tags` rule |

### Tag Style Application Logic

Tag styles are applied as overrides on top of element-kind styles. If multiple tags define styles, they are merged in tag-list order (last wins).

```go
func applyTagStyles(base ElementStyle, tags []string, tagDefs map[string]TagDefinition) ElementStyle {
    result := base
    for _, tag := range tags {
        if def, ok := tagDefs[tag]; ok && def.Style != nil {
            result = mergeStyle(result, *def.Style)
        }
    }
    return result
}
```

### View Filtering

Tag filtering applies after include/exclude pattern resolution:

```
1. Resolve include/exclude patterns → candidate elements
2. If filter-tags set: keep only elements that have ALL filter-tags
3. If exclude-tags set: remove elements that have ANY exclude-tag
4. Relationship endpoint lifting (existing logic) applies after filtering
```

## Backwards Compatibility

- Models without `spec.tags` definitions work as before (tags stored but unstyled)
- Views without `filter-tags`/`exclude-tags` behave identically to current
- Tag constraints only apply when a `constraints` section is present

## Testing

- Unit test: `FilterElementsByTags` with various tag combinations
- Unit test: `applyTagStyles` merges correctly with multiple tags
- Unit test: `no-relationship-across-tags` constraint with violation
- E2E test: view with `filter-tags` only renders tagged elements in draw.io output
- Property-based test: filter + exclude are mutually exclusive subsets
