# Bausteinsicht VS Code Extension — Setup & Configuration Guide

Dieses Dokument führt dich Schritt-für-Schritt durch Installation, Konfiguration und tägliche Nutzung der Bausteinsicht VS Code Extension.

## 📋 Voraussetzungen

- **VS Code** 1.85 oder später
- **Node.js** 18+ (für Development)
- **Bausteinsicht CLI** — das LSP Server Backend

## 🚀 Installation

### Option 1: Aus VS Code Marketplace (empfohlen)

1. Öffne VS Code
2. Gehe zu **Extensions** (Ctrl+Shift+X / Cmd+Shift+X)
3. Suche nach "Bausteinsicht"
4. Klicke **Install**

Die Extension wird automatisch aktiviert, wenn sie eine `architecture.jsonc` Datei findet.

### Option 2: Manuelle Installation aus .vsix

```bash
# Download .vsix aus GitHub Releases
wget https://github.com/docToolchain/Bausteinsicht/releases/download/v0.1.0/bausteinsicht-0.1.0.vsix

# Installation
code --install-extension bausteinsicht-0.1.0.vsix
```

### Option 3: Development Installation (lokal bauen)

```bash
# Repository klonen
git clone https://github.com/docToolchain/Bausteinsicht.git
cd Bausteinsicht/vscode-extension

# Dependencies installieren
npm install

# Extension im Development Mode laden
code --extensionDevelopmentPath=.
```

## 🔧 Schritt 1: Bausteinsicht CLI installieren

Die VS Code Extension ist nur ein Frontend. Das eigentliche Backend ist das **bausteinsicht-lsp** Binary (LSP Server in Go).

### Installation der CLI

**Homebrew (macOS/Linux):**
```bash
brew install docToolchain/tap/bausteinsicht
```

**Winget (Windows):**
```powershell
winget install docToolchain.Bausteinsicht
```

**Scoop (Windows):**
```powershell
scoop bucket add docToolchain https://github.com/docToolchain/scoop-bucket
scoop install bausteinsicht
```

**Aus Source bauen:**
```bash
cd Bausteinsicht
make build
make install
```

**Verify Installation:**
```bash
bausteinsicht-lsp --version
```

## ⚙️ Schritt 2: Extension konfigurieren

Die Extension sucht standardmäßig nach `bausteinsicht-lsp` im PATH. Falls das Binary an einem anderen Ort ist, musst du den Pfad konfigurieren.

### Settings öffnen

**VS Code GUI:**
1. Gehe zu **File** → **Preferences** → **Settings**
2. Suche nach "bausteinsicht"
3. Du siehst drei Einstellungen

**oder direkt in settings.json:**

```bash
# macOS/Linux
code ~/.config/Code/User/settings.json

# Windows
code %APPDATA%\Code\User\settings.json
```

### Konfigurierbare Einstellungen

```json
{
  // Pfad zum bausteinsicht-lsp Binary
  // Standard: "bausteinsicht-lsp" (sucht im PATH)
  "bausteinsicht.serverPath": "bausteinsicht-lsp",

  // Debug-Logging aktivieren
  // Standard: false
  // Setze auf true um LSP Server Logs in Debug Console zu sehen
  "bausteinsicht.debug": false,

  // draw.io URL für "Open in draw.io" Befehle
  // Standard: "https://app.diagrams.net"
  // Für selbstgehostete draw.io Instanz: "https://draw.example.com"
  "bausteinsicht.drawioUrl": "https://app.diagrams.net"
}
```

### Beispiel-Konfigurationen

**Standard (aus PATH):**
```json
{
  "bausteinsicht.serverPath": "bausteinsicht-lsp"
}
```

**Custom Pfad (macOS):**
```json
{
  "bausteinsicht.serverPath": "/usr/local/opt/bausteinsicht/bin/bausteinsicht-lsp"
}
```

