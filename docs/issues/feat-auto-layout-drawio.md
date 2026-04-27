---
title: "feat: Auto-Layout in draw.io"
labels: enhancement
---

## Beschreibung

Nach dem Forward-Sync werden neue Elemente an Standardpositionen eingefügt (oft überlappend oder außerhalb des sichtbaren Bereichs). Ein `layout` Befehl berechnet ein hierarchisches, kräftebasiertes oder radiales Layout und schreibt die Koordinaten in die draw.io-Datei. Manuell positionierte Elemente werden durch einen Pin-Mechanismus geschützt.

## Motivation

- Das häufigste manuelle Follow-up nach `bausteinsicht sync` ist das Neuanordnen von Elementen in draw.io
- Auto-Layout ist besonders wertvoll in Watch-Modus — neue Elemente erscheinen sofort sinnvoll positioniert
- Drei Algorithmen decken verschiedene Modell-Topologien ab: hierarchisch (Schichten), kräftebasiert (vernetzt), radial (fokussierte Views)
- Der Pin-Mechanismus stellt sicher, dass manuell justierte Layouts nicht überschrieben werden

## Proposed Implementation

**New command:**
```
bausteinsicht layout [--algorithm hierarchical|force|radial] [--view <key>] [--preserve-pinned]
```

**Integration in existing commands:**
```
bausteinsicht sync  --auto-layout
bausteinsicht watch --auto-layout
```

**Algorithms:**
- `hierarchical` (default): layered top-to-bottom, minimiert Edge-Kreuzungen
- `force`: Federsimulation für zirkuläre/vernetzte Topologien
- `radial`: Scope-Element im Zentrum, Abhängigkeiten in konzentrischen Ringen

**Pin mechanism:** Element in draw.io mit Custom Property `bausteinsicht-pinned=true` markieren → wird beim Layout nicht bewegt.

**Smart sync integration:** nur wirklich neue Elemente (Position = Default 10,10) werden repositioniert — bestehende manuell positionierte Elemente bleiben unverändert.

## Implementation Plan

See [`docs/plans/2026-03-18-auto-layout-drawio.md`](../plans/2026-03-18-auto-layout-drawio.md)

## Affected Components

- `internal/layout/` — new package (hierarchical, force, radial, pin handling)
- `internal/sync/forward.go` — apply layout after sync
- `cmd/bausteinsicht/layout.go` — new command
- `cmd/bausteinsicht/sync.go` + `watch.go` — `--auto-layout` flag
