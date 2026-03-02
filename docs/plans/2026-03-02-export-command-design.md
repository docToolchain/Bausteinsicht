# Design: `bausteinsicht export` command

## Purpose

Export draw.io diagram pages to PNG or SVG using the draw.io CLI. Exported files can be embedded in AsciiDoc/HTML/Markdown as rendered images, and if `--embed-diagram` is used, they can also be re-opened in draw.io for editing.

## CLI Interface

```
bausteinsicht export [--format png|svg] [--view <key>] [--output <dir>] [--embed-diagram]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--format` | `png` | Export format: `png` or `svg` |
| `--view` | (all) | Export only one view by key |
| `--output` | `.` | Output directory |
| `--embed-diagram` | false | Embed draw.io XML source in output |

Global flags `--model`, `--format` (text/json), `--verbose` also apply.

Note: the export-specific `--format` flag shadows the global one. The command uses a local `--format` for the image format, while the global `--format` controls output messaging (text vs json).

## Output Naming

Files are named `architecture-<view-key>.<ext>`:
- `architecture-context.png`
- `architecture-containers.svg`

## draw.io CLI Detection

Checked in order:
1. `drawio-export` (devcontainer wrapper with xvfb)
2. `drawio` (native install)
3. Error: "draw.io CLI not found. Install from https://www.drawio.com/"

## Architecture

### New files

- `cmd/bausteinsicht/export.go` — Cobra command definition
- `cmd/bausteinsicht/export_test.go` — Command-level tests
- `internal/export/export.go` — Core export logic (CLI detection, argument building, execution)
- `internal/export/export_test.go` — Unit tests

### Flow

1. Load model (to get view keys and titles)
2. Load draw.io document (to get page count and IDs)
3. Detect draw.io CLI binary
4. For each view page, run: `<drawio-bin> --export --format <fmt> --page-index <N> --output <file> [--embed-diagram] architecture.drawio`
5. Report results (files created, any errors)

Page index is 1-based. Views are matched by page ID (`view-<key>`).

### Error Handling

- draw.io CLI not found: exit 2 with install hint
- Export fails for one page: log error, continue with remaining pages, exit 1 at end
- No views in model: exit 2 "no views to export"
- `--view` key not found: exit 2 "view not found"

## Testing Strategy

- Unit tests: CLI detection logic (mock exec.LookPath), argument building, output path generation
- Integration test: skip unless draw.io CLI is available (`testing.Short()` or env var)
