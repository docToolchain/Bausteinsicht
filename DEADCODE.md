# Dead Code Detection

Bausteinsicht uses the `deadcode` analyzer to identify and track unused code, helping maintain a clean and manageable codebase.

## Quick Start

Run dead code analysis:

```bash
make deadcode
make install-tools  # if deadcode is not yet installed
```

## Baseline Analysis

Dead code analysis is run as a reporting tool (does not fail CI). Current report (v1.0+):

| Function | File | Status | Reason |
|----------|------|--------|--------|
| GenerateActorLabel | internal/drawio/label.go | Unreachable | Legacy export code, kept for backward compatibility |
| ImportSource (LikeC4) | internal/importer/likec4/likec4.go | Unreachable | Internal utility, safe to remove in v2 |
| ImportSource (Structurizr) | internal/importer/structurizr/structurizr.go | Unreachable | Internal utility, safe to remove in v2 |
| GenerateCodeLens | internal/lsp/codelens.go | Unreachable | LSP extension code, kept for future integration |
| extractKind | internal/lsp/codelens.go | Unreachable | LSP helper, kept for future use |
| extractStatus | internal/lsp/codelens.go | Unreachable | LSP helper, kept for future use |
| estimateViewCount | internal/lsp/codelens.go | Unreachable | LSP helper, kept for future use |
| AddStatusBadge | internal/sync/badge.go | Unreachable | Status badge feature, kept for future use |
| getAttr | internal/sync/badge.go | Unreachable | Helper for badge feature |
| getAttrFloat | internal/sync/badge.go | Unreachable | Helper for badge feature |

## How to Handle Dead Code

### 1. Identify (Run deadcode)

```bash
make deadcode
```

This lists all unreachable functions (functions with no callers).

### 2. Classify

Each unreachable function is either:

- **Legacy/Removed Feature**: Function was part of an older feature and can be safely deleted
- **Future Feature**: Function is intentionally kept for planned features (mark with `//nolint:deadcode`)
- **Public API**: Exported function that is part of the public contract (document why it's kept)

### 3. Act

**Remove truly dead code:**
```go
// Before
func unusedHelper() { ... }  // no callers, not in public API

// After
// (delete the function)
```

**Mark intentional dead code:**
```go
// Before
func FutureFeatureHelper() { ... }

// After (keep for v2 feature implementation)
//nolint:deadcode
func FutureFeatureHelper() { ... }
```

## Guidelines

### Safe to Remove

- Unused internal functions (not in `internal/*/`'s public API)
- Unused private functions (lowercase)
- Functions that were part of a feature that was removed or refactored

### Safe to Keep

- Exported functions that are part of the public API (upper case)
- Test helpers (even if unused, sometimes provided for external test libraries)
- Functions marked with `//nolint:deadcode` (deliberate keeper)
- Functions that may be called via reflection (rare, document why)

### Examples

```go
// REMOVE: internal helper with no callers
func parseInternalFormat(data string) { ... }

// KEEP: public API (part of public model package)
func (e Element) Tags() []string { ... }

// KEEP: marked as intentional
//nolint:deadcode
func futureLayoutAlgorithm() { ... }
```

## CI Integration

Dead code analysis is run as a **reporting tool** (informational, does not block merges). Future versions may:

- Fail on new dead code (opt-in via CI flag)
- Track dead code metrics over time
- Suggest removal candidates

## Tools

- **Tool**: [golang.org/x/tools/cmd/deadcode](https://pkg.go.dev/golang.org/x/tools/cmd/deadcode)
- **Docs**: [Dead Code Analysis in Go](https://pkg.go.dev/golang.org/x/tools)

## Related

- [Code Quality Metrics](BENCHMARKS.md) — Performance and quality tracking
- [Testing Strategy](src/docs/arc42/ADRs/ADR-006-Testing-Strategy.adoc) — How tests relate to code quality
