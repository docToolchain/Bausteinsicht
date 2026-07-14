#!/usr/bin/env bash
# scripts/fetch-xmi-testdata.sh — fetch the large XMI integration-test
# fixture (internal/importer/xmi/testdata/BigData.xmi, a real ~118MB
# Enterprise Architect/AUTOSAR export) from a GitHub Release asset on the
# separate, lightweight docToolchain/Bausteinsicht-Testdata repo, instead of
# committing or Git-LFS-tracking it in this repository (#553). A Release
# asset (not a committed/raw file) because the fixture is ~118MB — over
# GitHub's ~100MB per-file limit for a normal git blob.
#
# Usage:
#   scripts/fetch-xmi-testdata.sh
#   make fetch-testdata
#
# Safe to run repeatedly: skips the download if a real (>1MB) file is
# already present. Safe to fail: on any network/fetch/checksum error it
# prints a warning and exits 0 rather than failing the caller —
# internal/importer/xmi's TestImport_BigData already skips itself when the
# fixture is absent or too small, so offline/local dev is never blocked by
# this script. CI jobs that specifically need the fixture (see the
# xmi-bigdata-integration job in go.yml) check the test actually ran, so a
# silent fetch/checksum failure there is still caught — it just fails that
# one dedicated job, not the whole build.
#
# The downloaded bytes are verified against a pinned sha256 — Git LFS used to
# guarantee this at smudge time via the pointer's own oid; a plain curl fetch
# doesn't, so without this check a corrupted or unexpectedly replaced Release
# asset (still >1MB) would silently pass the size-only check and
# TestImport_BigData would run against the wrong fixture with no error.
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEST="$ROOT/internal/importer/xmi/testdata/BigData.xmi"
URL="https://github.com/docToolchain/Bausteinsicht-Testdata/releases/download/testdata-v1/BigData.xmi"
MIN_SIZE=1048576 # 1 MiB — matches TestImport_BigData's own presence check
EXPECTED_SHA256="7b761872b0f773caac76446b857162986761500d625c9bd3514d55bd1f63f5b0"

file_size() {
    stat -c%s "$1" 2>/dev/null || stat -f%z "$1" 2>/dev/null || echo 0
}

if [ -f "$DEST" ] && [ "$(file_size "$DEST")" -gt "$MIN_SIZE" ]; then
    echo "BigData.xmi already present ($(file_size "$DEST") bytes), skipping fetch."
    exit 0
fi

echo "Fetching BigData.xmi from docToolchain/Bausteinsicht-Testdata (release testdata-v1)..."
if curl -fsSL --retry 2 -o "$DEST.tmp" "$URL"; then
    ACTUAL_SHA256="$(sha256sum "$DEST.tmp" | cut -d' ' -f1)"
    if [ "$ACTUAL_SHA256" != "$EXPECTED_SHA256" ]; then
        echo "Warning: BigData.xmi checksum mismatch (expected $EXPECTED_SHA256, got $ACTUAL_SHA256) — discarding download." >&2
        echo "TestImport_BigData will skip itself; this is not treated as a hard failure here." >&2
        rm -f "$DEST.tmp"
        exit 0
    fi
    mv "$DEST.tmp" "$DEST"
    echo "Fetched and verified BigData.xmi ($(file_size "$DEST") bytes)."
else
    echo "Warning: could not fetch BigData.xmi from $URL (network issue or fixture repo unavailable)." >&2
    echo "TestImport_BigData will skip itself; this is not treated as a hard failure here." >&2
    rm -f "$DEST.tmp"
fi
