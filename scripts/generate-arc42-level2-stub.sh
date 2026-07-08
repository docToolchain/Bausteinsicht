#!/usr/bin/env bash
# scripts/generate-arc42-level2-stub.sh — print a ready-to-insert
# "=== Level 2: <Title>" AsciiDoc section for a given *-components view,
# sourced from that view's title/description in architecture.jsonc
# (the same source data check-arc42-level2-coverage.sh checks against).
#
# This answers a question from #539: chapter 5's Level-2 sections all
# follow one fixed shape (heading, one-line intro from the view's
# description, .Caption + image::, include::<view>-elements.adoc[]) — that
# shape is entirely mechanical, so it doesn't need to be hand-typed each
# time a view gains a section; only the model's title/description needs to
# exist (which every view already has, since export-table/-diagram use them
# too).
#
# Usage:
#   scripts/generate-arc42-level2-stub.sh <view-key>
#   scripts/generate-arc42-level2-stub.sh diagram-components
#
# Prints the section to stdout — pipe into the chapter file or review
# before pasting; this does not edit 05_building_block_view.adoc itself
# (section placement/ordering within the chapter is still an editorial
# choice, not mechanical).

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

MODEL="src/docs/arc42/architecture.jsonc"
VIEW="${1:?Usage: $0 <view-key>}"

python3 -c '
import json, re, sys

view_key = sys.argv[1]
with open("'"$MODEL"'") as f:
    text = f.read()
data = json.loads(re.sub(r"(?m)^\s*//.*$", "", text))

view = data["views"].get(view_key)
if view is None:
    sys.exit(f"view {view_key!r} not found in architecture.jsonc")

title = view["title"]
description = view.get("description", "").rstrip(".")

print(f"=== Level 2: {title}")
print()
if description:
    print(f"{description}.")
    print()
print(f".{title} (generated from architecture model via draw.io export)")
print(f"image::arc42/architecture-{view_key}.png[{title},width=80%]")
print()
print("==== Building Blocks")
print()
print(f"include::{view_key}-elements.adoc[]")
' "$VIEW"
