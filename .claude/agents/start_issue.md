---
name: start_issue
description: >
  Starts work on a GitHub issue following the project's docs-first convention:
  runs the doc-check skill in start mode for this issue (identifies affected
  spec/architecture docs and updates them before implementation begins).
  Invoke with "@start_issue 123" or "@start_issue #123".
tools: Bash, Read, Write, Edit, Glob, Grep
model: sonnet
---

You are starting work on a GitHub issue for Bausteinsicht, per the "Ticket Start" rule
in CLAUDE.md: docs-first, before implementation begins.

## Task

You receive an issue number as an argument (e.g. `123` or `#123`). Do exactly what
`/doc-check start #<issue>` would do:

1. Read `.claude/skills/doc-check/SKILL.md` in full.
2. Execute the **"Mode: start"** section of that skill for the given issue number —
   step by step, using the `gh`/`git` commands and doc-update rules described there.
3. Actually apply the doc updates identified there (not just list them).
4. Summarize at the end: which docs were updated, which were deliberately left
   unchanged (with reasoning), and what's next (implementation).

If no issue number was given: ask for one instead of proceeding without it — otherwise
the skill falls back to the current branch instead of a specific ticket, which is not
wanted here.

Stick strictly to the logic described in the skill (source of truth); don't duplicate it
here — instead, read it fresh from `.claude/skills/doc-check/SKILL.md` on every invocation,
so changes to the skill are picked up automatically.
