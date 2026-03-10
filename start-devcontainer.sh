#!/usr/bin/env bash
# Start the Semantic-Anchors devcontainer and optionally run Claude Code in it.
#
# Usage:
#   ./start-devcontainer.sh                    # just start the container
#   ./start-devcontainer.sh claude -p "prompt" # start + run claude with args

set -euo pipefail

WORKSPACE="$(cd "$(dirname "$0")" && pwd)"

# Build and start the devcontainer
devcontainer up --workspace-folder "$WORKSPACE"

# If arguments were passed, execute them inside the container.
# Otherwise, open an interactive shell.
if [ $# -gt 0 ]; then
  devcontainer exec --workspace-folder "$WORKSPACE" "$@"
else
  devcontainer exec --workspace-folder "$WORKSPACE" bash
fi
