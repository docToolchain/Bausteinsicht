#!/usr/bin/env bash
# scripts/fetch-xmi-testdata.sh — fetch the large XMI integration-test
# fixture (internal/importer/xmi/testdata/BigData.xmi, a real ~118MB
# Enterprise Architect/AUTOSAR export) from the separate, lightweight
# docToolchain/bausteinsicht-testdata repo, instead of committing or
# Git-LFS-tracking it in this repository (#553).
#
# Usage:
#   scripts/fetch-xmi-testdata.sh
#   make fetch-testdata
#
# Safe to run repeatedly: skips the download if a real (>1MB) file is
# already present. Safe to fail: on any network/fetch error it prints a
# warning and exits 0 rather than failing the caller — internal/importer/xmi's
# TestImport_BigData already skips itself when the fixture is absent or too
# small, so offline/local dev is never blocked by this script. CI jobs that
# specifically need the fixture (see the xmi-bigdata-integration job in
# go.yml) check the test actually ran, so a silent fetch failure there is
# still caught — it just fails that one dedicated job, not the whole build.
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEST="$ROOT/internal/importer/xmi/testdata/BigData.xmi"
URL="https://raw.githubusercontent.com/docToolchain/bausteinsicht-testdata/main/BigData.xmi"
MIN_SIZE=1048576 # 1 MiB — matches TestImport_BigData's own presence check

file_size() {
    stat -c%s "$1" 2>/dev/null || stat -f%z "$1" 2>/dev/null || echo 0
}

if [ -f "$DEST" ] && [ "$(file_size "$DEST")" -gt "$MIN_SIZE" ]; then
    echo "BigData.xmi already present ($(file_size "$DEST") bytes), skipping fetch."
    exit 0
fi

echo "Fetching BigData.xmi from docToolchain/bausteinsicht-testdata..."
if curl -fsSL --retry 2 -o "$DEST.tmp" "$URL" && mv "$DEST.tmp" "$DEST"; then
    echo "Fetched BigData.xmi ($(file_size "$DEST") bytes)."
else
    echo "Warning: could not fetch/place BigData.xmi from $URL (network issue, fixture repo unavailable, or move failed)." >&2
    echo "TestImport_BigData will skip itself; this is not treated as a hard failure here." >&2
    rm -f "$DEST.tmp"
fi
