## Description
<!-- What does this PR do? -->

## Related Issue
Closes #XXX

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Improvement/refactoring
- [ ] Documentation
- [ ] Other (describe)

## Pre-Merge Checklist (Developer)

- [ ] **Branch is up-to-date with main**
  - Ran: `git rebase origin/main` (if behind)
  - No merge conflicts
  
- [ ] **No duplicate work**
  - Checked open PRs for overlapping functionality
  - Did not start a duplicate of existing PR
  
- [ ] **Tests passing**
  - Unit tests: ✅
  - Integration tests: ✅
  - CI pipeline: ✅
  
- [ ] **Code quality**
  - Ran `make check` locally
  - No new warnings/errors
  
- [ ] **Documentation** — run `/doc-check review` (Claude Code) or fill in manually:
  - [ ] `spec/01_use_cases.adoc` — updated / N/A (reason: )
  - [ ] `spec/02_cli_specification.adoc` — updated / N/A (reason: ) ← required if any `--flag`, command, subcommand rename, or output format changed
  - [ ] `spec/03_data_models.adoc` — updated / N/A (reason: ) ← required if `types.go` or `schemas/bausteinsicht.schema.json` changed
  - [ ] `spec/04_acceptance_criteria.adoc` — updated / N/A (reason: )
  - [ ] `spec/05_sync_specification.adoc` — updated / N/A (reason: ) ← required if `internal/sync/` changed
  - [ ] `arc42/05_building_block_view.adoc` — updated / N/A (reason: ) ← required if new package added
  - [ ] `arc42/06_runtime_view.adoc` — updated / N/A (reason: ) ← required if new data flow or runtime path added
  - [ ] `arc42/08_concepts.adoc` — updated / N/A (reason: ) ← required if new cross-cutting pattern introduced
  - [ ] New ADR created / N/A (reason: )

## Testing
<!-- How was this tested? -->

## Notes for Reviewers
<!-- Any context for code review? -->
