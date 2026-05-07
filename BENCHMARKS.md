# Performance Benchmarks

Bausteinsicht benchmarks critical code paths to detect performance regressions and track optimization opportunities across releases.

## Quick Start

Run benchmarks locally:

```bash
go test ./internal/benchmarks -bench=. -benchmem -benchtime=5s
```

This runs all benchmarks with memory allocation stats (`-benchmem`), executing each for at least 5 seconds to stabilize results.

## Baseline Metrics (Linux 2.9GHz)

Current baseline (from `benchmarks/main.txt`):

| Benchmark | Time/op | Allocs/op | Notes |
|-----------|---------|-----------|-------|
| ModelLoad (100 elems) | 474μs | 1,348 | JSONC parsing + JSON unmarshal |
| ModelLoadLarge (500 elems) | 2.6ms | 6,556 | Stress test: linear scaling |
| ModelValidation (100 elems) | 214μs | 631 | Full validation pipeline |
| ModelValidationLarge (500 elems) | 1.4ms | 3,796 | Scales linearly with elements |
| FlattenElements (100 elems) | 29.5μs | 111 | Recursive hierarchy traversal |
| DiagramFormatView (50 elems) | 45.9μs | 172 | C4 diagram generation |
| TableExportMarkdown | 59.6μs | 227 | Markdown table rendering |
| TableExportAsciidoc | 62.5μs | 227 | AsciiDoc table rendering |
| RoundTripJSON | 764μs | 1,560 | Marshal + unmarshal cycle |
| FileWrite (100 elems) | 476μs | 229 | Save to disk |
| FileRead (100 elems) | 1.8ms | 1,454 | Load from disk (I/O bound) |
| ValidateAndFlatten (100 elems) | 263μs | 742 | Combined operation |

**Hardware**: Intel Pentium G4400T @ 2.9GHz (devcontainer baseline)

**Go Version**: 1.25+ (from go.mod)

## CI Integration

### On Every Push to Main
- Run benchmarks with current code
- Store results in `benchmarks/main.txt`
- Automatically commit baseline updates (if changed)

### On Every PR
- Run benchmarks
- Compare against baseline from main
- Flag regressions >5% in PR summary
- **No block**: regressions are informational (allow merge with justification)

## Regression Interpretation

A regression >5% indicates:

### Likely Causes (Investigate)
- Added complexity to hot path (e.g., extra validation loop)
- Inefficient data structure (e.g., repeated map lookups)
- Missed optimization (e.g., allocation that could be pooled)

### False Positives (Common)
- **Variance**: Benchmarks have ±3-5% variance due to system load; re-run locally to confirm
- **Hardware differences**: CI machine may differ from dev machine
- **Go GC variance**: Garbage collection timing can shift results ±10%

### Response Strategy

If PR shows >5% regression:

1. **Run locally** — confirm on your machine:
   ```bash
   go test ./internal/benchmarks -bench=BenchmarkModelLoad -benchmem
   ```

2. **Profile if real** — use pprof to locate bottleneck:
   ```bash
   go test -cpuprofile=cpu.prof -bench=BenchmarkModelLoad ./internal/benchmarks
   go tool pprof cpu.prof
   ```

3. **Optimize or justify** — either:
   - Optimize: reduce allocations, cache results, etc.
   - Justify: "Regression is acceptable because [reason]" in PR comment

4. **Re-run**: After optimization, re-run benchmarks:
   ```bash
   go test ./internal/benchmarks -bench=. -benchmem
   ```

## Adding New Benchmarks

When optimizing or adding features, add benchmarks for:

### Hot Paths (Required)
- Model loading and validation (frequent operation)
- Sync detection and apply (critical for interactive use)
- Export rendering (user-facing latency)

### Less Critical (Optional)
- File I/O (system-dependent, hard to optimize)
- Table formatting (fast enough that optimizations rarely matter)

### Benchmark Template

```go
// BenchmarkNewOperation benchmarks the new operation
func BenchmarkNewOperation(b *testing.B) {
	setup := prepareTestData()  // Do setup outside the timer
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewOperation(setup)
	}
}
```

**Guidelines:**
- All setup happens before `b.ResetTimer()`
- Only timed code goes inside the loop
- Use `-benchmem` to identify allocations
- Test both typical (100 elem) and stress (500 elem) cases

## Performance SLOs

Target runtimes per operation:

| Operation | SLO | Status |
|-----------|-----|--------|
| Model load | <1ms for 100 elems | ✅ 474μs |
| Model validation | <500μs for 100 elems | ✅ 214μs |
| Diagram export | <100μs per view | ✅ 46μs |
| Table export | <100μs | ✅ 60μs |
| File I/O | No hard SLO (system-dependent) | ℹ️ 1-2ms |

All SLOs met on current baseline.

## Optimization Opportunities

### Known Hot Paths (Priority)
1. **Model validation** — currently 214μs for 100 elems; recursive descent could be optimized with concurrent validation
2. **File I/O** — 1.8ms for read is I/O-bound; use memory-mapped files for very large models (>1000 elems)
3. **JSON marshaling** — 474μs; struct tags could be optimized or replaced with manual serialization for critical paths

### Investigation Needed
- Allocation rate: 1,348 allocations for 100-elem model suggests opportunity for pooling
- GC pressure: Benchmarks run ~7,700 iterations without issue; GC not a bottleneck

## Regression Detection in CI

GitHub Actions workflow (`.github/workflows/benchmarks.yml`) runs on every PR:

```
1. Run benchmarks on PR branch
2. Download baseline from main
3. Run `benchstat` to compare
4. Post summary (regression alert if >5%)
5. Update baseline on main push
```

**Note**: This is informational only. Regressions are flagged but PRs can still merge with justification (performance is secondary to correctness).

## Further Reading

- [Go Benchmarking Guide](https://pkg.go.dev/testing#hdr-Benchmarks)
- [benchstat Documentation](https://github.com/golang/perf/tree/master/cmd/benchstat)
- [Performance Analysis Debugging](https://www.brendangregg.com/profiling.html) (Brendan Gregg)
