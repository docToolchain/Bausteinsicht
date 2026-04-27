---
title: "feat: VS Code Extension"
labels: enhancement
---

## Beschreibung

Eine VS Code Extension (`bausteinsicht.bausteinsicht`) die den kompletten Bausteinsicht-Workflow direkt in den Editor bringt: Live-Diagram-Preview, Inline-Validierungsfehler, CodeLens-Links zu Tests und ADRs, Command Palette für alle CLI-Operationen, und Statusbar-Integration für Watch-Modus.

## Motivation

- Architekten verlassen VS Code nie wirklich — alle Workflows (Modell editieren, Diagramm prüfen, sync auslösen) sollten im Editor möglich sein
- Inline-Validierung (rote Unterstreichungen bei `"kind": "unknown"`) ist deutlich ergonomischer als Terminal-Output
- CodeLens-Links von Elementen zu ihren Tests und ADRs bauen Brücken zwischen Architektur und Implementierung
- Die Extension ist ein dünner UI-Wrapper — alle Business-Logik bleibt im Go CLI

## Proposed Implementation

**Features:**

1. **Live-Preview:** `Ctrl+Shift+P → "Bausteinsicht: Open Diagram Preview"` — Webview mit draw.io Viewer, auto-refresh bei Änderungen, View-Selector

2. **Inline-Diagnostics:** `bausteinsicht validate --format json` bei Save → VS Code Diagnostic Collection mit Errors/Warnings direkt im Editor

3. **CodeLens** über Elementen:
```jsonc
// 🔗 Open in draw.io  |  📋 ADR-012  |  ✅ 3 tests  |  🟡 deprecated
{ "id": "order-service", "kind": "service", "status": "deprecated" }
```

4. **Command Palette:** alle `bausteinsicht` Befehle (sync, validate, lint, health, add element, ...)

5. **Statusbar:** `$(sync~spin) Bausteinsicht: watching | Last sync: 14:32 | ✅ 12 elements`

6. **Snippets:** `bbs-element`, `bbs-relationship`, `bbs-view`, `bbs-constraint`

**Technology:** TypeScript + VS Code Extension API (thin wrapper around Go CLI).

**Distribution:** VS Code Marketplace + `.vsix` from GitHub Releases.

## Implementation Plan

See the implementation plan embedded in the GitHub issue.

## Affected Components

- `vscode-extension/` — new directory (TypeScript, separate from Go codebase)
- Go CLI: no changes needed (extension calls existing CLI)
- CI: new workflow for extension packaging and marketplace publish