**Custom Pfad (Windows):**
```json
{
  "bausteinsicht.serverPath": "C:\\Program Files\\Bausteinsicht\\bausteinsicht-lsp.exe"
}
```

**Mit Debug Logging:**
```json
{
  "bausteinsicht.serverPath": "bausteinsicht-lsp",
  "bausteinsicht.debug": true
}
```

## 📁 Schritt 3: Architecture Modell erstellen

1. Erstelle eine neue JSONC-Datei mit "architecture" im Namen:

```bash
# Standard-Konvention
touch architecture.jsonc

# Oder flexibel benannt (z.B. für mehrere Modelle)
touch my-architecture.jsonc
touch system-design.jsonc
touch platform-architecture.jsonc
```

**Die Extension aktiviert sich automatisch**, wenn der Dateiname "architecture" enthält.

2. Starte mit einer minimalen Architektur:

```jsonc
{
  "name": "My System",
  "description": "A sample architecture",
  "version": "1.0.0",
  "elements": {
    "system": {
      "kind": "System",
      "description": "The main system",
      "tags": "element"
    },
    "system.api": {
      "kind": "Container",
      "description": "REST API",
      "technology": "Go",
      "tags": "element"
    }
  },
  "relationships": [
    {
      "from": "system.api",
      "to": "system.database",
      "description": "reads/writes",
      "tags": "relationship"
    }
  ],
  "views": {
    "system-context": {
      "title": "System Context",
      "description": "High-level system overview",
      "include": ["system", "external"],
      "type": "landscape"
    }
  }
}
```

3. Die Extension wird automatisch aktiviert und zeigt:
   - ✅ Grünes Häkchen in der Status Bar: "Bausteinsicht: Connected"
   - 📝 Inline-Validierung (Fehler/Warnungen rot unterstrichen)
   - 🎯 CodeLens oberhalb von Element-Definitionen

## 🎮 Schritt 4: Extension nutzen

### Health Check (Verbindung testen)

1. Öffne Command Palette: **Ctrl+Shift+P** (Windows/Linux) oder **Cmd+Shift+P** (macOS)
2. Tippe: "Bausteinsicht: Health Check"
3. Erwartetes Ergebnis: Notification "✅ Bausteinsicht is healthy"

Wenn das nicht funktioniert, siehe **Troubleshooting** weiter unten.

### Validieren

```
Command Palette → "Bausteinsicht: Validate"
```

Zeigt alle Validierungsfehler in einer Benachrichtigung. Fehler werden auch als rote Underlines im Editor angezeigt.

### Sync mit draw.io

```
Command Palette → "Bausteinsicht: Sync Model"
```

Synchronisiert dein JSONC Modell mit einer draw.io Datei (falls konfiguriert).

### Watch Mode (Echtzeit-Überwachung)

```
Command Palette → "Bausteinsicht: Toggle Watch Mode"
```

Aktiviert Auto-Sync: Jede Änderung wird automatisch validiert und zu draw.io synchronisiert.

Status Bar zeigt: "Watch enabled" oder "Watch disabled"

### CodeLens nutzen

CodeLens-Links erscheinen über Element-Definitionen:

```
system.api — kind: Container, status: active, views: 2
├─ Open in draw.io
```

Klick auf "Open in draw.io" öffnet das Element in draw.io.

## 🐛 Troubleshooting

### Problem: "LSP server is not running"

**Symptom:** Status Bar zeigt "Bausteinsicht: Failed to connect"

**Lösungen:**
1. Stelle sicher, dass `bausteinsicht-lsp` im PATH ist:
   ```bash
   which bausteinsicht-lsp    # macOS/Linux
   where bausteinsicht-lsp    # Windows
   ```

2. Falls nicht gefunden, installiere das CLI:
   ```bash
   brew install docToolchain/tap/bausteinsicht
   ```

3. Falls anderer Pfad, konfiguriere `bausteinsicht.serverPath` in VS Code settings

4. Restart VS Code (Ctrl+Shift+P → "Developer: Reload Window")

