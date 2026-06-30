#!/usr/bin/env bash
# scripts/coverage-report.sh — side-by-side unit vs. E2E coverage report
#
# Usage:
#   scripts/coverage-report.sh            # prints Markdown to stdout
#   make coverage-report                  # writes coverage-report.md
#
# Requirements: Go 1.20+ (for 'go build -cover' and 'go tool covdata')
#
# How it works:
#   1. Unit coverage: go test -coverprofile with ./internal/... ./cmd/...
#   2. E2E coverage:  build a -cover binary, run ./e2e/ with GOCOVERDIR set,
#      convert raw data with 'go tool covdata textfmt'
#   3. Parse per-package stats and emit a side-by-side Markdown table

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

cd "$ROOT"

MODULE="github.com/docToolchain/Bausteinsicht"

# ── 1. Unit test coverage ──────────────────────────────────────────────────────
echo "==> Unit tests with coverage..." >&2
go test \
  -coverprofile="$WORK/unit.cov" \
  -covermode=atomic \
  ./internal/... ./cmd/... \
  >/dev/null 2>&1 || true

if [ ! -s "$WORK/unit.cov" ]; then
  echo "WARNING: unit coverage file is empty or missing" >&2
fi

# ── 2. E2E coverage via instrumented binary ────────────────────────────────────
E2E_COV="$WORK/e2e.cov"
touch "$E2E_COV"

if [ -d "e2e" ]; then
  echo "==> Building coverage-instrumented binary..." >&2
  go build -cover -o "$WORK/bausteinsicht-cov" ./cmd/bausteinsicht

  mkdir -p "$WORK/e2ecov"
  echo "==> E2E tests against instrumented binary (GOCOVERDIR set)..." >&2
  BAUSTEINSICHT_E2E_BIN="$WORK/bausteinsicht-cov" \
  GOCOVERDIR="$WORK/e2ecov" \
    go test ./e2e/ -timeout 300s -count=1 >/dev/null 2>&1 || true

  # Convert raw binary coverage data to go-cover text format
  # Go writes covmeta.* and covcounters.* files (not *.cov), so check for any files.
  if [ -n "$(ls -A "$WORK/e2ecov" 2>/dev/null)" ]; then
    go tool covdata textfmt -i="$WORK/e2ecov" -o="$E2E_COV" 2>/dev/null || true
  else
    echo "WARNING: no E2E coverage data produced (GOCOVERDIR may be empty)" >&2
  fi
else
  echo "INFO: no e2e/ directory found — skipping E2E coverage" >&2
fi

# ── 3. Parse per-package coverage ─────────────────────────────────────────────
# Coverage profile format (go test -coverprofile / go tool covdata textfmt):
#   mode: atomic
#   github.com/…/pkg/file.go:start,end  numstmt  count
#
# We compute statement-weighted coverage per package (same method Go uses for
# its own "total:" line), avoiding the unweighted-function-mean bias that
# inflates or deflates packages with few large functions vs. many small ones.

pkg_coverage() {
  local file="$1"
  [ -s "$file" ] || return 0
  awk -v mod="$MODULE/" '
    NR == 1 { next }
    {
      path = $1
      sub(/:.*/, "", path)
      sub(mod "/", "", path)
      n = split(path, parts, "/")
      pkg = ""
      for (i = 1; i < n; i++) pkg = (pkg == "" ? parts[i] : pkg "/" parts[i])
      if (pkg == "") pkg = path
      total[pkg]   += $2
      if ($3 > 0) covered[pkg] += $2
    }
    END {
      for (p in total)
        printf "%s %.4f\n", p, (total[p] > 0 ? covered[p]/total[p]*100 : 0)
    }
  ' "$file"
}

unit_data=$(pkg_coverage "$WORK/unit.cov")
e2e_data=$(pkg_coverage "$E2E_COV")

# Union of all packages
all_pkgs=$(printf "%s\n%s" "$unit_data" "$e2e_data" \
  | awk 'NF==2 {print $1}' | sort -u)

# ── 4. Emit Markdown table ─────────────────────────────────────────────────────
{
  echo "# Coverage Report — Unit vs. E2E"
  echo ""
  echo "Generated: $(date -u '+%Y-%m-%dT%H:%M:%SZ')"
  echo ""
  echo "| Package | Unit % | E2E % | Gap |"
  echo "|---------|-------:|------:|----:|"

  while IFS= read -r pkg; do
    [ -z "$pkg" ] && continue
    unit=$(printf "%s" "$unit_data" | awk -v p="$pkg" '$1==p {printf "%.1f", $2}')
    e2e=$(printf "%s" "$e2e_data"  | awk -v p="$pkg" '$1==p {printf "%.1f", $2}')
    [ -z "$unit" ] && unit="—"
    [ -z "$e2e"  ] && e2e="—"
    if [ "$unit" != "—" ] && [ "$e2e" != "—" ]; then
      gap=$(awk "BEGIN {printf \"%+.1f\", $e2e - $unit}")
    else
      gap="—"
    fi
    echo "| \`$pkg\` | $unit | $e2e | $gap |"
  done <<< "$all_pkgs"

  echo ""
  echo "## Totals"
  echo ""
  unit_total="—"
  e2e_total="—"
  if [ -s "$WORK/unit.cov" ]; then
    unit_total=$(go tool cover -func="$WORK/unit.cov" 2>/dev/null \
      | awk '/^total:/ {print $NF}')
  fi
  if [ -s "$E2E_COV" ]; then
    e2e_total=$(go tool cover -func="$E2E_COV" 2>/dev/null \
      | awk '/^total:/ {print $NF}')
  fi
  echo "| Layer | Total coverage |"
  echo "|-------|---------------|"
  echo "| Unit  | **$unit_total** |"
  echo "| E2E   | **$e2e_total** |"
}
