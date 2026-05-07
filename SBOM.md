# Software Bill of Materials (SBOM)

A Software Bill of Materials (SBOM) is a complete, formal record of all components used in Bausteinsicht.

## What is in the SBOM?

The SBOM documents all Go dependencies and their versions, including:
- **Direct dependencies** — explicitly imported packages
- **Transitive dependencies** — dependencies of dependencies
- **License information** — compliance and attribution
- **Vulnerability data** — known CVEs (when available)

## Available SBOM Formats

We provide SBOMs in two formats for maximum compatibility:

### SPDX Format (`sbom.spdx.json`)
- **Standard**: SPDX 2.3 (ISO/IEC 5962:2021)
- **Use case**: Wide compatibility across tools, regulatory compliance
- **Tools**: CycloneDX converters, vulnerability scanners (Grype, Trivy)
- **Spec**: https://spdx.dev/

### CycloneDX Format (`sbom.cyclonedx.xml`)
- **Standard**: CycloneDX 1.4
- **Use case**: Container scanning, supply chain analytics
- **Tools**: Dependency-Check, BlackDuck, OWASP tools
- **Spec**: https://cyclonedx.org/

## How to Use the SBOM

### 1. Audit Dependencies

Download the SBOM from a release and inspect with tools:

```bash
# Using SPDX tools
spdx-tools validate sbom.spdx.json

# Using CycloneDX tools
cyclonedx-cli validate --input-file sbom.cyclonedx.xml
```

### 2. Check for Known Vulnerabilities

```bash
# Using Grype (SPDX format)
grype sbom:sbom.spdx.json

# Using Trivy (CycloneDX format)
trivy sbom sbom.cyclonedx.xml
```

### 3. Integrate into Your Compliance Process

Many organizations require SBOMs for:
- **Vendor security assessments** — prove provenance
- **License compliance audits** — verify open-source licenses
- **Supply chain risk analysis** — identify known CVEs
- **Regulatory compliance** — NIST SSDF, ISO 27001

## Dependency Policy

### Update Cadence

- **Security patches**: Merged within 48 hours of availability
- **Minor version updates**: Quarterly review
- **Major version updates**: Evaluated for breaking changes

### Vulnerability Response

See link:SECURITY.md[SECURITY.md] for vulnerability disclosure and response process.

### Direct Dependencies

Bausteinsicht intentionally minimizes direct dependencies:

| Package | Version | Justification |
|---------|---------|---|
| `github.com/spf13/cobra` | Latest | CLI framework (required) |
| `github.com/beevik/etree` | Latest | XML processing for draw.io (required) |
| `pgregory.net/rapid` | Latest | Property-based testing (dev-only) |

All other dependencies are transitive (pulled in by the above).

## Supply Chain Security (SLSA)

This project aims to comply with [SLSA (Supply-chain Levels for Software Artifacts)](https://slsa.dev/) Level 2+:

- ✅ **Provenance**: Git commit hash included in binary
- ✅ **Reproducibility**: Deterministic builds (pinned go.mod)
- ✅ **Signed releases**: GitHub releases signed with repository key (v1.0.0+)
- ✅ **SBOM publication**: Available for all releases
- 🔄 **Code review**: Required before merge (see CONTRIBUTING.md)
- 🔄 **Signed commits**: Recommended (see SECURITY.md)

## Tooling

SBOMs are generated automatically during the release process using:

- **Syft**: https://github.com/anchore/syft — generates SBOMs from source/binary/container
- **GoReleaser**: https://goreleaser.com — orchestrates build, sign, and release

## FAQ

**Q: Does Bausteinsicht have any critical dependencies?**  
A: No. The only critical dependencies are Cobra (CLI) and etree (XML). Both are mature, widely-used, and actively maintained.

**Q: How do I report a vulnerability in a Bausteinsicht dependency?**  
A: See link:SECURITY.md[SECURITY.md] — submit a GitHub Security Advisory.

**Q: Can I use Bausteinsicht in an air-gapped environment?**  
A: Yes. Bausteinsicht has no runtime network dependencies. All dependencies are vendored in `go.mod`.

**Q: How often are SBOMs updated?**  
A: For each release. Patch releases (v1.0.1, v1.0.2) generate new SBOMs reflecting any dependency updates.

## Related Documents

- link:SECURITY.md[SECURITY.md] — Vulnerability disclosure, signed releases
- link:CONTRIBUTING.md[CONTRIBUTING.md] — Contributing guidelines, code review policy
