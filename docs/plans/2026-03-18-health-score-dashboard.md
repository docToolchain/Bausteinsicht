# Plan: Health Score Dashboard

## Purpose

Aggregate multiple quality signals (constraint violations, lifecycle status, ADR orphans, stale elements, metric thresholds) into a single architecture health score (0–100) with a per-category breakdown. Suitable as a CI gate and architecture review starting point.

## CLI Interface

```
bausteinsicht health [--model <file>] [--metrics <file>] [--format text|json] [--fail-below <score>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--metrics` | (none) | Optional metrics JSON file for runtime health signals |
| `--fail-below` | (none) | Exit code 1 if score drops below threshold (for CI) |
| `--format` | `text` | Output format |

### Example Output (text)

```
Architecture Health Score
=========================
Overall: 74 / 100  🟡

Category Breakdown:
  Constraints         20/25  ✅  1 violation (C-002: missing descriptions)
  Lifecycle           18/25  🟡  2 deprecated elements without successors
  Decision Coverage   12/20  🟡  3 elements without any ADR link
  Metric Thresholds   24/30  ✅  all metrics within bounds  (requires --metrics)

Top Issues:
  ❌ [constraints]  C-002: payment-service, order-service missing description
  ⚠  [lifecycle]   legacy-monolith: deprecated, no deployed successor
  ⚠  [lifecycle]   auth-service: deprecated, no deployed successor
  ⚠  [decisions]   web-frontend, api-gateway, user-db: no ADR linked

Recommendation: address constraint violations first (highest weight, easiest fix).
```

### Example Output (JSON)

```json
{
  "score": 74,
  "max": 100,
  "grade": "C",
  "categories": {
    "constraints":        { "score": 20, "max": 25, "issues": [...] },
    "lifecycle":          { "score": 18, "max": 25, "issues": [...] },
    "decision_coverage":  { "score": 12, "max": 20, "issues": [...] },
    "metric_thresholds":  { "score": 24, "max": 30, "issues": [] }
  },
  "top_issues": [...]
}
```

## Scoring Model

### Category Weights

| Category | Max Points | Requires |
|----------|-----------|---------|
| Constraints | 25 | `constraints` section in model |
| Lifecycle | 25 | `status` field on any element |
| Decision Coverage | 20 | `spec.decisions` section |
| Metric Thresholds | 30 | `--metrics` flag |

Categories without data score their maximum (not penalized for unused features).

### Penalty Calculation

Each category starts at max and deducts per issue:

```
constraints:  -5 per violation (capped at -25)
lifecycle:    -3 per deprecated-without-successor, -5 per archived-with-relationships
decisions:    -2 per element without any ADR link (capped at -20)
metrics:      -3 per element exceeding warning threshold, -6 per critical threshold
```

### Grade Table

| Score | Grade | Emoji |
|-------|-------|-------|
| 90–100 | A | 🟢 |
| 75–89  | B | 🟢 |
| 60–74  | C | 🟡 |
| 40–59  | D | 🟡 |
| 0–39   | F | 🔴 |

## CI Integration

```yaml
- name: Architecture Health Check
  run: bausteinsicht health --metrics metrics.json --fail-below 70 --format json | tee health.json

- name: Post Health Score to PR
  if: always()
  uses: actions/github-script@v7
  with:
    script: |
      const h = JSON.parse(require('fs').readFileSync('health.json'));
      github.rest.issues.createComment({
        issue_number: context.issue.number,
        body: `## Architecture Health: ${h.score}/100 (${h.grade})\n\nSee details in CI artifacts.`
      });
```

## Architecture

### New / Modified Files

| File | Change |
|------|--------|
| `internal/health/scorer.go` | New: `Score(model, metrics) HealthReport` |
| `internal/health/categories.go` | New: one scorer function per category |
| `internal/health/types.go` | New: `HealthReport`, `CategoryScore`, `HealthIssue` |
| `cmd/bausteinsicht/health.go` | New `health` command |

### Extensibility

Category scorers implement an interface, making it easy to add new categories:

```go
type CategoryScorer interface {
    Name() string
    MaxPoints() int
    Score(model *model.BausteinsichtModel, ctx ScoringContext) CategoryScore
}
```

## Testing

- Unit test: perfect model scores 100
- Unit test: each penalty rule deducts correct points
- Unit test: score is capped at 0 (never negative)
- Unit test: categories without data don't penalize
- E2E test: `--fail-below 80` exits 1 when score is 74
- Property-based test: score is always in [0, 100]
