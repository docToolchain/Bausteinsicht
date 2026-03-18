---
title: "feat: draw.io Template Generator from Spec"
labels: enhancement
---

## Beschreibung

Automatically generate a draw.io template file from `spec.elementKinds`. Currently, teams must manually maintain a template file with shapes and colors per element kind. This command generates a ready-to-use template directly from the spec — ensuring templates stay in sync with the model automatically.

## Motivation

- The draw.io template file is the most common source of drift: teams add a new element kind to spec but forget to update the template
- Manual template creation requires draw.io expertise; most architects don't know the XML format
- A generated template removes setup friction for new projects (currently a pain point in `bausteinsicht init`)
- Multiple visual presets (`default`, `c4`, `minimal`, `dark`) serve different presentation contexts

## Proposed Implementation

**New command:**
```
bausteinsicht generate-template [--style default|c4|minimal|dark] [--output architecture-template.drawio]
```

**What gets generated:**
- One example element per defined kind, arranged in a grid
- Correct fill/stroke colors from the Bausteinsicht palette
- Correct shape per kind (person → archimate actor, database → flowchart cylinder, etc.)
- Ready to use as the `--template` input for `sync`

**Integration with `init`:**
```bash
bausteinsicht init --generate-template
# → Creates both architecture.jsonc AND architecture-template.drawio
```

**Custom shape support:** element kinds can declare `"shape": "mxgraph.cisco.servers.standard_server"` in spec for custom draw.io shapes.

## Implementation Plan

See [`docs/plans/2026-03-18-drawio-template-generator.md`](../plans/2026-03-18-drawio-template-generator.md)

## Affected Components

- `internal/template/generator.go` — new
- `internal/template/shapes.go` — kind→shape mapping + presets
- `cmd/bausteinsicht/generate_template.go` — new command
- `cmd/bausteinsicht/init.go` — `--generate-template` flag
