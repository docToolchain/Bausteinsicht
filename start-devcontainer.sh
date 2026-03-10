#!/usr/bin/env bash
# Start the devcontainer and optionally run a command in it.
#
# Usage:
#   ./start-devcontainer.sh                       # just start the container
#   ./start-devcontainer.sh --rebuild              # rebuild and start
#   ./start-devcontainer.sh claude -p "prompt"     # start + run claude with args

set -euo pipefail

WORKSPACE="$(cd "$(dirname "$0")" && pwd)"

# Check for --rebuild flag
REBUILD=""
if [ "${1:-}" = "--rebuild" ]; then
  REBUILD="--rebuild"
  shift
fi

# Build and start the devcontainer
devcontainer up --workspace-folder "$WORKSPACE" $REBUILD

# If arguments were passed, execute them inside the container.
# Otherwise, open an interactive shell.
if [ $# -gt 0 ]; then
  devcontainer exec --workspace-folder "$WORKSPACE" "$@"
else
  devcontainer exec --workspace-folder "$WORKSPACE" bash
fi
