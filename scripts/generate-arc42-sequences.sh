#!/usr/bin/env bash
# scripts/generate-arc42-sequences.sh — regenerate chapter 6's runtime
# sequence diagrams from `dynamicViews` defined in
# src/docs/arc42/architecture.jsonc, via `bausteinsicht export-sequence`
# (the same CLI feature product users get from `internal/model`'s
# DynamicView/SequenceStep types) — instead of the hand-written PlantUML
# blocks 06_runtime_view.adoc currently embeds directly.
#
# Sibling of generate-arc42-docs.sh (chapter 5's generator): same pattern
# (call the self-hosted CLI on the self-hosted model, write the result into
# the docs tree), applied to dynamic (sequence) views instead of static
# (structural) ones. Proposed as a follow-up in #535: chapter 6 has real
# but incomplete/stale content, and this closes the loop the same way
# `make arc42-docs` does for chapter 5 — as content is added to
# `dynamicViews`, this script keeps the rendered diagrams in sync
# automatically instead of requiring another manual pass.
#
# Usage:
#   scripts/generate-arc42-sequences.sh
#   make arc42-sequences
#
# Requirements:
#   - `bausteinsicht` binary on PATH, or built via `make build` first
#   - `plantuml` CLI (already required elsewhere in this repo, e.g. for
#     PDF/HTML builds via asciidoctor-diagram) for the optional PNG step
#
# What it does:
#   1. `export-sequence --diagram-format plantuml --output <dir>` ->
#      src/docs/arc42/sequence-<key>.puml, one per `dynamicViews` entry
#   2. If `plantuml` is on PATH: renders each .puml to
#      src/docs/images/arc42/architecture-sequence-<key>.png
#
# NOTE: as of #535, architecture.jsonc has zero `dynamicViews` defined —
# chapter 6's scenarios are still hand-written. This script is the
# generation mechanism; migrating each existing/missing scenario into
# `dynamicViews` (so it actually produces output, and so 06_runtime_view.adoc
# can include:: the generated .puml instead of embedding PlantUML source) is
# separate content-authoring work tracked in #535.

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

MODEL="src/docs/arc42/architecture.jsonc"
PUML_DIR="src/docs/arc42"
IMAGES_DIR="src/docs/images/arc42"

BIN="${BAUSTEINSICHT_BIN:-}"
if [ -z "$BIN" ]; then
  if command -v bausteinsicht >/dev/null 2>&1; then
    BIN="bausteinsicht"
  elif [ -x "./bausteinsicht" ]; then
    BIN="./bausteinsicht"
  else
    echo "==> Building bausteinsicht (no BAUSTEINSICHT_BIN set, none on PATH)..." >&2
    go build -o /tmp/bausteinsicht-arc42-sequences ./cmd/bausteinsicht
    BIN="/tmp/bausteinsicht-arc42-sequences"
  fi
fi

echo "==> Using binary: $BIN" >&2

# export-sequence prints "No dynamic views defined in model." to stderr and
# produces no stdout at all when dynamicViews is empty (rather than "[]"),
# so the JSON parse has to tolerate empty/invalid stdout, not just parse it.
seq_json="$("$BIN" export-sequence --model "$MODEL" --format json 2>/dev/null || true)"
view_count="$(printf '%s' "$seq_json" | python3 -c '
import json, sys
try:
    print(len(json.load(sys.stdin)))
except Exception:
    print(0)
')"
if [ "$view_count" -eq 0 ]; then
  echo "==> No dynamicViews defined in $MODEL — nothing to generate." >&2
  echo "    (chapter 6's scenarios are still hand-written; see #535)" >&2
  exit 0
fi

# ── 1. PlantUML sources ─────────────────────────────────────────────────────
echo "==> Regenerating sequence-*.puml files ($view_count view(s))..." >&2
"$BIN" export-sequence --model "$MODEL" --diagram-format plantuml --output "$PUML_DIR" >&2

# ── 2. PNG diagrams (local plantuml CLI, no draw.io/xvfb needed) ───────────
if ! command -v plantuml >/dev/null 2>&1; then
  echo "==> WARNING: no 'plantuml' CLI found — skipping PNG export." >&2
  echo "    .puml sources were still regenerated." >&2
else
  echo "==> Regenerating PNG diagrams..." >&2
  mkdir -p "$IMAGES_DIR"
  for puml in "$PUML_DIR"/sequence-*.puml; do
    [ -f "$puml" ] || continue
    plantuml -tpng -o "$ROOT/$IMAGES_DIR" "$puml"
    base="$(basename "$puml" .puml)"
    # plantuml names its output after the `@startuml <name>` directive
    # inside the file, not the source filename — and that name has
    # dashes/dots replaced with underscores (sanitizeID in
    # internal/diagram), so a view key like "as-is-vs-to-be" produces
    # `as_is_vs_to_be.png`, not `sequence-as-is-vs-to-be.png`.
    startuml_name="$(grep -m1 -oP '^@startuml \K\S+' "$puml")"
    src="$IMAGES_DIR/$startuml_name.png"
    dst="$IMAGES_DIR/architecture-$base.png"
    if [ -f "$src" ]; then
      mv "$src" "$dst"
      echo "    $dst" >&2
    else
      echo "    WARNING: expected $src not produced — skipping $dst" >&2
    fi
  done
fi

echo "==> Done. Review with 'git diff' before committing." >&2
