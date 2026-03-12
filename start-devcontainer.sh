#!/usr/bin/env bash
# Start the devcontainer and optionally run a command in it.
#
# Usage:
#   ./start-devcontainer.sh                       # just start the container
#   ./start-devcontainer.sh --refresh              # rebuild with cache (fast, ~seconds)
#   ./start-devcontainer.sh --rebuild              # rebuild from scratch (slow, ~10min)
#   ./start-devcontainer.sh -- claude -p "prompt"  # start + run command
#   ./start-devcontainer.sh --refresh -- claude    # refresh + run command

set -euo pipefail

WORKSPACE="$(cd "$(dirname "$0")" && pwd)"

# Parse flags (everything before --)
UP_ARGS=()
while [ $# -gt 0 ]; do
  case "$1" in
    --refresh)
      # Rebuild using Docker layer cache — fast for small Dockerfile changes
      UP_ARGS+=(--remove-existing-container)
      shift
      ;;
    --rebuild)
      # Full rebuild without cache — use when cache is stale or broken
      UP_ARGS+=(--remove-existing-container --build-no-cache)
      shift
      ;;
    -h|--help)
      echo "Usage: $(basename "$0") [OPTIONS] [-- COMMAND...]"
      echo ""
      echo "Start the devcontainer and optionally run a command in it."
      echo ""
      echo "Options:"
      echo "  --refresh   Rebuild with Docker layer cache (fast, ~seconds)"
      echo "  --rebuild   Full rebuild without cache (slow, ~10min)"
      echo "  -h, --help  Show this help"
      echo ""
      echo "Examples:"
      echo "  $(basename "$0")                        Start/reuse container, open shell"
      echo "  $(basename "$0") --refresh              Rebuild with cache"
      echo "  $(basename "$0") -- claude -p \"prompt\"  Start + run command"
      exit 0
      ;;
    --)
      shift
      break
      ;;
    *)
      break
      ;;
  esac
done

# Build and start the devcontainer
devcontainer up --workspace-folder "$WORKSPACE" "${UP_ARGS[@]+"${UP_ARGS[@]}"}"

# If arguments remain, execute them inside the container.
# Otherwise, open an interactive shell.
if [ $# -gt 0 ]; then
  devcontainer exec --workspace-folder "$WORKSPACE" "$@"
else
  devcontainer exec --workspace-folder "$WORKSPACE" bash
fi
