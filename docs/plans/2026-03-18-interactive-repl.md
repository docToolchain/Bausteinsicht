# Plan: Interactive REPL Mode

## Purpose

Provide a guided interactive shell for editing architecture models without requiring knowledge of the JSONC format. Useful for non-technical stakeholders and for rapid model exploration. Internally reuses existing `add element`, `add relationship`, and `validate` commands.

## CLI Interface

```
bausteinsicht repl [--model <file>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--model` | `architecture.jsonc` | Model file to edit |

## REPL Session Example

```
$ bausteinsicht repl
Bausteinsicht REPL — architecture.jsonc (7 elements, 5 relationships)
Type 'help' for available commands, 'exit' to quit.

> help

Commands:
  list elements [kind]       — List all elements (optionally filtered by kind)
  list relationships         — List all relationships
  list views                 — List all views
  add element                — Add a new element (guided prompts)
  add relationship           — Add a new relationship (guided prompts)
  remove element <id>        — Remove element by ID
  remove relationship <id>   — Remove relationship by ID
  show <id>                  — Show details of an element or relationship
  validate                   — Validate the model
  sync                       — Sync model to draw.io diagram
  save                       — Save changes to model file
  undo                       — Undo last change
  exit                       — Exit (prompts to save if unsaved changes)

> list elements

ID                  Kind        Title
──────────────────────────────────────────────────────
web-frontend        frontend    Web Frontend
api-gateway         service     API Gateway
order-service       service     Order Service
payment-service     service     Payment Service
message-broker      infra       Message Broker
user-db             database    User Database
order-db            database    Order Database

> add element

Element ID   : notification-service
Kind         : service
Title        : Notification Service
Description  : Sends email and push notifications
Technology   : Go

✅ Element 'notification-service' added.
   Use 'save' to write to file, or 'add relationship' to connect it.

> add relationship

From element ID : order-service
To element ID   : notification-service
Label           : OrderConfirmed event

✅ Relationship added: order-service → notification-service

> validate
✅ Model valid (8 elements, 6 relationships, 0 warnings)

> save
✅ Saved to architecture.jsonc
```

## Architecture

### New / Modified Files

| File | Change |
|------|--------|
| `internal/repl/repl.go` | New: REPL loop, command dispatcher, undo stack |
| `internal/repl/commands.go` | New: handlers for each REPL command |
| `internal/repl/printer.go` | New: table rendering for `list` commands |
| `internal/repl/history.go` | New: undo stack (copy of model before each mutation) |
| `cmd/bausteinsicht/repl.go` | New `repl` command entry point |

### Internal Design

```go
type REPL struct {
    modelPath string
    model     *model.BausteinsichtModel
    history   []model.BausteinsichtModel  // undo stack
    dirty     bool                        // unsaved changes
}

func (r *REPL) Run() error {
    // readline loop with tab-completion
}
```

**Undo stack:** Before each mutation, a deep copy of the model is pushed onto `history`. `undo` pops the last snapshot.

**Tab completion:** Element IDs, view keys, and command names are offered as completions using `github.com/chzyer/readline` or similar.

**No auto-save:** Changes are only written on explicit `save` command or confirmed `exit`.

## Guided Prompts

Each guided command asks for required fields in order. Optional fields are skipped unless `--verbose` is set:

```
> add element
Element ID   : <tab-complete existing IDs shown as hints>
Kind         : <tab-complete from spec.elementKinds>
Title        :
Description  : (optional, press Enter to skip)
Technology   : (optional, press Enter to skip)
```

Validation runs after each field: IDs must be unique, kind must exist in spec.

## Exit Behaviour

```
> exit
⚠ You have unsaved changes (2 additions, 0 removals).
Save before exiting? [Y/n]: y
✅ Saved. Goodbye.
```

## Dependencies

- `golang.org/x/term` — terminal raw mode detection
- `github.com/chzyer/readline` or equivalent — line editing, history, tab-completion

Both are already commonly used in Go CLI tools and have permissive licenses (MIT/Apache).

## Testing

- Unit tests for `list`, `add`, `remove`, `validate`, `undo` REPL commands with mock stdin/stdout
- Integration test: scripted REPL session → compare model file output
- Test: `undo` after `add element` restores original model
- Test: `exit` with unsaved changes prompts correctly
