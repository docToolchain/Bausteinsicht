# Plan: Multi-Model Workspace

## Purpose

Allow merging multiple `architecture.jsonc` files (e.g. one per team, bounded context, or domain) into a unified workspace view. Teams own their local model; a workspace config references all team models and generates cross-team views showing inter-domain relationships.

## CLI Interface

```
bausteinsicht workspace init     [--output workspace.jsonc]
bausteinsicht workspace merge    [--workspace <file>] [--output <drawio-file>]
bausteinsicht workspace validate [--workspace <file>]
bausteinsicht workspace list     [--workspace <file>]
```

### Example

```bash
bausteinsicht workspace merge --workspace workspace.jsonc
# → Merges all referenced models, generates workspace-architecture.drawio
```

## Workspace Config Format

```jsonc
// workspace.jsonc
{
  "workspace": {
    "name": "E-Commerce Platform",
    "models": [
      { "id": "orders",   "path": "teams/orders/architecture.jsonc",   "prefix": "orders" },
      { "id": "payments", "path": "teams/payments/architecture.jsonc", "prefix": "payments" },
      { "id": "catalog",  "path": "teams/catalog/architecture.jsonc",  "prefix": "catalog" }
    ]
  },
  "views": [
    {
      "key": "platform-context",
      "title": "Platform Context",
      "include-from": ["orders", "payments", "catalog"],
      "include-kinds": ["system", "service"],   // only top-level kinds
      "show-cross-model-relationships": true
    }
  ],
  "crossModelRelationships": [
    {
      "id": "xr-001",
      "from": "orders/order-service",    // prefixed element ID
      "to": "payments/payment-service",
      "label": "charges via"
    }
  ]
}
```

## ID Namespacing

Each model's elements are prefixed with the model's `id` to avoid collisions:
- `orders/order-service` (from orders model)
- `payments/payment-service` (from payments model)

Elements already prefixed in cross-model relationships are resolved at merge time.

## Merge Algorithm

1. Load all referenced model files
2. Prefix all element and relationship IDs with model `id`
3. Merge `spec.elementKinds` (deduplicate by `id`)
4. Merge `model.elements` and `model.relationships`
5. Add `crossModelRelationships` to merged relationships
6. Resolve workspace views against merged element set
7. Run standard forward sync to generate draw.io output

## Validation

- Warn if a model path does not exist
- Error if two models define conflicting element kinds with different properties
- Warn if a cross-model relationship references a non-existent prefixed element ID
- Validate that workspace views only reference valid model IDs in `include-from`

## View Filtering for Workspace Views

Workspace views support `include-kinds` to show only high-level elements (avoiding low-level noise in cross-team views):

```jsonc
{
  "key": "domain-overview",
  "include-from": ["orders", "payments"],
  "include-kinds": ["system"],         // show only system-level elements
  "exclude-kinds": ["database", "cache"]
}
```

## File Layout (Recommended)

```
platform/
├── workspace.jsonc                    ← workspace config
├── workspace-architecture.drawio      ← generated merged diagram
├── teams/
│   ├── orders/
│   │   ├── architecture.jsonc         ← orders team model
│   │   └── architecture.drawio        ← orders team diagram
│   ├── payments/
│   │   ├── architecture.jsonc
│   │   └── architecture.drawio
│   └── catalog/
│       ├── architecture.jsonc
│       └── architecture.drawio
```

## Architecture

### New / Modified Files

| File | Change |
|------|--------|
| `internal/workspace/types.go` | New: `WorkspaceConfig`, `ModelRef`, `CrossModelRelationship` |
| `internal/workspace/loader.go` | New: load and prefix all referenced models |
| `internal/workspace/merge.go` | New: `MergeModels(refs []LoadedModel) BausteinsichtModel` |
| `internal/workspace/validate.go` | New: cross-model reference validation |
| `cmd/bausteinsicht/workspace.go` | New `workspace` command with subcommands |

## Testing

- Unit test: `MergeModels` with two models → verify ID prefixing
- Unit test: conflicting kind definitions → error
- Unit test: cross-model relationship resolves correctly after prefix
- E2E test: workspace with 3 models → `merge` → draw.io contains elements from all models
- Test: missing model path → clear error message
