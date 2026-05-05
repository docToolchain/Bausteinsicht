# Bausteinsicht VS Code Extension

Architecture-as-Code with real-time VS Code integration via Language Server Protocol.

## Features

✨ **Real-time Validation**
- Inline error/warning diagnostics for architecture models
- Instant feedback as you edit

🎯 **CodeLens**
- Element metadata at a glance (kind, status, view count)
- Click to open element in draw.io

📝 **Language Support**
- Full support for `architecture.jsonc` files
- JSON schema validation and autocompletion

🔄 **Synchronization**
- Sync architecture model with draw.io diagrams
- Watch mode for live updates

## Installation

### From VS Code Marketplace
1. Open VS Code Extensions (Ctrl+Shift+X / Cmd+Shift+X)
2. Search for "Bausteinsicht"
3. Click Install

### From Release
1. Download `.vsix` file from [GitHub Releases](https://github.com/docToolchain/Bausteinsicht/releases)
2. Run: `code --install-extension bausteinsicht-*.vsix`

## Requirements

- VS Code 1.85+
- `bausteinsicht-lsp` binary in PATH (or configure path in settings)

Install the Bausteinsicht CLI:
```bash
# Homebrew
brew install docToolchain/tap/bausteinsicht

# Or build from source
cd Bausteinsicht
make build
```

## Usage

### Basic Workflow
1. Create `architecture.jsonc` in your project
2. Extension activates automatically
3. Edit your model and see errors/warnings inline
4. Use CodeLens to navigate and inspect elements

### Commands

Open Command Palette (Ctrl+Shift+P / Cmd+Shift+P):

- **Bausteinsicht: Validate** — Validate the current model
- **Bausteinsicht: Sync Model** — Sync changes to draw.io
- **Bausteinsicht: Health Check** — Verify server connection
- **Bausteinsicht: Toggle Watch Mode** — Enable/disable auto-sync

### Settings

Configure in VS Code settings (File → Preferences → Settings):

```json
{
  "bausteinsicht.serverPath": "bausteinsicht-lsp",
  "bausteinsicht.debug": false,
  "bausteinsicht.drawioUrl": "https://app.diagrams.net"
}
```

## Architecture

```
VS Code Extension (TypeScript)
  ↓ (LSP Client)
bausteinsicht-lsp (Go)
  ↓ (exec)
bausteinsicht validate, export, etc.
```

- **Extension**: Spawns LSP server process, connects via stdin/stdout
- **LSP Server**: Handles validation, CodeLens, diagnostics
- **CLI**: Does the actual work (validate, export, sync)

## Development

### Build from Source

```bash
cd vscode-extension
npm install
npm run build
```

### Run in Dev Mode

```bash
npm run watch          # Auto-recompile on changes
code --extensionDevelopmentPath=. 
```

### Test

```bash
npm test
```

## Contributing

See [CONTRIBUTING.md](../CONTRIBUTING.md) for development guidelines.

## License

MIT — See [LICENSE](../LICENSE) for details.
