# Plan: VS Code Extension (LSP Server in Go)

## Overview

A Language Server Protocol (LSP) server in Go that brings Bausteinsicht diagnostics, CodeLens, and hover information to VS Code (and other LSP-compatible editors). This separates concerns:
- **LSP Server (Go)**: Business logic, validation, analysis
- **VS Code Extension (TypeScript)**: UI wrapper, command palette, status bar

## MVP Scope (Phase 1)

### Phase 1a: LSP Server Foundation
- [ ] Create `cmd/bausteinsicht-lsp/` with LSP server entry point
- [ ] Implement LSP lifecycle (initialize, shutdown, textDocument/didOpen, didChange, didSave)
- [ ] File watching and model loading on `architecture.jsonc` changes
- [ ] JSON-RPC transport layer

**Files:**
- `cmd/bausteinsicht-lsp/main.go` — server entry point
- `internal/lsp/server.go` — LSP server implementation
- `internal/lsp/handler.go` — message handlers
- `internal/lsp/document.go` — document state management

**Test:** Unit tests for LSP message handling

### Phase 1b: Diagnostics (Validation) ✅ COMPLETE
- [x] Map `bausteinsicht validate --format json` output to LSP Diagnostics
- [x] Display errors/warnings inline in editor
- [x] Support: unknown kinds, duplicate IDs, broken relationships, syntax errors
- [x] Unit tests for diagnostic mapping

**Files:**
- `internal/lsp/diagnostics.go` — validation → diagnostics mapping
- `internal/lsp/diagnostics_test.go` — comprehensive test suite

**Test Coverage** (All passing ✓):
- ConvertValidateOutput: maps errors & warnings to LSP Diagnostics
- FindLineInDocument: resolves JSON path to line numbers
- DiagnosticRange: calculates range for error highlights
- DiagnosticSeverityMapping: Error (1) vs Warning (2) classification
- EmptyValidateOutput: valid models produce no diagnostics

**Output:** Red/yellow underlines in architecture.jsonc for validation errors

### Phase 1c: CodeLens ✅ COMPLETE
- [x] CodeLens provider for element definitions
- [x] Show: element kind, element status, view count
- [x] Actions: "Open in draw.io" with element ID
- [x] Unit tests for CodeLens logic

**Files:**
- `internal/lsp/codelens.go` — CodeLens provider
- `internal/lsp/codelens_test.go` — comprehensive test suite

**Test Coverage** (All passing ✓):
- GenerateCodeLens: extracts elements and generates CodeLens objects
- ExtractKind: parses "kind" field from JSON metadata
- ExtractStatus: parses "status" field with "active" default
- EstimateViewCount: counts element references in document
- CodeLensRange: verifies range calculation for highlighting
- NonArchitectureFile: filters non-architecture.jsonc files
- CodeLensCommand: verifies command structure and arguments

**Output:** Clickable links above element definitions showing kind, status, and view count

### Phase 2: VS Code Extension (TypeScript/Minimal) 🚧 IN PROGRESS
- [x] Extension manifest (`package.json`)
- [x] LSP client setup (spawn server process)
- [x] Command palette: sync, validate, health, watch toggle
- [x] Status bar: watch mode status indicator
- [ ] Integration tests for VS Code extension
- [ ] Publish to VS Code Marketplace (future)

**Files:**
- `vscode-extension/package.json` — manifest with commands, config
- `vscode-extension/tsconfig.json` — TypeScript configuration
- `vscode-extension/src/extension.ts` — entry point, LSP client, commands
- `vscode-extension/.eslintrc.json` — code quality rules
- `vscode-extension/README.md` — installation & usage

**Features:**
- Auto-activate on `architecture.jsonc` files
- LSP client spawns `bausteinsicht-lsp` process
- Displays diagnostics inline (Phase 1b)
- Shows CodeLens with metadata (Phase 1c)
- Status bar shows connection & watch mode status
- Commands for sync, validate, health check, watch toggle
- Settings: server path, debug mode, draw.io URL

**Not in MVP:** Diagram preview, code snippets, advanced IDE integrations

## Architecture

```
bausteinsicht-lsp (Go server)
├── main.go
├── server.go (LSP lifecycle)
├── handler.go (message dispatch)
├── document.go (doc state)
├── diagnostics.go (validate → diagnostics)
├── codelens.go (CodeLens provider)
└── hover.go (hover info)

vscode-extension/ (TypeScript client)
├── package.json
├── src/
│   ├── extension.ts (LSP client)
│   ├── commands.ts (Command palette)
│   └── statusbar.ts (Status bar)
```

## LSP Features Implemented

| Feature | MVP | Status |
|---------|-----|--------|
| Initialize/Shutdown | ✓ | P1a |
| textDocument/didOpen | ✓ | P1a |
| textDocument/didChange | ✓ | P1a |
| textDocument/didSave | ✓ | P1a |
| textDocument/diagnostic | ✓ | P1b |
| textDocument/codeLens | ✓ | P1c |
| textDocument/hover | ✗ | Future |
| codeLens/resolve | ✗ | Future |
| workspace/symbol | ✗ | Future |

## Acceptance Criteria

- [ ] LSP server starts and responds to `initialize` request
- [ ] Diagnostics appear in VS Code when `architecture.jsonc` has validation errors
- [ ] CodeLens links appear above element definitions
- [ ] CodeLens "Open in draw.io" action works (launches browser with element ID)
- [ ] Extension Command Palette commands (sync, validate, health) work
- [ ] Status bar shows watch mode status
- [ ] Server handles file changes gracefully (no crashes)
- [ ] Tests pass for diagnostics mapping and CodeLens provider

## Deliverables

1. **Go LSP Server** (`cmd/bausteinsicht-lsp/`)
   - Standalone executable (`bausteinsicht-lsp`)
   - Published to GitHub Releases
   - Optional: include in main `bausteinsicht` binary as `bausteinsicht lsp-server`

2. **VS Code Extension** (`vscode-extension/`)
   - Extension ID: `bausteinsicht.bausteinsicht`
   - Published to VS Code Marketplace
   - Also: `.vsix` file in GitHub Releases

3. **Documentation**
   - Install instructions (Extension Marketplace)
   - LSP client/server protocol docs
   - Contributing guide for extension

## Testing Strategy

1. **LSP Server Unit Tests:**
   - Message parsing/serialization
   - Diagnostics mapping
   - CodeLens generation
   - Document state management

2. **Integration Tests:**
   - Server lifecycle (start, process change, shutdown)
   - E2E: change model file, verify diagnostics appear in server output

3. **Extension Tests:**
   - Extension activates when `.jsonc` file opened
   - LSP client connects to server
   - Commands execute successfully

## Future Phases (Post-MVP)

- Phase 3: Hover information (element description, kind, status)
- Phase 4: Diagram webview preview
- Phase 5: Snippet library
- Phase 6: CodeLens resolve actions (jump to tests/ADRs)
- Phase 7: Workspace symbols (search elements by name)
