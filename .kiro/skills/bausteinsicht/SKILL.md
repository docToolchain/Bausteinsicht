---
name: bausteinsicht
description: >
  Work with the Bausteinsicht architecture-as-code CLI. Create and modify JSONC architecture models,
  synchronize with draw.io diagrams, validate models, and export to various formats.
  Use this skill when the user wants to work with architecture models, draw.io diagrams,
  or the Bausteinsicht CLI.
license: MIT
compatibility:
  os: [linux, macos, windows]
  tools: [bausteinsicht]
metadata:
  author: docToolchain
  version: "1.0"
allowed-tools: Bash Read Write Edit Glob Grep
argument-hint: "<task description, e.g. 'add a cache container to the webshop system'>"
---

# Bausteinsicht CLI Skill

You are an expert in using the **Bausteinsicht** architecture-as-code CLI tool. Bausteinsicht uses JSONC models as the single source of truth and synchronizes bidirectionally with draw.io diagrams.

## CLI Commands

### Project Setup

```bash
# Initialize a new project (creates architecture.jsonc, template.drawio, architecture.drawio)
bausteinsicht init --format json

# Validate the model for errors and warnings
bausteinsicht validate --format json
```

### Adding Elements

```bash
# Add a top-level element
bausteinsicht add element --id <id> --kind <kind> --title "<title>" [--technology "<tech>"] [--description "<desc>"]

# Add a child element (nested under parent)
bausteinsicht add element --id <id> --kind <kind> --title "<title>" --parent <parent.path> [--technology "<tech>"]
```

**Rules:**
- `--id` must match `[a-zA-Z][a-zA-Z0-9_-]*` (no dots — dots are hierarchy separators)
- `--kind` must be defined in the model's `specification.elements`
- `--parent` uses dot notation (e.g., `onlineshop.api`)
- Parent's kind must have `"container": true` in the specification

### Adding Relationships

```bash
bausteinsicht add relationship --from <element.path> --to <element.path> [--label "<label>"] [--kind "<kind>"] [--description "<desc>"]
```

**Rules:**
- `--from` and `--to` use dot notation referencing existing elements
- `--kind` must be defined in `specification.relationships` (if provided)
- Duplicate relationships (same from/to/kind) are rejected

### Synchronization

```bash
# Bidirectional sync between model and draw.io
bausteinsicht sync --format json

# Watch mode: auto-sync on file changes
bausteinsicht watch
```

**Sync behavior:**
- Forward sync: Model changes → draw.io diagram updates
- Reverse sync: draw.io edits (position, size, label) → model updates
- New elements appear with red dashed border (needs manual positioning)
- Exit codes: 0=success, 1=conflicts resolved, 2=error

### Export

```bash
# Export views as PNG/SVG images (requires draw.io CLI)
bausteinsicht export [--image-format png|svg] [--view <viewID>] [--output <dir>] [--scale 2.0]

# Export elements as AsciiDoc or Markdown table
bausteinsicht export-table [--table-format adoc|markdown] [--output <file>]

# Export views as PlantUML C4 or Mermaid diagrams
bausteinsicht export-diagram [--diagram-format plantuml|mermaid] [--view <viewID>] [--output <dir>]
```

### Global Flags

| Flag | Description |
|------|-------------|
| `--format json\|text` | Output format (default: text). Use `json` for machine-readable output. |
| `--model <path>` | Path to model file. Auto-detected if omitted (looks for `architecture.jsonc`). |
| `--template <path>` | Path to draw.io template. Must have `.drawio` extension. |
| `--verbose` | Enable verbose output. |

## JSONC Model Structure

The architecture model has four sections:

```jsonc
{
  "$schema": "https://raw.githubusercontent.com/docToolchain/Bausteinsicht/main/schema/bausteinsicht.schema.json",

  // 1. Specification: Define element kinds and relationship kinds
  "specification": {
    "elements": {
      "actor":     { "notation": "Actor", "description": "A person or external system" },
      "system":    { "notation": "Software System", "container": true },
      "container": { "notation": "Container", "container": true },
      "component": { "notation": "Component", "container": true }
    },
    "relationships": {
      "uses":     { "notation": "uses" },
      "includes": { "notation": "includes", "dashed": true }
    }
  },

  // 2. Model: Element hierarchy with unique variable names as IDs
  "model": {
    "customer": {
      "kind": "actor",
      "title": "Customer",
      "description": "End user of the webshop"
    },
    "onlineshop": {
      "kind": "system",
      "title": "Online Shop",
      "children": {
        "frontend": { "kind": "container", "title": "Web Frontend", "technology": "React" },
        "api":      { "kind": "container", "title": "REST API", "technology": "Go" },
        "db":       { "kind": "container", "title": "Database", "technology": "PostgreSQL" }
      }
    }
  },

  // 3. Relationships: Connections between elements (dot notation)
  "relationships": [
    { "from": "customer", "to": "onlineshop.frontend", "label": "uses", "kind": "uses" },
    { "from": "onlineshop.frontend", "to": "onlineshop.api", "label": "calls", "kind": "uses" },
    { "from": "onlineshop.api", "to": "onlineshop.db", "label": "reads/writes", "kind": "uses" }
  ],

  // 4. Views: What to show on each draw.io page
  "views": {
    "context": {
      "title": "System Context",
      "include": ["customer", "onlineshop"],
      "description": "High-level view showing the system and its users"
    },
    "containers": {
      "title": "Container View",
      "scope": "onlineshop",
      "include": ["customer", "onlineshop.*"],
      "description": "Internal structure of the Online Shop"
    }
  }
}
```

### Key Concepts

- **Element IDs** are the JSONC object keys (e.g., `customer`, `frontend`). Nested elements are referenced via dot notation: `onlineshop.api.catalog`.
- **`container: true`** in the specification allows an element kind to have children.
- **Views** control what appears on each draw.io diagram page:
  - `include`: List of element IDs or glob patterns (`onlineshop.*`)
  - `scope`: Element whose boundary is drawn as a swimlane
  - `exclude`: Elements to hide
- **Relationships** are automatically shown when both `from` and `to` elements are visible in a view.

## Typical Workflows

### Create a new architecture model

```bash
bausteinsicht init
# Edit architecture.jsonc to define your model
bausteinsicht validate
bausteinsicht sync
```

### Add elements and sync

```bash
# Add a new system
bausteinsicht add element --id payment --kind system --title "Payment Service" --technology "Stripe"

# Add a child container
bausteinsicht add element --id gateway --kind container --title "Payment Gateway" --technology "Go" --parent payment

# Add a relationship
bausteinsicht add relationship --from onlineshop.api --to payment.gateway --label "processes payments" --kind uses

# Sync to update draw.io
bausteinsicht sync
```

### Export for documentation

```bash
# PNG images for embedding in docs
bausteinsicht export --image-format png --scale 2.0

# AsciiDoc table for arc42 documentation
bausteinsicht export-table --table-format adoc --output elements.adoc

# PlantUML C4 diagrams
bausteinsicht export-diagram --diagram-format plantuml
```

## Tips

- Always use `--format json` when scripting or when processing output programmatically.
- Run `bausteinsicht validate` after manual model edits to catch errors early.
- Use `bausteinsicht watch` during development for continuous sync.
- The `$schema` property enables IDE autocompletion in VS Code, IntelliJ, and other editors.
- New elements synced to draw.io appear with a red dashed border — position them manually, then re-sync.
