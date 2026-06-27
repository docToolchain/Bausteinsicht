# Release Process

Releases are triggered manually on request. GitHub Actions handles all build, packaging, and publish steps automatically when a tag is pushed.

## Versioning

Semver (MAJOR.MINOR.PATCH) derived from Conventional Commits since the last tag:

| Commit prefix | Version bump |
|---|---|
| `fix:`, `refactor:`, `perf:`, `docs:`, `style:`, `test:`, `chore:` | patch `v1.1.x` |
| `feat:` | minor `v1.x.0` |
| `feat!:` or footer `BREAKING CHANGE:` | major `vX.0.0` |

Highest-priority rule wins. If only patch-level commits are present → patch. If any `feat:` → minor. If any breaking → major.

## Trigger: Manual on Request (Option B)

Releases are created by asking Claude: **"make a release"** or **"release for milestone N"**.

**Readiness check before releasing:**
1. All issues in the target milestone are closed (`gh milestone view <N>`)
2. `main` branch CI is green (`gh run list --branch main --limit 3`)
3. No open PRs targeting this milestone

If any milestone issues are still open, do not release — either close them or move them to the next milestone.

## Claude Prompt

Use this prompt to trigger a release:

```
Erstelle ein Release für Milestone <N> (<Name>).

Schritte:
1. Prüfe ob alle Issues in Milestone <N> geschlossen sind
2. Bestimme die neue Version mit: git log <last-tag>..HEAD --oneline
3. Schreibe eine kurze Zusammenfassung der Highlights
4. Erstelle den Tag: git tag -a vX.Y.Z -m "Release vX.Y.Z\n\n<summary>"
5. Pushe den Tag: git push origin vX.Y.Z
6. Warte auf den GitHub Actions Release-Run
7. Erweitere die Release Notes auf GitHub mit kuratierter Einleitung
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
