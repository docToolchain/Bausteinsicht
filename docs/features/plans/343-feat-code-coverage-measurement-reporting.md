# Code Coverage Measurement & Reporting

**Issue:** #343  
**Epic:** Infrastructure  
**Priority:** post-release  

## Overview

Implement a comprehensive code coverage measurement and reporting system for Bausteinsicht.

## Phases

### Phase 1: Makefile Coverage Target
**Objective:** Create a local development target to measure and display code coverage

**Tasks:**
1. Add `make coverage` target to Makefile
   - Run `go test -cover ./...` to get package-level coverage
   - Generate HTML report using `go tool cover -html`
   - Generate text report showing coverage by package
   
2. Create `.coverprofile` output directory structure
3. Add coverage targets to `.gitignore`

**Output:** Local `coverage.html` and text summary showing coverage % per package

**Success Criteria:**
- `make coverage` runs without errors
- HTML report is readable in browser
- Text summary shows coverage % and identifies low-coverage packages

### Phase 2: CI Integration
**Objective:** Automate coverage measurement in GitHub Actions

**Tasks:**
1. Update `.github/workflows/go.yml` to:
   - Run `go test -coverprofile=coverage.out ./...` in CI
   - Calculate overall coverage percentage
   - Upload coverage artifacts to CI
   
2. Create GitHub Check comment showing:
   - Overall coverage percentage
   - Coverage trend (if possible)
   - Links to artifact

**Success Criteria:**
- CI generates coverage.out file
- Coverage % is visible in PR checks
- Artifacts are accessible after run

### Phase 3: Codecov Integration (Optional)
**Objective:** Use third-party service for coverage tracking and badges

**Tasks:**
1. Set up Codecov.io integration (if repo is public)
2. Add coverage badge to README.md
3. Configure badge to show latest coverage

**Success Criteria:**
- Badge displays in README
- Coverage history tracked on Codecov dashboard

### Phase 4: Report Generation & Artifacts
**Objective:** Generate and store coverage reports as CI artifacts

**Tasks:**
1. Generate HTML coverage report in CI
2. Create markdown coverage summary
3. Upload both as CI artifacts
4. Create GitHub Actions workflow to publish reports

**Success Criteria:**
- HTML report available in CI artifacts
- Markdown summary in PR checks
- Coverage data persisted for historical tracking

## Dependencies

- Go 1.x (already available)
- GitHub Actions (already in use)
- Optional: Codecov.io account (for v1.2+)

## Testing Strategy

- Verify `make coverage` runs locally
- Verify CI generates coverage.out
- Test coverage report generation
- Verify badge updates

## Acceptance Criteria

- [x] `make coverage` target implemented
- [x] CI generates coverage metrics
- [x] Coverage reports are accessible
- [x] Coverage percentage is visible in PR/README
- [ ] (Optional) Historical tracking via Codecov

## Timeline

- Phase 1: 1-2 hours (local setup)
- Phase 2: 1-2 hours (CI integration)
- Phase 3: 1 hour (Codecov setup)
- Phase 4: 1-2 hours (reporting)
