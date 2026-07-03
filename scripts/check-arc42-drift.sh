#!/usr/bin/env bash
# scripts/check-arc42-drift.sh — verify that every real internal/<pkg> and
# cmd/<binary> Go package has a matching `container` element in
# src/docs/arc42/architecture.jsonc, and vice versa.
#
# This is the check that would have caught #524 (11 internal/ packages
# missing from the model) before it happened. It uses `bausteinsicht
# export-table --format json` — the same CLI the model itself is built
# with — to get a machine-readable element list, rather than re-parsing
# the JSONC by hand.
#
# Usage:
#   scripts/check-arc42-drift.sh
#   make arc42-drift-check
#
# Exit code 0 = no drift, 1 = drift found (real package with no container,
# or container with no matching real package) — usable as a CI gate.
#
# Deliberately excluded from the "every real package needs a container"
# check (kept in sync with the JSONC comment above the container list and
# with 05_building_block_view.adoc's NOTE — if you add a new deliberate
# exception, update all three places):
#   - internal/benchmarks, internal/chaos, internal/e2eplan — test/tooling
#     packages, not product architecture (see architecture.jsonc's own
#     comment on this, added in #524)
#   - cmd/bausteinsicht-lsp — a thin binary entry point reusing
#     internal/lsp's logic, not an independent package group; "lsp"
#     already covers it

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

MODEL="src/docs/arc42/architecture.jsonc"
EXCLUDE_PACKAGES=(benchmarks chaos e2eplan)

BIN="${BAUSTEINSICHT_BIN:-}"
if [ -z "$BIN" ]; then
  if command -v bausteinsicht >/dev/null 2>&1; then
    BIN="bausteinsicht"
  elif [ -x "./bausteinsicht" ]; then
    BIN="./bausteinsicht"
  else
    echo "==> Building bausteinsicht (no BAUSTEINSICHT_BIN set, none on PATH)..." >&2
    go build -o /tmp/bausteinsicht-arc42-drift-check ./cmd/bausteinsicht
    BIN="/tmp/bausteinsicht-arc42-drift-check"
  fi
fi

# ── Modeled container short-names (last dot-path segment) ──────────────────
modeled="$("$BIN" export-table --model "$MODEL" --view containers --format json \
  | python3 -c '
import json, sys
for e in json.load(sys.stdin):
    if e["kind"] == "container":
        print(e["id"].rsplit(".", 1)[-1])
')"

# ── Real package short-names ────────────────────────────────────────────────
real_internal="$(ls -d internal/*/ | xargs -n1 basename)"
# cmd/bausteinsicht maps to the "cli" container; cmd/bausteinsicht-lsp is a
# deliberate exception (see header comment).
real_cmd="cli"

exit_code=0

echo "==> Checking for real internal/ packages missing from the model..." >&2
for pkg in $real_internal; do
  skip=false
  for ex in "${EXCLUDE_PACKAGES[@]}"; do
    if [ "$pkg" = "$ex" ]; then
      skip=true
      break
    fi
  done
  $skip && continue
  if ! grep -qx "$pkg" <<<"$modeled"; then
    echo "  MISSING: internal/$pkg has no container element in $MODEL" >&2
    exit_code=1
  fi
done

echo "==> Checking cmd/ packages..." >&2
if ! grep -qx "$real_cmd" <<<"$modeled"; then
  echo "  MISSING: cmd/bausteinsicht ('cli') has no container element in $MODEL" >&2
  exit_code=1
fi

echo "==> Checking for modeled containers with no matching real package..." >&2
all_real="$real_internal
$real_cmd"
while IFS= read -r name; do
  [ -z "$name" ] && continue
  if ! grep -qx "$name" <<<"$all_real"; then
    echo "  STALE: container '$name' in $MODEL has no matching internal/$name or cmd/ package (renamed or removed?)" >&2
    exit_code=1
  fi
done <<<"$modeled"

if [ "$exit_code" -eq 0 ]; then
  echo "==> No drift: every real internal/ and cmd/ package (except deliberate exceptions) has a matching container element, and vice versa." >&2
else
  echo "==> Drift found — see #524/#526 for how this class of gap was found and fixed before; run 'make arc42-docs' after updating $MODEL." >&2
fi

exit "$exit_code"
