# Bausteinsicht VS Code Extension

🏗️ Architecture-as-Code with real-time VS Code integration via Language Server Protocol.

Bring your architecture models to life with instant validation, CodeLens metadata, and seamless draw.io integration.

## ✨ Features

✅ **Real-time Validation** — Inline error/warning diagnostics as you edit  
🎯 **CodeLens** — Element metadata at a glance (kind, status, view count)  
🔍 **draw.io Integration** — Click to open elements directly in draw.io  
📝 **Full JSONC Support** — Schema validation and autocompletion  
🔄 **Live Sync** — Watch mode for automatic model synchronization  

## 🚀 Quick Start

### 1. Install the Extension
```
VS Code Extensions (Ctrl+Shift+X) → Search "Bausteinsicht" → Install
```

### 2. Install the CLI Backend
```bash
# Homebrew
brew install docToolchain/tap/bausteinsicht

# Or: Winget, Scoop, or build from source
```

### 3. Create Architecture Model
```
Create a file with "architecture" in the name (e.g., architecture.jsonc, my-architecture.jsonc)
Extension auto-activates when it detects the file
```

### 4. Start Editing
- Extension auto-activates on `architecture.jsonc`
- See validation errors inline
- Use CodeLens to inspect elements

🎯 **For detailed setup instructions, see [SETUP.md](SETUP.md)**

## 📋 Requirements

- VS Code 1.85+
- Bausteinsicht CLI (`bausteinsicht-lsp` binary)

## ⚙️ Configuration

Configure in VS Code Settings (File → Preferences → Settings, search "bausteinsicht"):

```json
{
  // Path to bausteinsicht-lsp binary (default: searches PATH)
  "bausteinsicht.serverPath": "bausteinsicht-lsp",

  // Enable LSP debug logging (default: false)
  "bausteinsicht.debug": false,

  // draw.io URL for "Open in draw.io" commands
  "bausteinsicht.drawioUrl": "https://app.diagrams.net"
}
```

See [SETUP.md](SETUP.md#-schritt-2-extension-konfigurieren) for detailed configuration examples.

## 🎮 Commands

Open **Command Palette** (Ctrl+Shift+P / Cmd+Shift+P):

| Command | Description |
|---------|-------------|
| **Bausteinsicht: Health Check** | Verify LSP server connection |
| **Bausteinsicht: Validate** | Run validation on current model |
| **Bausteinsicht: Sync Model** | Sync changes to draw.io |
| **Bausteinsicht: Toggle Watch Mode** | Enable/disable auto-sync |
| **Bausteinsicht: Open in draw.io** | Open element in draw.io |

See [SETUP.md](SETUP.md#-schritt-4-extension-nutzen) for usage examples.

## 🏛️ Architecture

```
VS Code Extension (TypeScript)
  ↓ (LSP Client via JSON-RPC)
bausteinsicht-lsp (Go)
  ↓ (exec)
bausteinsicht validate, export, sync
```

- **Extension**: Spawns LSP server, displays diagnostics and CodeLens
- **LSP Server**: Validates models, provides metadata, publishes diagnostics
- **CLI**: Performs actual validation, export, synchronization

## 🛠️ Development

### Build from Source

```bash
cd vscode-extension
npm install        # Install dependencies
npm run build      # Compile TypeScript
npm run lint       # Run ESLint
npm run package    # Build .vsix artifact
```

### Run in Dev Mode

```bash
npm run watch              # Auto-recompile on changes
code --extensionDevelopmentPath=.
```

### Test

```bash
npm test                   # Run integration tests
npm test -- --grep "pattern"  # Run specific tests
```

## 🐛 Troubleshooting

**Extension won't activate?**  
→ Create a file named `architecture.jsonc` in your project root

**"LSP server is not running"?**  
→ Ensure `bausteinsicht-lsp` is in PATH: `which bausteinsicht-lsp`

**Debug mode enabled?**  
→ Set `"bausteinsicht.debug": true` in VS Code settings, check Debug Console (Ctrl+Shift+Y)

Full troubleshooting guide: See [SETUP.md](SETUP.md#-troubleshooting)

## 🔗 Resources

- **Setup Guide**: [SETUP.md](SETUP.md) — Detailed installation and configuration
- **Bausteinsicht Docs**: https://github.com/docToolchain/Bausteinsicht/wiki
- **JSON Schema**: https://github.com/docToolchain/Bausteinsicht/raw/main/schemas/bausteinsicht.schema.json
- **Issues**: https://github.com/docToolchain/Bausteinsicht/issues

## 📄 Contributing

See [CONTRIBUTING.md](../CONTRIBUTING.md) for development guidelines.

## 📜 License

MIT — See [LICENSE](../LICENSE) for details.

---

**Questions?** Open an issue on [GitHub](https://github.com/docToolchain/Bausteinsicht/issues) or reach out on [Discord](https://discord.gg/docToolchain).