### Problem: Extension wird nicht aktiviert

**Symptom:** Keine CodeLens, keine Validierungsfehler, obwohl Architecture-Datei vorhanden ist

**Lösungen:**
1. Stelle sicher, dass die Datei "architecture" im Namen hat:
   - ✅ `architecture.jsonc`
   - ✅ `my-architecture.jsonc`
   - ✅ `system-design-architecture.jsonc`
   - ❌ `model.jsonc` (kein "architecture" im Namen)
   
2. Dateiendung muss `.jsonc` sein (nicht `.json`)
3. Öffne Debug Console (Ctrl+Shift+Y) und suche nach Extension-Aktivierungsfehlern
4. Restart VS Code (Ctrl+Shift+P → Developer: Reload Window)

### Problem: "Cannot find module 'mocha'" (Development)

**Symptom:** npm test schlägt fehl mit "Cannot find module 'mocha'"

**Lösung:**
```bash
npm install
npm test
```

### Problem: LSP Server crasht beim Start

**Debug-Modus aktivieren:**
1. Setze `"bausteinsicht.debug": true` in settings.json
2. Öffne Debug Console (Ctrl+Shift+Y)
3. Suche nach LSP Error-Meldungen
4. Stelle sicher, dass das CLI korrekt installiert ist

### Problem: Port/Socket Fehler unter Windows

**Symptom:** "Failed to connect: pipe is busy" oder ähnlich

**Lösung:**
1. Alle VS Code Fenster schließen
2. Task Manager öffnen und `node.exe` Prozesse beenden
3. VS Code neustarten

## 📚 Weitere Ressourcen

- **Bausteinsicht Dokumentation:** https://github.com/docToolchain/Bausteinsicht/wiki
- **JSON Schema:** https://github.com/docToolchain/Bausteinsicht/raw/main/schemas/bausteinsicht.schema.json
- **CLI Referenz:** `bausteinsicht --help`

## 🔍 Debug & Entwicklung

### Debug Console Logs anschauen

```
Ctrl+Shift+Y → Wähle "Bausteinsicht LSP" aus der Dropdown
```

Zeigt:
- LSP Server Startmeldungen
- JSON-RPC Nachrichten
- Validierungsergebnisse
- Fehler und Warnungen

### LSP Server manuell starten (für Debugging)

```bash
# Terminal öffnen
bausteinsicht-lsp --debug

# In VS Code Extension Code ändern
# Reload Window: Ctrl+Shift+P → Developer: Reload Window
# Extension verbindet sich mit laufendem Server
```

### Extension von Source neu bauen

```bash
cd vscode-extension
npm install
npm run build
code --extensionDevelopmentPath=.
```

## ❓ FAQ

**F: Kann ich mehrere `architecture.jsonc` Dateien nutzen?**  
A: Ja! Die Extension funktioniert mit allen `.jsonc` Dateien, die das Bausteinsicht-Schema erfüllen.

**F: Funktioniert es mit anderen LSP-Clients (z.B. Vim, Neovim)?**  
A: Das `bausteinsicht-lsp` Binary ist ein Standard LSP Server. Theoretisch ja, aber wir unterstützen offiziell nur VS Code.

**F: Kann ich den LSP Server im Remote Development (SSH) nutzen?**  
A: Ja! Das Binary muss auf dem Remote-Host installiert sein. VS Code Remote Development funktioniert mit LSP.

**F: Muss ich die Extension installieren, oder kann ich nur die CLI nutzen?**  
A: Du kannst die CLI allein nutzen (`bausteinsicht validate`, `bausteinsicht export` etc.). Die Extension ist optional für IDE-Integration.

**F: Wo werden die Logs gespeichert?**  
A: LSP Logs gehen an stderr. Mit `--debug` Flag sichtbar in VS Code Debug Console.

---

**Brauchen Sie noch Hilfe?** Öffne ein Issue auf GitHub oder kontaktiere uns im docToolchain Discord.
