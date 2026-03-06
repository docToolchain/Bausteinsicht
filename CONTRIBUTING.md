# Contributing to Bausteinsicht

## Development Setup

The recommended way to develop Bausteinsicht is using the devcontainer:

```bash
devcontainer up --workspace-folder .
devcontainer exec --workspace-folder . make check
```

This provides all required tools: Go 1.24+, static analysis, security scanners, draw.io (headless), and Claude Code.

For manual setup, install Go 1.24+ and run `make install-tools`.

## Adding a New Model Field

When adding a new field to the `Element` struct, follow this checklist:

1. **Add the field** to the `Element` struct in `internal/model/types.go`
2. **Update the JSON Schema** in `bausteinsicht.schema.json` to include the new field
3. **Verify `elementFieldPath()`** handles it in `internal/sync/patchops.go` — this enables comment-preserving patch saves
4. **Add reverse sync handling** in `internal/sync/reverse.go` — the `applyElementChange` function must map the field for draw.io → model sync
5. **Add a comment-preservation E2E test** — verify that JSONC comments survive a roundtrip when the new field changes
6. **Add a roundtrip E2E test** — verify forward sync (model → draw.io) and reverse sync (draw.io → model) for the new field

## Build and Test

```bash
make build          # build the CLI binary
make test           # run all tests
make test-race      # run tests with race detector
make check          # all analysis tools + race-detected tests
make bench          # run benchmarks
```

## PR Merge Policy

Before merging any PR:

1. **Security review** on the changes
2. **Code review** on the changes
3. All CI checks must pass (`make check`)

## Code Style

- Go standard formatting (`gofmt`)
- No unnecessary abstractions — keep it simple
- Tests next to the code they test (`*_test.go`)
- Property-based tests where applicable (`pgregory.net/rapid`)

## Documentation

- All documentation in English
- Documentation format: AsciiDoc (`.adoc`)
- ADRs go in `src/docs/arc42/ADRs/` with filename `ADR-NNN-Name.adoc`
