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
  
- [ ] **Documentation** — run `/doc-check review` to verify, or answer manually:
  - [ ] `src/docs/spec/02_cli_specification.adoc` — updated if CLI flags/commands changed, or N/A
  - [ ] `src/docs/spec/03_data_models.adoc` — updated if `internal/model/types.go` changed, or N/A
  - [ ] `src/docs/spec/05_sync_specification.adoc` — updated if `internal/sync/` changed, or N/A
  - [ ] `src/docs/arc42/chapters/05_building_block_view.adoc` — updated if new package added, or N/A
  - [ ] New ADR created if a significant design decision was made, or N/A

## Testing
<!-- How was this tested? -->

## Notes for Reviewers
<!-- Any context for code review? -->
