# Plan: Auto-Layout in draw.io

## Purpose

After forward sync, newly added elements are placed at default positions (often overlapping or off-screen). A `layout` command computes a clean hierarchical or force-directed layout and writes coordinates back into the draw.io file. Manually positioned elements are optionally preserved using a pin mechanism.

## CLI Interface

```
bausteinsicht layout [--model <file>] [--view <key>] [--algorithm hierarchical|force|radial] [--preserve-pinned]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--view` | (all views) | Layout only one specific view/page |
| `--algorithm` | `hierarchical` | Layout algorithm |
| `--preserve-pinned` | true | Don't move elements marked as pinned in draw.io |

Also available as a flag on `sync` and `watch`:

```
bausteinsicht sync  --auto-layout
bausteinsicht watch --auto-layout
```

## Layout Algorithms

### Hierarchical (default)

Arranges elements in layers based on relationship direction. Top-to-bottom for `rankdir=TB`, left-to-right for `rankdir=LR`.

- Layer assignment via longest-path algorithm
- Same-layer elements sorted to minimize edge crossings
- Fixed element dimensions: service 160×60, person 60×80, container 200×120

```
Layer 0 (sources):    [user]  [external-system]
Layer 1 (services):   [api-gateway]
Layer 2 (backends):   [order-service]  [payment-service]
Layer 3 (data):       [order-db]  [user-db]  [message-broker]
```

### Force-Directed

Spring-force simulation: relationships act as springs pulling connected elements together, repulsion keeps unconnected elements apart. Runs for a fixed number of iterations (200) with cooling.

Best for: models without clear hierarchy, circular relationships.

### Radial

Places the scoped element (if a view has a scope) at center; dependent elements radiate outward in concentric rings by distance from center.

Best for: context views and container views with a clear focal element.

## Pin Mechanism

Elements can be pinned in draw.io by setting a custom property `bausteinsicht-pinned=true` via the draw.io "Edit Style" or right-click menu. The layout engine reads this property and skips pinned elements (keeps their current `x`, `y`).

```go
type ElementPosition struct {
    ID     string
    X, Y   float64
    Pinned bool  // read from draw.io cell property "bausteinsicht-pinned"
}
```

## Layout Integration with Sync

When `--auto-layout` is passed to `sync` or `watch`:

1. Forward sync runs (elements added/updated)
2. Layout computes positions for **newly added elements only** (elements that had no prior position)
3. Positions written to draw.io file

This avoids overwriting manually adjusted positions on every sync — only genuinely new elements get auto-positioned.

"New element" detection: element whose draw.io cell has default position (x=10, y=10) or was just created by this sync run.

## Spacing Constants

```go
const (
    HorizontalSpacing = 40   // px between elements in same layer
    VerticalSpacing   = 80   // px between layers
    MarginX           = 60   // px from left edge
    MarginY           = 60   // px from top edge
)
```

## Architecture

### New / Modified Files

| File | Change |
|------|--------|
| `internal/layout/hierarchical.go` | New: layered layout algorithm |
| `internal/layout/force.go` | New: force-directed layout |
| `internal/layout/radial.go` | New: radial layout |
| `internal/layout/types.go` | New: `Graph`, `Node`, `Edge`, `LayoutResult` |
| `internal/layout/pins.go` | New: read/write pinned property from draw.io XML |
| `internal/sync/forward.go` | Apply layout result to draw.io cells after sync |
| `cmd/bausteinsicht/layout.go` | New `layout` command |
| `cmd/bausteinsicht/sync.go` | Add `--auto-layout` flag |
| `cmd/bausteinsicht/watch.go` | Add `--auto-layout` flag |

## Testing

- Unit test: hierarchical layout assigns correct layers for linear chain
- Unit test: pinned elements are not moved
- Unit test: new elements get non-overlapping positions
- Unit test: force-directed converges (positions change between iterations)
- E2E test: `layout` → draw.io cell positions differ from default (10,10)
- Property-based test: no two elements overlap after hierarchical layout
