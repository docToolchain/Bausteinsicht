#!/usr/bin/env bash
# scripts/check-arc42-level2-coverage.sh — verify that every "*-components"
# view defined in src/docs/arc42/architecture.jsonc has a matching
# "=== Level 2: ..." section in chapters/05_building_block_view.adoc.
#
# Third sibling of check-arc42-process-coverage.sh / check-arc42-runtime-
# coverage.sh, but at a different granularity: those check individual CLI
# commands against chapters 3/6; this checks model *views* against chapter
# 5. Found via #539: architecture.jsonc defines 10 views, but chapter 5 had
# only 4 "Level 2" sections — a gap explicitly deferred in #526
# ("kein Drift, nur potenziell fehlende Doku-Tiefe") and never revisited.
#
# A view "counts" as covered if its key appears in chapter 5 as either
# an image:: filename (architecture-<view>.png) or an include:: filename
# (<view>-elements.adoc) — the same two artifacts generate-arc42-docs.sh
# produces per view.
#
# Usage:
#   scripts/check-arc42-level2-coverage.sh
#   make arc42-level2-coverage-check
#
# Exit code 0 = every "*-components" view has a Level-2 section, 1 = gaps
# found. containers/context are Level-1 views (System Context / Container
# View sections already cover them) and are deliberately excluded.

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

MODEL="src/docs/arc42/architecture.jsonc"
CHAPTER="src/docs/arc42/chapters/05_building_block_view.adoc"

modeled_views="$(python3 -c '
import json, re
with open("'"$MODEL"'") as f:
    text = f.read()
data = json.loads(re.sub(r"(?m)^\s*//.*$", "", text))
for k in data["views"]:
    if k.endswith("-components"):
        print(k)
')"

exit_code=0
checked=0

echo "==> Checking Level-2 section coverage in $CHAPTER..." >&2
while IFS= read -r view; do
  [ -z "$view" ] && continue
  checked=$((checked + 1))
  # Two separate grep -q calls (not a single "a\|b" alternation): \| for
  # alternation in BRE is a GNU extension, not POSIX-portable to BSD grep
  # (e.g. macOS's default /usr/bin/grep) — there it degrades to a literal,
  # never-matching string, falsely reporting every view as MISSING. Same
  # class of portability bug as the -oP issue fixed in
  # check-arc42-process-coverage.sh / check-arc42-runtime-coverage.sh.
  if ! grep -q -- "architecture-$view.png" "$CHAPTER" && ! grep -q -- "$view-elements.adoc" "$CHAPTER"; then
    echo "  MISSING: view '$view' has no Level-2 section (no architecture-$view.png or $view-elements.adoc reference) in $CHAPTER" >&2
    exit_code=1
  fi
done <<<"$modeled_views"

echo "==> Checked $checked component views." >&2
if [ "$exit_code" -eq 0 ]; then
  echo "==> No gaps: every *-components view has a Level-2 section in chapter 5." >&2
else
  echo "==> Gaps found — see #539/#526 for the class of issue this catches." >&2
fi

exit "$exit_code"
