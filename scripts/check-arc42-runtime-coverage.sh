#!/usr/bin/env bash
# scripts/check-arc42-runtime-coverage.sh — verify that every real
# cmd/bausteinsicht/*.go command is mentioned somewhere in
# src/docs/arc42/chapters/06_runtime_view.adoc.
#
# Sibling of check-arc42-process-coverage.sh (see that script's header for
# the shared rationale): this one checks chapter 6's sequence-diagram
# scenarios instead of chapter 3's process diagram. Proposed in #535, where
# ch6 was found to have real coverage (8 scenarios) but whole command groups
# — Evolution beyond REPL-save, the Export family, Analysis beyond `stale`,
# `validate`, `workspace`, and others — had no scenario at all.
#
# Same substring/case-insensitive heuristic and same caveat as
# check-arc42-process-coverage.sh: this is a low bar meant to surface
# candidates for review, not to judge documentation quality.
#
# Usage:
#   scripts/check-arc42-runtime-coverage.sh
#   make arc42-runtime-coverage-check
#
# Exit code 0 = every command mentioned, 1 = at least one gap found.

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

CHAPTER="src/docs/arc42/chapters/06_runtime_view.adoc"
CMD_DIR="cmd/bausteinsicht"

# Files that are not themselves a user-facing command (entry point, internal
# helpers reused by other commands, or a parent whose Use: token is covered
# via its own file).
EXCLUDE_FILES=(main.go root.go repl_save.go template_resolve.go)

# Subcommand files whose Cobra `Use:` token alone is too generic to search
# for (e.g. "view", "save", "list") — combine with their parent command.
declare -A PARENT_PREFIX=(
  [add_element.go]="add"
  [add_relationship.go]="add"
  [add_specification.go]="add"
  [add_view.go]="add"
  [snapshot_delete.go]="snapshot"
  [snapshot_diff.go]="snapshot"
  [snapshot_list.go]="snapshot"
  [snapshot_restore.go]="snapshot"
  [snapshot_save.go]="snapshot"
)

is_excluded() {
  local base="$1"
  for ex in "${EXCLUDE_FILES[@]}"; do
    [ "$base" = "$ex" ] && return 0
  done
  case "$base" in *_test.go) return 0 ;; esac
  return 1
}

exit_code=0
checked=0

echo "==> Checking command mentions in $CHAPTER..." >&2
for f in "$CMD_DIR"/*.go; do
  base="$(basename "$f")"
  is_excluded "$base" && continue

  use="$(grep -m1 -oP 'Use:\s*"\K[a-zA-Z0-9_-]+' "$f" || true)"
  [ -z "$use" ] && continue

  prefix="${PARENT_PREFIX[$base]:-}"
  if [ -n "$prefix" ]; then
    term="$prefix $use"
  else
    term="$use"
  fi

  checked=$((checked + 1))
  if ! grep -qiF -- "$term" "$CHAPTER"; then
    echo "  MISSING: '$term' (from $CMD_DIR/$base) not mentioned in $CHAPTER" >&2
    exit_code=1
  fi
done

echo "==> Checked $checked commands." >&2
if [ "$exit_code" -eq 0 ]; then
  echo "==> No gaps: every cmd/bausteinsicht command is mentioned in chapter 6." >&2
else
  echo "==> Gaps found — see #535 for the class of issue this catches." >&2
fi

exit "$exit_code"
