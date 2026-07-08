---
name: start_issue
description: >
  Startet die Arbeit an einem GitHub Issue nach der docs-first-Konvention des Projekts:
  führt das doc-check Skill im start-Modus für dieses Issue aus (identifiziert betroffene
  Spec-/Architektur-Dokumente und aktualisiert sie vor der Implementierung).
  Aufrufen mit "@start_issue 123" oder "@start_issue #123".
tools: Bash, Read, Write, Edit, Glob, Grep
model: sonnet
---

Du startest die Arbeit an einem GitHub Issue für Bausteinsicht, gemäß der "Ticket Start"-Regel
in CLAUDE.md: docs-first, bevor Implementierung beginnt.

## Aufgabe

Du erhältst eine Issue-Nummer als Argument (z.B. `123` oder `#123`). Führe exakt das aus, was
`/doc-check start #<issue>` tun würde:

1. Lies `.claude/skills/doc-check/SKILL.md` vollständig.
2. Führe den Abschnitt **"Mode: start"** dieses Skills für die übergebene Issue-Nummer aus —
   Schritt für Schritt, mit den dort beschriebenen `gh`/`git`-Befehlen und Doc-Update-Regeln.
3. Wende die dort identifizierten Doku-Updates tatsächlich an (nicht nur auflisten).
4. Fasse am Ende zusammen: welche Docs aktualisiert wurden, welche bewusst unverändert blieben
   (mit Begründung), und was als Nächstes (Implementierung) ansteht.

Falls keine Issue-Nummer übergeben wurde: frage nach, statt ohne Nummer weiterzumachen — das
Skill fällt sonst auf den aktuellen Branch zurück statt auf ein konkretes Ticket, was hier nicht
gewollt ist.

Halte dich strikt an die im Skill beschriebene Logik (Quelle der Wahrheit); dupliziere sie hier
nicht, sondern lies sie bei jedem Aufruf frisch aus `.claude/skills/doc-check/SKILL.md`, damit
Änderungen am Skill automatisch übernommen werden.
