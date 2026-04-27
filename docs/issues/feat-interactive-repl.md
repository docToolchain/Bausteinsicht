---
title: "feat: Interactive REPL Mode"
labels: enhancement
---

## Beschreibung

Currently, editing the architecture model requires knowledge of the JSONC format and a text editor. Non-technical stakeholders (product managers, architects less familiar with JSON) have no guided entry point. This feature adds an interactive REPL that wraps existing commands in a conversational CLI interface.

## Motivation

- Lowers the barrier for stakeholders who want to contribute to the architecture model
- Tab-completion of element IDs and kinds reduces typos and reference errors
- Undo stack prevents accidental data loss
- Reuses existing `add element`, `add relationship`, `validate` logic — no duplicate code

## Proposed Implementation

**New command:**
```
bausteinsicht repl [--model architecture.jsonc]
```

**Available REPL commands:**
```
list elements [kind]     — show elements table
list relationships       — show relationships
add element              — guided prompts (ID, kind, title, description, technology)
add relationship         — guided prompts (from, to, label)
remove element <id>      — remove by ID
show <id>                — show element/relationship details
validate                 — run model validation
sync                     — sync to draw.io
save                     — write changes to file
undo                     — undo last change
exit                     — exit (prompts if unsaved changes)
```

**Undo stack:** Deep copy of model before each mutation; `undo` restores last snapshot.

**No auto-save:** Changes are written only on explicit `save` or confirmed `exit`.

**Tab completion:** Element IDs, element kinds, and command names.

## Example Session

```
$ bausteinsicht repl
Bausteinsicht REPL (7 elements, 5 relationships)

> add element
Element ID   : notification-service
Kind         : service
Title        : Notification Service

✅ Added. Use 'save' to write to file.

> validate
✅ Model valid

> save
✅ Saved to architecture.jsonc
```

## Implementation Plan

See [`docs/plans/2026-03-18-interactive-repl.md`](../plans/2026-03-18-interactive-repl.md)

## Affected Components

- `internal/repl/` (new package)
- `cmd/bausteinsicht/repl.go` (new)
- New dependency: readline library (MIT/Apache licensed)
