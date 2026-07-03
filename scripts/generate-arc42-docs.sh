#!/usr/bin/env bash
# scripts/generate-arc42-docs.sh — regenerate arc42 chapter 5's generated
# artifacts (element tables, PlantUML sources, PNG diagrams) from
# src/docs/arc42/architecture.jsonc via `bausteinsicht` CLI calls, instead
# of the manual, easy-to-forget process that caused the drift fixed in #524
# (11 packages missing from the model) and #526 (chapter 5's generated
# artifacts stale relative to the model).
#
# Usage:
#   scripts/generate-arc42-docs.sh
#   make arc42-docs
#
# Requirements:
#   - `bausteinsicht` binary on PATH, or built via `make build` first
#   - draw.io CLI (`drawio-export` wrapper or `drawio`) for the PNG step;
#     headless export needs a running dbus + xvfb-run -a (see CLAUDE.md's
#     "Headless draw.io Export" section) — this script starts dbus itself
#     if it isn't already running, but assumes xvfb-run is installed
#
# What it does, per view:
#   1. `export-table --view <view> --table-format adoc` ->
#      src/docs/arc42/chapters/<view>-elements.adoc (with the leading
#      "=== <View Title>" heading + blank line stripped, since this file is
#      include::'d into a chapter section that already has its own heading)
#   2. `export-diagram --view <view> --diagram-format plantuml` ->
#      src/docs/arc42/<view>.puml
#   3. `export --view <view>` -> src/docs/images/arc42/architecture-<view>.png
#
# Only regenerates the views actually referenced by a chapter's include::/
# image:: — not every view defined in the model (some, like
# importer-components/exporter-components/search-components/
# diagram-components, exist in the model but have no chapter section; see
# #526's discussion of that separate question).

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

MODEL="src/docs/arc42/architecture.jsonc"
CHAPTERS_DIR="src/docs/arc42/chapters"
PUML_DIR="src/docs/arc42"
IMAGES_DIR="src/docs/images/arc42"

VIEWS=(containers context drawio-components model-components sync-components cli-components)

BIN="${BAUSTEINSICHT_BIN:-}"
if [ -z "$BIN" ]; then
  if command -v bausteinsicht >/dev/null 2>&1; then
    BIN="bausteinsicht"
  elif [ -x "./bausteinsicht" ]; then
    BIN="./bausteinsicht"
  else
    echo "==> Building bausteinsicht (no BAUSTEINSICHT_BIN set, none on PATH)..." >&2
    go build -o /tmp/bausteinsicht-arc42-docs ./cmd/bausteinsicht
    BIN="/tmp/bausteinsicht-arc42-docs"
  fi
fi

echo "==> Using binary: $BIN" >&2

# ── 1. Element tables ──────────────────────────────────────────────────────
echo "==> Regenerating *-elements.adoc tables..." >&2
for view in "${VIEWS[@]}"; do
  out="$CHAPTERS_DIR/$view-elements.adoc"
  "$BIN" export-table --model "$MODEL" --view "$view" --table-format adoc \
    | tail -n +3 > "$out"
  echo "    $out" >&2
done

# ── 2. PlantUML sources ─────────────────────────────────────────────────────
echo "==> Regenerating .puml files..." >&2
for view in "${VIEWS[@]}"; do
  out="$PUML_DIR/$view.puml"
  "$BIN" export-diagram --model "$MODEL" --view "$view" --diagram-format plantuml > "$out"
  echo "    $out" >&2
done

# ── 3. PNG diagrams (headless draw.io) ──────────────────────────────────────
echo "==> Regenerating PNG diagrams..." >&2

# No --drawio-path needed: `export` auto-detects drawio-export/drawio/draw.io
# via PATH lookup on its own (internal/export.ResolveDrawioBinary). Passing
# a bare command name (not a real filesystem path) via --drawio-path fails,
# since that flag is for an explicit path, not a PATH-searchable name.
if ! command -v drawio-export >/dev/null 2>&1 && ! command -v drawio >/dev/null 2>&1; then
  echo "    WARNING: no draw.io CLI found (drawio-export/drawio) — skipping PNG export." >&2
  echo "    Element tables and .puml files were still regenerated." >&2
else
  if ! pgrep -x dbus-daemon >/dev/null 2>&1; then
    echo "    Starting dbus (required for headless draw.io export)..." >&2
    sudo mkdir -p /run/dbus
    sudo dbus-daemon --system --fork
  fi

  WORK="$(mktemp -d)"
  trap 'rm -rf "$WORK"' EXIT

  xvfb-run -a "$BIN" export --model "$MODEL" --output "$WORK" >&2

  for view in "${VIEWS[@]}"; do
    src="$WORK/architecture-$view.png"
    dst="$IMAGES_DIR/architecture-$view.png"
    if [ -f "$src" ]; then
      cp "$src" "$dst"
      echo "    $dst" >&2
    else
      echo "    WARNING: expected $src not produced — skipping $dst" >&2
    fi
  done
fi

echo "==> Done. Review with 'git diff' before committing." >&2
