# Plan: VS Code Extension

## Purpose

A VS Code extension (`bausteinsicht.vscode`) that brings the Bausteinsicht workflow directly into the editor: live diagram preview, inline validation errors, CodeLens links to tests and ADRs, and a Command Palette for all CLI operations.

## Extension ID

`bausteinsicht.bausteinsicht` (publisher: `bausteinsicht`)

## Features

### 1. Live Diagram Preview

A webview panel showing the current draw.io diagram, updated automatically when `architecture.jsonc` or `architecture.drawio` changes.

```
Ctrl+Shift+P → "Bausteinsicht: Open Diagram Preview"
```

- Renders the draw.io XML using the draw.io embed API (read-only, no editing in preview)
- View selector dropdown to switch between views/pages
- Auto-refreshes on file save

### 2. Inline Validation (Diagnostics)

Runs `bausteinsicht validate --format json` on save and maps results to VS Code diagnostics:

- Red underlines for errors (unknown kind references, duplicate IDs, broken relationship targets)
- Yellow underlines for warnings (orphan ADRs, deprecated elements without successors)
- Hover tooltip shows full error message

```json
// architecture.jsonc — with inline error
{ "id": "svc", "kind": "unknown-kind" }
//                       ^^^^^^^^^^^^^
//  Error: kind "unknown-kind" not defined in spec.elementKinds
```

### 3. CodeLens

CodeLens links appear above element definitions:

```jsonc
// 🔗 Open in draw.io  |  📋 2 ADRs  |  ✅ 3 tests  |  🟡 deprecated
{ "id": "order-service", "kind": "service", "status": "deprecated" }
```

- **Open in draw.io**: opens the element in the draw.io diagram (scrolls to it)
- **ADRs**: opens the linked ADR files in a split editor
- **Tests**: opens related test files (matched by element ID in test filenames)
- **Status**: quick-action to change lifecycle status

### 4. Command Palette

All `bausteinsicht` CLI commands available via `Ctrl+Shift+P`:

| Command | CLI equivalent |
|---------|---------------|
| Bausteinsicht: Sync | `bausteinsicht sync` |
| Bausteinsicht: Watch (toggle) | `bausteinsicht watch` |
| Bausteinsicht: Validate | `bausteinsicht validate` |
| Bausteinsicht: Lint | `bausteinsicht lint` |
| Bausteinsicht: Health Score | `bausteinsicht health` |
| Bausteinsicht: Add Element | `bausteinsicht add element` (guided) |
| Bausteinsicht: Add Relationship | `bausteinsicht add relationship` (guided) |
| Bausteinsicht: Generate Template | `bausteinsicht generate-template` |
| Bausteinsicht: Open Diagram Preview | (webview) |

Output shown in a dedicated "Bausteinsicht" Output Channel.

### 5. Status Bar Item

Shows current watch mode status and last sync time:

```
$(sync~spin) Bausteinsicht: watching  |  Last sync: 14:32:01  |  ✅ 12 elements
```

Click to toggle watch mode.

### 6. Snippet Library

JSON snippets for common patterns:

- `bbs-element` → element scaffold with all fields
- `bbs-relationship` → relationship scaffold
- `bbs-view` → view definition
- `bbs-constraint` → constraint definition
- `bbs-dynamic-view` → dynamic view scaffold

## Technology

| Component | Technology |
|-----------|-----------|
| Extension host | TypeScript + VS Code Extension API |
| Webview (preview) | draw.io embed (iframe with `viewer.min.js`) |
| CLI invocation | `child_process.spawn` → `bausteinsicht` binary |
| Diagnostics | `vscode.languages.createDiagnosticCollection` |
| Schema | JSON Language Server (via `$schema` in architecture.jsonc) |

**Note:** This is the only planned feature that uses TypeScript/JavaScript. The extension is a thin UI wrapper — all business logic remains in the Go CLI.

## Extension Settings

```json
{
  "bausteinsicht.cliPath": "",        // path to bausteinsicht binary (default: from PATH)
  "bausteinsicht.autoSync": false,    // sync on save
  "bausteinsicht.validateOnSave": true,
  "bausteinsicht.previewTheme": "light"
}
```

## Architecture

### Files

```
vscode-extension/
├── package.json              ← extension manifest
├── src/
│   ├── extension.ts          ← entry point, command registration
│   ├── preview.ts            ← webview panel (diagram preview)
│   ├── diagnostics.ts        ← validation → VS Code diagnostics
│   ├── codelens.ts           ← CodeLens provider
│   ├── commands.ts           ← CLI command wrappers
│   ├── statusbar.ts          ← status bar item
│   └── snippets.ts           ← snippet registration
├── snippets/
│   └── bausteinsicht.json    ← JSON snippet definitions
└── test/
    └── extension.test.ts
```

## Distribution

- Published to VS Code Marketplace (`bausteinsicht.bausteinsicht`)
- Also installable via `.vsix` file from GitHub Releases
- Requires `bausteinsicht` CLI installed separately (extension downloads it if missing, with user confirmation)

## Testing

- Unit tests for diagnostics mapping (validate JSON output → VS Code Diagnostic objects)
- Unit tests for CodeLens provider (element parsing → lens items)
- Integration test: extension activates on `.jsonc` file with `$schema` reference to bausteinsicht schema
- E2E test: command palette sync → output channel shows sync result
