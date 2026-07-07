#!/usr/bin/env bash
# scripts/check-arc42-process-coverage.sh — verify that every real
# cmd/bausteinsicht/*.go command is mentioned somewhere in
# src/docs/arc42/chapters/03_context_and_scope.adoc (§3.1.1 "Architecture
# Maintenance Process").
#
# This is a lightweight, textual companion to check-arc42-drift.sh: that
# script catches *structural* drift (a package missing from
# architecture.jsonc); this one catches *prose* drift (a command the process
# diagram/narrative never mentions). It was proposed in #535, where §3.1.1
# was found to describe only 5 of ~35 commands after the CLI grew far beyond
# its original sync-loop scope.
#
# The check is deliberately a substring/case-insensitive text search, not a
# semantic one — a command "counts" as covered if its name (or its
# "<parent> <child>" form for subcommands) appears anywhere in the chapter.
# That is a low bar on purpose: this script flags candidates for a
# human/doc-check review, it does not judge documentation quality.
#
# Usage:
#   scripts/check-arc42-process-coverage.sh
#   make arc42-process-coverage-check
#
# Exit code 0 = every command mentioned, 1 = at least one gap found.

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

CHAPTER="src/docs/arc42/chapters/03_context_and_scope.adoc"
CMD_DIR="cmd/bausteinsicht"

# Files that are not themselves a user-facing command (entry point, internal
# helpers reused by other commands, or a parent whose Use: token is covered
# via its own file).
EXCLUDE_FILES=(main.go root.go repl_save.go template_resolve.go)

# Subcommand files whose own first (or only) `Use:` token is too generic to
# search for alone (e.g. "element", "view", "save") because their parent
# command lives in a *different* file (add.go / snapshot.go) — prefix with
# that parent. Composes with the multi-Use: local-parent logic below: e.g.
# add_specification.go's three tokens (specification/element/relationship)
# become "add specification", "add specification element", "add
# specification relationship". Files where parent and subcommands are
# defined together (workspace.go, overlay.go, adr.go, cmd_schema.go) don't
# need an entry here — their own first Use: token is already a specific
# enough parent on its own.
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

# add_from_pattern.go defines two Use: tokens ("add-from-pattern
# <pattern-id>" and "list"), but they are *not* parent/child in the real
# Cobra command tree: newListPatternsCmd()'s "list" is wired in add.go
# under a completely separate "pattern" command group
# (patternCmd.AddCommand(newListPatternsCmd())), never under
# add-from-pattern. Combining them as "add-from-pattern list" would assert
# a command that doesn't exist (verified: `bausteinsicht add-from-pattern`
# errors "unknown command" — the real invocations are
# `bausteinsicht add add-from-pattern <pattern-id>` and
# `bausteinsicht add pattern list`). So this file's extra tokens are
# ignored here, and the real 3-level term is listed in EXTRA_TERMS instead
# — a cross-file relationship (add.go's "pattern" parent + this file's
# "list" child) the per-file heuristic can't derive on its own.
IGNORE_EXTRA_USES=(add_from_pattern.go)

# Terms that can't be derived by the per-file heuristic above (cross-file
# 3-level nesting, or any other real command the automatic derivation
# can't reach) — checked in addition to the per-file loop.
EXTRA_TERMS=(
  "add pattern list"
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

check_term() {
  local term="$1" base="$2"
  checked=$((checked + 1))
  if ! grep -qiF -- "$term" "$CHAPTER"; then
    echo "  MISSING: '$term' (from $CMD_DIR/$base) not mentioned in $CHAPTER" >&2
    exit_code=1
  fi
}

echo "==> Checking command mentions in $CHAPTER..." >&2
for f in "$CMD_DIR"/*.go; do
  base="$(basename "$f")"
  is_excluded "$base" && continue

  # Portable extraction (POSIX BRE via sed, not grep -P/PCRE — GNU grep's -P
  # isn't available on BSD grep, e.g. macOS's default /usr/bin/grep, which
  # would otherwise silently yield zero matches everywhere and make this
  # script falsely report "no gaps"). Extracts *every* Use: token in the
  # file, not just the first — a file can define a parent command and all
  # its subcommands together (e.g. workspace.go: workspace/merge/validate/
  # list), and only checking the first token used to leave every subcommand
  # in such files permanently unchecked.
  mapfile -t uses < <(sed -n 's/^[[:space:]]*Use:[[:space:]]*"\([a-zA-Z0-9_-]*\).*/\1/p' "$f")
  [ "${#uses[@]}" -eq 0 ] && continue

  for ignore in "${IGNORE_EXTRA_USES[@]}"; do
    [ "$base" = "$ignore" ] && uses=("${uses[0]}")
  done

  prefix="${PARENT_PREFIX[$base]:-}"
  if [ -n "$prefix" ]; then
    local_parent="$prefix ${uses[0]}"
  else
    local_parent="${uses[0]}"
  fi
  check_term "$local_parent" "$base"
  for ((i = 1; i < ${#uses[@]}; i++)); do
    check_term "$local_parent ${uses[$i]}" "$base"
  done
done

for term in "${EXTRA_TERMS[@]}"; do
  check_term "$term" "(EXTRA_TERMS)"
done

echo "==> Checked $checked commands." >&2
if [ "$exit_code" -eq 0 ]; then
  echo "==> No gaps: every cmd/bausteinsicht command is mentioned in §3.1.1." >&2
else
  echo "==> Gaps found — see #535 for the class of issue this catches." >&2
fi

exit "$exit_code"
