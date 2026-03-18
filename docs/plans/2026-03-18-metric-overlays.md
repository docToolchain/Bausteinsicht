# Plan: Metric Overlays (Heatmap)

## Purpose

Load external metrics (error rate, CPU usage, deployment frequency, test coverage) from JSON files and overlay them as a heatmap on draw.io elements. Answers the question: "which components are most problematic right now?" — directly in the architecture diagram.

## CLI Interface

```
bausteinsicht overlay apply  <metrics-file> [--model <file>] [--metric <key>] [--output <drawio-file>]
bausteinsicht overlay list   <metrics-file>
```

| Flag | Description |
|------|-------------|
| `<metrics-file>` | JSON file with per-element metric values |
| `--metric <key>` | Which metric to visualize (default: first in file) |
| `--output` | Output draw.io file (default: overwrites model's draw.io file) |

### Example

```bash
# Generate metrics file from CI/monitoring (team's responsibility)
echo '{"metrics": [{"elementId": "order-service", "error_rate": 4.2, "p99_ms": 320}]}' > metrics.json

# Apply overlay
bausteinsicht overlay apply metrics.json --metric error_rate
# → Rewrites draw.io file with heatmap colors; original styles preserved in metadata
```

## Metrics File Format

```json
{
  "meta": {
    "generated": "2026-03-18T10:00:00Z",
    "source": "Prometheus",
    "metric_descriptions": {
      "error_rate": "HTTP 5xx error rate (%)",
      "p99_ms": "P99 response time (ms)",
      "deploy_freq": "Deployments per week",
      "coverage": "Test coverage (%)"
    }
  },
  "metrics": [
    { "elementId": "order-service",   "error_rate": 4.2,  "p99_ms": 320, "deploy_freq": 12, "coverage": 78 },
    { "elementId": "payment-service", "error_rate": 0.1,  "p99_ms": 45,  "deploy_freq": 3,  "coverage": 92 },
    { "elementId": "api-gateway",     "error_rate": 0.8,  "p99_ms": 15,  "deploy_freq": 8,  "coverage": 65 },
    { "elementId": "legacy-monolith", "error_rate": 12.5, "p99_ms": 890, "deploy_freq": 1,  "coverage": 23 }
  ]
}
```

## Heatmap Color Scheme

Values are normalized to [0, 1] across all elements for the selected metric. Color interpolation:

| Normalized Value | Color | Hex |
|-----------------|-------|-----|
| 0.0 – 0.25 | Green | `#d5e8d4` |
| 0.25 – 0.50 | Yellow | `#fff2cc` |
| 0.50 – 0.75 | Orange | `#ffe6cc` |
| 0.75 – 1.0 | Red | `#f8cecc` |

For metrics where lower is better (error rate, p99): high value = red.
For metrics where higher is better (coverage, deploy_freq): high value = green. Direction is auto-detected from `--metric` name or configurable in metrics file.

## Style Preservation

The overlay **does not permanently modify** the model's draw.io file:
- Original element fill colors are saved in draw.io custom property `data-original-fill`
- `bausteinsicht overlay remove` restores original colors from saved properties
- Re-running `sync` after `overlay apply` triggers a warning: "overlay active — sync will remove overlay"

## Architecture

### New / Modified Files

| File | Change |
|------|--------|
| `internal/overlay/types.go` | New: `MetricsFile`, `ElementMetric`, `NormalizedMetric` |
| `internal/overlay/normalize.go` | New: normalize values across elements, compute colors |
| `internal/overlay/apply.go` | New: read draw.io file, apply/remove fill color overrides |
| `cmd/bausteinsicht/overlay.go` | New `overlay` command with `apply`, `remove`, `list` subcommands |

### Normalization Algorithm

```go
func Normalize(values []float64, higherIsBetter bool) []float64 {
    min, max := slices.Min(values), slices.Max(values)
    span := max - min
    if span == 0 { return make([]float64, len(values)) }
    result := make([]float64, len(values))
    for i, v := range values {
        normalized := (v - min) / span
        if higherIsBetter { normalized = 1 - normalized }
        result[i] = normalized
    }
    return result
}
```

## Integration with CI/CD

Teams can generate `metrics.json` from any monitoring system and call `overlay apply` in a report job:

```yaml
- name: Architecture Heatmap
  run: |
    curl -s https://prometheus/api/query?query=error_rate | \
      jq '{metrics: [.data.result[] | {elementId: .metric.service, error_rate: (.value[1]|tonumber)}]}' \
      > metrics.json
    bausteinsicht overlay apply metrics.json --metric error_rate --output reports/architecture-heatmap.drawio
```

## Testing

- Unit test: `Normalize` with known values → verify color assignment
- Unit test: `higherIsBetter` flag inverts color mapping
- Unit test: single-value edge case (all same → all green)
- Unit test: original style preservation / restore
- E2E test: apply → draw.io contains modified fill colors; remove → original colors restored
