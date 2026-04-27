---
title: "feat: Element Search / find Command"
labels: enhancement
---

## Beschreibung

Ein `find` Befehl durchsucht alle Elemente, Beziehungen und Views per Freitext-Query über alle Felder (ID, Titel, Beschreibung, Technologie, Tags, Kind, Status). Ergebnisse werden nach Relevanz-Score sortiert ausgegeben. Ergänzt durch einen `show` Befehl für vollständige Elementdetails.

## Motivation

- In Modellen mit 50+ Elementen ist das manuelle Suchen in JSON mühsam
- LLM-Agenten brauchen eine zuverlässige Methode, Elemente per Name zu finden ohne ihre ID zu kennen
- `bausteinsicht find payment` ist schneller als `grep -r "payment" architecture.jsonc`
- JSON-Output (`--format json`) macht `find` zum Baustein für komplexe Agent-Workflows

## Proposed Implementation

**New commands:**
```
bausteinsicht find <query> [--type element|relationship|view|all] [--format text|json]
bausteinsicht show <element-id> [--format text|json]
```

**Search fields per object type:**
- Elements: id (3×), title (3×), description (1×), technology (2×), tags (2×), kind (2×)
- Relationships: id (3×), label (3×), protocol (2×), from/to-titles (2×)
- Views: key (3×), title (3×), description (1×)

**Multi-word:** `"order service"` = AND-Suche über alle Felder.

**Output:**
```
Search results for "payment" (4 matches)

Elements (3):
  payment-service  [service]   "Payment Service v2"    score: 18
  payment-db       [database]  "Payment Database"      score: 12

Relationships (1):
  r-order-payment  order-service → payment-service  "charges via"  score: 6
```

**`show` Output:**
```
Element: payment-service
Kind: service | Status: deployed | Technology: Go
Relationships: ← order-service, → payment-gateway, → audit-log
Views: context-view, containers-view
ADRs: ADR-012
```

## Implementation Plan

See [`docs/plans/2026-03-18-element-search-find.md`](../plans/2026-03-18-element-search-find.md)

## Affected Components

- `internal/search/` — new package (search, scorer)
- `cmd/bausteinsicht/find.go` — new command
- `cmd/bausteinsicht/show.go` — new command
