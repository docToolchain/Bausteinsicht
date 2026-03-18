# Plan: Element Search / `find` Command

## Purpose

A `find` command that searches all elements, relationships, and views by free-text query across all fields (ID, title, description, technology, tags, kind). Returns ranked results as a table or JSON — useful in large models and essential for LLM-driven workflows where an agent needs to locate elements by name without knowing their IDs.

## CLI Interface

```
bausteinsicht find <query> [--model <file>] [--type element|relationship|view|all] [--format text|json]
```

| Flag | Default | Description |
|------|---------|-------------|
| `<query>` | (required) | Free-text search term (case-insensitive, partial match) |
| `--type` | `all` | Limit search to specific object type |
| `--format` | `text` | Output format |

### Examples

```bash
bausteinsicht find payment
bausteinsicht find "order service" --type element
bausteinsicht find grpc --type relationship
bausteinsicht find context --type view
```

## Search Algorithm

Fields searched per object type:

### Elements
| Field | Weight |
|-------|--------|
| `id` | 3 (exact match: 10) |
| `title` | 3 |
| `description` | 1 |
| `technology` | 2 |
| `tags[]` | 2 |
| `kind` | 2 |
| `status` | 1 |

### Relationships
| Field | Weight |
|-------|--------|
| `id` | 3 |
| `label` | 3 |
| `protocol` | 2 |
| `dataFlow` | 1 |
| `from` / `to` (element titles) | 2 |

### Views
| Field | Weight |
|-------|--------|
| `key` | 3 |
| `title` | 3 |
| `description` | 1 |

**Scoring:** each field match contributes its weight to the total score. Results sorted by descending score. Ties broken alphabetically by ID.

**Partial match:** `"pay"` matches `"payment-service"`, `"payment"`, `"payroll"`.

**Multi-word query:** `"order service"` matches elements containing both "order" AND "service" in any field (AND semantics).

## Output (text format)

```
Search results for "payment" (4 matches)
=========================================

Elements (3):
  payment-service   [service]    "Payment Service v2"       technology: Go          score: 18
  payment-db        [database]   "Payment Database"          technology: PostgreSQL  score: 12
  legacy-payment    [service]    "Legacy Payment Handler"    status: deprecated      score:  9

Relationships (1):
  r-order-payment   order-service → payment-service   "charges via"   protocol: gRPC   score: 6
```

## Output (JSON format)

```json
{
  "query": "payment",
  "results": [
    {
      "type": "element",
      "id": "payment-service",
      "title": "Payment Service v2",
      "kind": "service",
      "score": 18,
      "matchedFields": ["id", "title", "technology"]
    }
  ],
  "total": 4
}
```

## LLM Integration

The JSON output is designed for LLM agent consumption:

```bash
# Agent workflow: "find the payment service and show its relationships"
bausteinsicht find payment --type element --format json | \
  jq '.results[0].id' | \
  xargs -I{} bausteinsicht show {}
```

## `show` Subcommand

`find` is complemented by a `show` command (also new) that displays full details of one element:

```bash
bausteinsicht show payment-service
```

```
Element: payment-service
========================
Kind:        service
Title:       Payment Service v2
Description: Handles all payment processing via external gateway
Technology:  Go
Status:      deployed
Tags:        [core, pci-dss]
Decisions:   [ADR-012]

Relationships:
  ← order-service     "charges via"  [gRPC, 1:N]
  → payment-gateway   "processes"    [HTTPS, 1:1]
  → audit-log         "records"      [AMQP, async]

Views containing this element:
  context-view, containers-view, payment-domain
```

## Architecture

### New / Modified Files

| File | Change |
|------|--------|
| `internal/search/search.go` | New: `Search(query, model, opts) []SearchResult` |
| `internal/search/scorer.go` | New: field-weighted scoring per object type |
| `internal/search/types.go` | New: `SearchResult`, `SearchOptions` |
| `cmd/bausteinsicht/find.go` | New `find` command |
| `cmd/bausteinsicht/show.go` | New `show` command |

## Testing

- Unit test: `"payment"` matches `payment-service` with score > `legacy-payment`
- Unit test: exact ID match scores highest
- Unit test: multi-word query requires all words present
- Unit test: `--type element` excludes relationships from results
- E2E test: `find grpc --format json` returns relationship with protocol gRPC
- Test: empty result set returns gracefully with total: 0
