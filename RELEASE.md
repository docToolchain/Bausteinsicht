# Release Process

Releases use a **B+D hybrid trigger**: a milestone reaching 100% completion is the signal (D), but Claude waits for explicit user confirmation before tagging (B). GitHub Actions handles all build, packaging, and publish steps automatically when a tag is pushed.

## Versioning

Semver (MAJOR.MINOR.PATCH) derived from Conventional Commits since the last tag:

| Commit prefix | Version bump |
|---|---|
| `fix:`, `refactor:`, `perf:`, `docs:`, `style:`, `test:`, `chore:` | patch `v1.1.x` |
| `feat:` | minor `v1.x.0` |
| `feat!:` or footer `BREAKING CHANGE:` | major `vX.0.0` |

Highest-priority rule wins. If only patch-level commits are present → patch. If any `feat:` → minor. If any breaking → major.

## Trigger: Milestone Completion + Manual Confirmation (Options B + D)

### Option D — Milestone-based detection

A GitHub Milestone represents one release. When all issues in the milestone are closed, that is the signal that the release is ready.

Claude checks milestone readiness on request or periodically via `/loop`:

```
Prüfe Milestone <N> (<Name>): Sind alle Issues geschlossen?
- gh api repos/docToolchain/Bausteinsicht/milestones/<N> --jq '{open: .open_issues, closed: .closed_issues}'
- gh issue list --milestone <N> --state open
Falls open_issues == 0: Schlage ein Release vor und warte auf Bestätigung.
Falls noch Issues offen: Liste sie auf und stoppe.
```

### Option B — Manual confirmation gate

Claude never tags autonomously. After detecting milestone completion (D), it presents the proposed version and waits for explicit user confirmation before proceeding:

```
Milestone <N> ist vollständig (X Issues geschlossen, 0 offen).
Vorgeschlagene Version: vX.Y.Z (begründung aus git log)

Highlights:
- ...

Soll ich den Tag erstellen und pushen? [ja/nein]
```

Only after confirmation does Claude create the tag and push.

**Readiness check before releasing:**
1. All issues in the target milestone are closed (`gh milestone view <N>`)
2. `main` branch CI is green (`gh run list --branch main --limit 3`)
3. No open PRs targeting this milestone

If any milestone issues are still open, do not release — either close them or move them to the next milestone.

## Claude Prompt

Use this prompt to check milestone status and trigger a release:

```
Prüfe Milestone <N> (<Name>) auf Release-Bereitschaft.

Schritte:
1. Prüfe ob alle Issues in Milestone <N> geschlossen sind:
   gh api repos/docToolchain/Bausteinsicht/milestones/<N> --jq '{open: .open_issues, closed: .closed_issues}'
   Falls open_issues > 0: Liste offene Issues auf und stoppe hier.
2. Prüfe ob main CI grün ist: gh run list --branch main --limit 3
3. Bestimme die neue Version mit: git log <last-tag>..HEAD --oneline
4. Schreibe eine kurze Zusammenfassung der Highlights
5. Präsentiere Version + Highlights und warte auf Bestätigung
6. Nach Bestätigung: git tag -a vX.Y.Z -m "Release vX.Y.Z\n\n<summary>"
7. Pushe den Tag: git push origin vX.Y.Z
8. Warte auf den GitHub Actions Release-Run
9. Erweitere die Release Notes auf GitHub mit kuratierter Einleitung
10. Lege einen neuen Milestone für die nächste Version an
```

## What GitHub Actions Produces

`.github/workflows/release.yml` runs **goreleaser** on tag push:

- Cross-compiles for linux/darwin/windows × amd64/arm64/arm
- Injects version via `-X main.version={{.Version}}`
- Emits SPDX + CycloneDX SBOMs (SLSA L2) and `checksums.txt`
- Creates GitHub Release with changelog grouped by Conventional Commit prefix

## Release Notes Format

goreleaser's auto-changelog is one line per commit. Always prepend a curated intro above it:

```markdown
## Highlights

<1-3 themes, what this release improves for users>

### New Commands

| Command | Description |
|---------|-------------|
| `bausteinsicht foo` | Does X |

## Install

```bash
# Download from GitHub Releases, then:
go install github.com/docToolchain/Bausteinsicht/cmd/bausteinsicht@vX.Y.Z
```

## Verify

```bash
sha256sum -c checksums.txt
```

---
<!-- auto-generated changelog below -->
```

See [v1.2.0](https://github.com/docToolchain/Bausteinsicht/releases/tag/v1.2.0) as reference shape.

## Pre-Release Checklist

- [ ] All milestone issues closed
- [ ] `main` CI green (build + test + lint + SonarCloud)
- [ ] Version bump correct (check `git log <last-tag>..HEAD --oneline`)
- [ ] Tag message includes short summary
- [ ] GitHub Actions release run succeeded
- [ ] Release notes enriched with curated intro

## If a Release Fails

Fix the cause on `main` via PR, then move the tag:

```bash
git tag -d vX.Y.Z && git push origin :refs/tags/vX.Y.Z
git tag -a vX.Y.Z -m "..." && git push origin vX.Y.Z
```
