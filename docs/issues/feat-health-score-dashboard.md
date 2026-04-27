---
title: "feat: Architecture Health Score Dashboard"
labels: enhancement
---

## Beschreibung

Aggregate multiple quality signals (constraint violations, lifecycle status, ADR coverage, metric thresholds) into a single architecture health score (0–100) with per-category breakdown. Designed as a CI gate and architecture review entry point.

## Motivation

- Teams need a single number to answer "how healthy is our architecture?" in CI and reviews
- Currently, each signal (lint, lifecycle, metrics) must be checked separately
- A composite score makes architecture quality visible in PRs and dashboards
- Builds on top of planned features (constraints, lifecycle, ADR, metrics) — zero additional model changes needed

## Proposed Implementation

**New command:**
```
bausteinsicht health [--metrics metrics.json] [--fail-below 70] [--format text|json]
```

**Scoring (0–100):**
| Category | Max | Signal source |
|----------|-----|--------------|
| Constraints | 25 | `bausteinsicht lint` violations |
| Lifecycle | 25 | deprecated-without-successor, archived-with-relationships |
| Decision Coverage | 20 | elements without any ADR link |
| Metric Thresholds | 30 | optional `--metrics` file |

**CI integration:**
```yaml
- run: bausteinsicht health --fail-below 70 --format json | tee health.json
```

**Grade table:** A (90+), B (75+), C (60+), D (40+), F (<40)

**Extensible:** category scorers implement an interface — new categories can be added independently.

## Implementation Plan

See the implementation plan embedded in the GitHub issue.

## Affected Components

- `internal/health/` — new package
- `cmd/bausteinsicht/health.go` — new command
- Composes: constraints engine, lifecycle validation, ADR validation, metric overlay
