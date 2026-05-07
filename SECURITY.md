# Security Policy

## Reporting Vulnerabilities

If you discover a security vulnerability in Bausteinsicht, please report it responsibly using GitHub's Security Advisory mechanism.

### How to Report

1. **Do NOT open a public issue** — this exposes the vulnerability before a patch is available
2. **Use GitHub Security Advisory**: https://github.com/docToolchain/Bausteinsicht/security/advisories
3. **Include**:
   - Description of the vulnerability
   - Affected versions (or branch)
   - Steps to reproduce (if possible)
   - Potential impact
   - Suggested fix (optional)

### Response Timeline

We aim to:
- **Acknowledge** your report within 24 hours
- **Confirm/triage** within 48 hours
- **Release patch** within 5 business days (for critical issues)
- **Publicly disclose** after patching or 90 days (whichever is sooner)

## Supported Versions

| Version | Status | End of Life |
|---------|--------|---|
| 1.0.x | LTS | 2027-05-01 |
| 0.9.x | EOL | 2026-01-01 |
| < 0.9 | EOL | Immediate |

Only the latest minor version receives security updates.

## Vulnerability Disclosure

### Public Disclosures

Once patched, vulnerabilities are disclosed in the repository's **Security** tab with:
- CVE-ID (if applicable)
- Affected versions
- Impact assessment
- Patch details
- Credits to reporter (unless requested otherwise)

### Do Not Publicly Disclose Until

1. A patch has been released, **AND**
2. A reasonable time (typically 48 hours) has passed for users to patch

## Security Standards

### Supply Chain Security

Bausteinsicht follows [SLSA (Supply-chain Levels for Software Artifacts)](https://slsa.dev/) Level 2:

- ✅ **Source control**: Git commits to GitHub
- ✅ **Provenance**: Release artifacts include commit hash (`-X main.version={{.Version}}`)
- ✅ **SBOM**: Software Bill of Materials included in all releases
- ✅ **Code review**: All changes reviewed before merge (see CONTRIBUTING.md)
- ✅ **Signed commits** (recommended): `git config commit.gpgsign true`
- ✅ **Signed releases**: All v1.0.0+ releases are cryptographically signed

### Signed Commits

While optional, we **strongly encourage** signed commits:

```bash
# Configure Git to sign commits
git config --global commit.gpgsign true

# Or sign per-commit
git commit -S -m "your message"

# Verify signatures
git log --pretty=format:"%h %G? %s"  # %G? shows valid (G), bad (B), or untrusted (U)
```

GitHub will show a "Verified" badge next to signed commits, indicating:
- The committer has proven their identity
- The commit has not been tampered with

### Signed Releases

For v1.0.0 and later, all releases are signed using the repository's default signing key:

```bash
# Verify release signature
gh release view v1.0.0 --json assets
# Check GitHub UI for "Verified" badge

# Download and verify checksums
curl -O https://github.com/docToolchain/Bausteinsicht/releases/download/v1.0.0/checksums.txt
shasum -a 256 bausteinsicht_1.0.0_linux_amd64 | grep -f checksums.txt
```

## Dependency Management

### Automated Scanning

We use GitHub's [Dependabot](https://dependabot.com/) to:
- Monitor dependencies daily
- Alert on security updates
- Open PRs with patches (security priority)
- Detect outdated versions

### Vulnerability Scanning

All PR builds run:
- `go govulncheck` — scan for known Go vulnerabilities (go.dev/vuln)
- `gosec` — static analysis for security issues

### Policy

1. **Critical** (CVSS ≥ 9.0): Patch within 24 hours
2. **High** (CVSS 7.0-8.9): Patch within 5 days
3. **Medium** (CVSS 4.0-6.9): Patch in next minor release
4. **Low** (CVSS < 4.0): Patch with other maintenance

Exceptions may apply for:
- Vulnerabilities only affecting development dependencies (test tools, CI)
- Vulnerabilities requiring Go runtime update (outside project control)

## Known Security Constraints

### By Design

Bausteinsicht is a local CLI tool with these characteristics:
- No network communication (except downloading releases)
- No remote authentication
- No user database or persistent storage
- No server component

### Potential Risks

| Risk | Mitigation | Status |
|------|-----------|--------|
| Malicious JSONC model files | Validate structure, bounds-check recursion depth | ✅ Implemented |
| Path traversal via --model flag | SEC-001 path containment checks | ✅ Implemented |
| XML billion laughs attack | Reasonable DTD limits in etree | ✅ Implemented |
| ReDoS in validation patterns | Static regex patterns, no user input | ✅ Safe |

## Security Incident Response

### If Bausteinsicht is Exploited

1. **Report immediately** via GitHub Security Advisory
2. **Do not open public issues** 
3. **Include**: How the exploit works, when it was discovered
4. **Do not publish details** until patch is available

We will:
- Acknowledge receipt within 24 hours
- Investigate and develop a patch
- Release a patch release (e.g., v1.0.1)
- Publicly disclose after patching (or 90 days max)

## Security Contacts

For security issues, contact maintainers via:
- **GitHub Security Advisory**: https://github.com/docToolchain/Bausteinsicht/security/advisories
- **Email**: [See repository contact info or CONTRIBUTING.md]

**Do not use public issues or pull requests for security disclosures.**

## Related Documents

- [SBOM.md](SBOM.md) — Software Bill of Materials, dependency audit
- [CONTRIBUTING.md](CONTRIBUTING.md) — Contributing guidelines, code review policy
- [SLSA.dev](https://slsa.dev/) — Supply chain security framework
- [OWASP Top 10](https://owasp.org/Top10/) — Common security risks
