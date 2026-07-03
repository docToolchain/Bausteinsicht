#!/usr/bin/env bash
# scripts/e2e-test-report.sh — line-accurate PASS/FAIL/SKIP report for
# E2E-Test-Plan.adoc, generated from a real `go test ./e2e/...` run.
#
# Usage:
#   scripts/e2e-test-report.sh              # prints AsciiDoc to stdout
#   scripts/e2e-test-report.sh > report.adoc
#
# This is the automated counterpart to hand-writing
# src/docs/e2e-test-report-YYYY-MM-DD.adoc after a manual test round (see
# "How to Execute" in E2E-Test-Plan.adoc) — see #519's "per CI statt
# manuell erzeugen" follow-up. It does not replace the narrative sections
# (Environment, Failures, Skips write-ups) a manual report includes; it
# generates the mechanical per-line PASS/FAIL/SKIP table those reports are
# built around.
#
# Always runs the full e2e suite (not a filtered subset) — see the
# "Limitation: run the full suite" note in
# scripts/e2e-test-report-gen/main.go for why a partial run produces a
# misleading report.

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

go test ./e2e/... -json | go run ./scripts/e2e-test-report-gen "$@"
