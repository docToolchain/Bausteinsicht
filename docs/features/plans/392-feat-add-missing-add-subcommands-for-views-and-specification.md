# #392 Implementation Plan: Add Missing 'add' Subcommands

## Overview
Add missing CLI commands to complete the `bausteinsicht add` command family:
- `bausteinsicht add view` — Create views or modify include lists
- `bausteinsicht add specification` — Add element/relationship types to specification

## Problem Statement
Users must manually edit `architecture.jsonc` to:
1. Create new views
2. Add elements to a view's include list
3. Define new element/relationship types in specification

This blocks LLM-driven automation workflows.

## Acceptance Criteria
- [ ] `add view` command works with all flags
- [ ] `add specification` command works for elements & relationships  
- [ ] CLI Specification (02_cli_specification.adoc) updated with examples
- [ ] All tests pass (unit + integration)
- [ ] JSON output format (`--format json`) supported
- [ ] Error messages are helpful

---

## Phase 1: Implement `add view` Command

### Tasks
1. Create `cmd/bausteinsicht/add_view.go`
   - Command structure with flags: `--scope`, `--include` (repeatable), `--title`, `--description`
   - Parse and validate arguments
   
2. Implement view creation logic in `internal/model/`
   - `model.AddView()` function
   - Validation: scope must exist, include paths must exist
   - JSONC write via existing `model.Save()`
   
3. Add unit tests
   - Test view creation
   - Test include list handling
   - Test error cases (scope not found, duplicate view key, etc.)

### Acceptance
- [ ] `bausteinsicht add view my-view --scope system --include "element1" "element2"` works
- [ ] New view appears in model file
- [ ] Validation catches scope/include errors
- [ ] Tests pass

---

## Phase 2: Implement `add specification` Command

### Tasks
1. Create `cmd/bausteinsicht/add_specification.go`
   - Subcommands: `element` and `relationship`
   - Element flags: `--notation`, `--description`, `--container`, `--technology`
   - Relationship flags: `--notation`, `--description`, `--dashed`
   
2. Implement specification logic in `internal/model/`
   - `model.AddSpecificationElement()` function
   - `model.AddSpecificationRelationship()` function
   - Validation: no duplicate types, valid field values
   - JSONC write via `model.Save()`
   
3. Add unit tests
   - Test element creation
   - Test relationship creation
   - Test error cases (duplicate type, invalid notation)

### Acceptance
- [ ] `bausteinsicht add specification element custom --notation "Custom" --container` works
- [ ] `bausteinsicht add specification relationship uses --notation "uses" --dashed` works
- [ ] New types appear in model's specification section
- [ ] Validation prevents duplicates
- [ ] Tests pass

---

## Phase 3: Update CLI Specification & Documentation

### Tasks
1. Update `src/docs/spec/02_cli_specification.adoc`
   - Add entries to Command Overview table:
     - `bausteinsicht add view` → Implemented
     - `bausteinsicht add specification` → Implemented
   
   - Add detailed sections (after existing `add` commands):
     ```adoc
     ==== `bausteinsicht add view`
     <Usage, flags, examples>
     
     ==== `bausteinsicht add specification`
     <Usage, subcommands, examples>
     ```

2. Add to `cmd/bausteinsicht/add.go`
   - Register new subcommands in `newAddCmd()`

### Acceptance
- [ ] Spec document updated with command overview
- [ ] Detailed sections with examples
- [ ] All flags documented
- [ ] Examples are copy-paste ready

---

## Testing Strategy

### Unit Tests
- `add_view_test.go` — view creation, validation, JSONC write
- `add_specification_test.go` — element/relationship creation, validation

### Integration Tests
- Add → Validate → Sync round-trip
- JSON output format (`--format json`)
- Error handling and messages

### Manual Testing
- [ ] Create view with multiple includes
- [ ] Modify existing view's include list
- [ ] Add custom element type
- [ ] Add custom relationship type
- [ ] Test error cases (invalid scope, duplicate types)

---

## Files to Modify/Create

### Create
- `cmd/bausteinsicht/add_view.go`
- `cmd/bausteinsicht/add_view_test.go`
- `cmd/bausteinsicht/add_specification.go`
- `cmd/bausteinsicht/add_specification_test.go`

### Modify
- `cmd/bausteinsicht/add.go` — Register new subcommands
- `internal/model/add.go` or `internal/model/model.go` — AddView(), AddSpecificationElement/Relationship()
- `src/docs/spec/02_cli_specification.adoc` — Update documentation

---

## Commit Strategy

| Phase | Commits |
|-------|---------|
| Phase 1 | `feat(add): Implement add view command` |
| Phase 2 | `feat(add): Implement add specification element/relationship` |
| Phase 3 | `docs(spec): Update CLI specification with add view/specification` |

---

## Risk & Mitigation

| Risk | Mitigation |
|------|-----------|
| JSONC write consistency | Use existing `model.Save()` — no new serialization logic |
| Validation not catching invalid includes | Write comprehensive tests + validation helpers |
| Incomplete spec documentation | Cross-check against existing `add element` / `add relationship` docs |

---

## Definition of Done
- [ ] All phases complete
- [ ] Unit + integration tests pass
- [ ] CLI Specification updated
- [ ] No lint/vet errors
- [ ] Commits follow convention
- [ ] Ready for PR review
