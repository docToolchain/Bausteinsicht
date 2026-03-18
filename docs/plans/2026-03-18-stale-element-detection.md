# Plan: Stale Element Detection

## Purpose

Detect elements that have not been referenced in git commits for a configurable period, have no ADR link, no test coverage link, and no lifecycle status. These are likely forgotten components — either undocumented active services or candidates for archiving. A new `stale` command reports them and optionally marks them in draw.io.

## CLI Interface

```
bausteinsicht stale [--model <file>] [--days <n>] [--format text|json] [--mark-drawio]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--days` | 90 | Elements not touched in git for this many days are considered stale |
| `--format` | `text` | Output format |
| `--mark-drawio` | false | Add visual stale indicator to draw.io elements |

### Example Output (text)

```
Stale Elements (not updated in 90+ days, no ADR/test link)
===========================================================

legacy-auth     [service]   Last changed: 2025-09-12 (187 days ago)
                            No lifecycle status set
                            No ADR linked
                            ⚠ Has 3 incoming relationships — may still be active

report-service  [service]   Last changed: 2025-10-01 (168 days ago)
                            Status: deployed
                            No ADR linked

legacy-importer [service]   Last changed: 2025-08-03 (227 days ago)
                            No lifecycle status set
                            No ADR linked
                            ✅ No incoming relationships — safe to archive

3 stale elements found.

Suggested actions:
  - Set status: "deprecated" or "archived" if no longer needed
  - Add ADR link if the element has a documented decision
  - Run 'bausteinsicht snapshot save' before archiving elements
```

## Staleness Criteria

An element is flagged as stale when ALL of the following are true:

1. **Git age:** The model file containing the element was not modified in the last `--days` days (checked via `git log`)
2. **No status:** The element has no `status` field set
3. **No ADR:** The element has no `decisions` field set

Elements with `status: archived` are excluded (already handled).

## Risk Assessment

For each stale element, the command assesses removal risk:

| Signal | Risk |
|--------|------|
| Has incoming relationships | Medium — other elements depend on it |
| Has outgoing relationships only | Low — it depends on others, not vice versa |
| No relationships | Low — isolated element, safe to archive |
| Referenced in view `include` patterns | Medium — explicitly included in diagrams |

## draw.io Marking (`--mark-drawio`)

When `--mark-drawio` is set, stale elements receive a visual indicator:
- Grey diagonal stripes overlay on element fill
- Small 💤 icon badge in top-left corner
- draw.io tooltip: "Stale: not updated since <date>"

Marking is non-destructive (stored in draw.io metadata; restored by `sync`).

## Git Integration

```go
func lastModifiedDate(modelPath, elementID string) (time.Time, error) {
    // git log --follow -1 --format=%aI -- <modelPath>
    // Returns date of last commit touching the model file
    // Note: per-element git history requires JSON diff parsing (future enhancement)
    // Initial implementation: use model file date as proxy
}
```

Initial implementation uses the model file's last git commit date as a proxy for all elements. A future enhancement could parse JSON diffs to track per-element change dates.

## Configuration in Model

Stale detection thresholds can be configured in model metadata:

```jsonc
{
  "meta": {
    "staleDetection": {
      "thresholdDays": 90,
      "excludeKinds": ["database", "infra"],   // never flag infrastructure as stale
      "excludeTags": ["external"]               // never flag external systems
    }
  }
}
```

## Architecture

### New / Modified Files

| File | Change |
|------|--------|
| `internal/stale/detector.go` | New: `Detect(model, gitLog, config) []StaleElement` |
| `internal/stale/git.go` | New: `GetLastModifiedDate(modelPath string) (time.Time, error)` |
| `internal/stale/risk.go` | New: `AssessRisk(element, model) RiskLevel` |
| `internal/stale/types.go` | New: `StaleElement`, `RiskLevel`, `StaleConfig` |
| `cmd/bausteinsicht/stale.go` | New `stale` command |

## Testing

- Unit test: element without status/ADR beyond threshold → flagged
- Unit test: element with status set → not flagged
- Unit test: `excludeKinds` config prevents flagging database elements
- Unit test: risk assessment — incoming relationships → Medium risk
- E2E test: `stale --days 0` flags all elements without status/ADR (threshold 0 = always stale)
- Test: `--mark-drawio` adds badge cells to stale elements in draw.io output
