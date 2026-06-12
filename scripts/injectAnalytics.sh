#!/usr/bin/env bash
#
# injectAnalytics.sh TARGET_DIR
#
# Inject the GoatCounter analytics snippet into every *.html below TARGET_DIR,
# right before the first </head>. This runs at deploy time only (see .ci.sh →
# publish_doc), so the snippet never ends up in the jBake templates or in a
# locally generated site.
#
# The endpoint is taken from the GOATCOUNTER_ENDPOINT environment variable
# (a GitHub Actions secret). If it is empty or unset, this script is a no-op,
# so forks and local builds without the secret stay analytics-free.

set -o nounset -o pipefail -o errexit

TARGET_DIR="${1:-}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SNIPPET_TEMPLATE="${SCRIPT_DIR}/goatcounter-snippet.html"

if [ -z "${GOATCOUNTER_ENDPOINT:-}" ]; then
    echo "injectAnalytics: GOATCOUNTER_ENDPOINT not set — skipping analytics injection."
    exit 0
fi

# The endpoint is substituted into a double-quoted HTML attribute. Require a clean
# https:// URL so a misconfigured secret cannot break the markup or inject content.
# Fail the build rather than publish broken or unsafe HTML.
endpoint_re='^https://[^[:space:]"'\''<>]+$'
if [[ ! "${GOATCOUNTER_ENDPOINT}" =~ $endpoint_re ]]; then
    echo "injectAnalytics: GOATCOUNTER_ENDPOINT must be a plain https:// URL without quotes or angle brackets." >&2
    exit 1
fi

if [ -z "${TARGET_DIR}" ] || [ ! -d "${TARGET_DIR}" ]; then
    echo "injectAnalytics: target directory '${TARGET_DIR}' does not exist." >&2
    exit 1
fi

if [ ! -f "${SNIPPET_TEMPLATE}" ]; then
    echo "injectAnalytics: snippet template '${SNIPPET_TEMPLATE}' not found." >&2
    exit 1
fi

# Substitute the placeholder with the real endpoint (bash replace handles the
# slashes in the URL safely, unlike a sed expression).
template="$(cat "${SNIPPET_TEMPLATE}")"
snippet="${template//__GOATCOUNTER_ENDPOINT__/${GOATCOUNTER_ENDPOINT}}"

# Hand the resolved snippet to awk via a temp file so multi-line content and
# any special characters survive untouched.
snippet_file="$(mktemp)"
trap 'rm -f "${snippet_file}"' EXIT
printf '%s\n' "${snippet}" > "${snippet_file}"

injected=0
skipped=0
nohead=0

while IFS= read -r -d '' file; do
    if grep -q 'data-goatcounter' "${file}"; then
        skipped=$((skipped + 1))
        continue
    fi
    # awk exits non-zero when it found no </head> to insert before, so the file
    # is left untouched and counted separately instead of as an injection.
    if awk -v snippetfile="${snippet_file}" '
        BEGIN {
            snippet = ""
            while ((getline line < snippetfile) > 0) {
                snippet = snippet line "\n"
            }
        }
        # Insert immediately before the first </head>, even when it shares a
        # line with other markup (minified / single-line HTML). index()/substr()
        # avoid sub()s special handling of & and \ in the replacement.
        !done {
            p = index($0, "</head>")
            if (p > 0) {
                printf "%s%s%s\n", substr($0, 1, p - 1), snippet, substr($0, p)
                done = 1
                next
            }
        }
        { print }
        END { exit(done ? 0 : 1) }
    ' "${file}" > "${file}.tmp"; then
        mv "${file}.tmp" "${file}"
        injected=$((injected + 1))
    else
        rm -f "${file}.tmp"
        nohead=$((nohead + 1))
        echo "injectAnalytics: no </head> in '${file}', left unchanged." >&2
    fi
done < <(find "${TARGET_DIR}" -type f -name '*.html' -print0)

echo "injectAnalytics: injected into ${injected} file(s), skipped ${skipped} already-instrumented, ${nohead} without </head>."
