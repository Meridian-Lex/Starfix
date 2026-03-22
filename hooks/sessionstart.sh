#!/usr/bin/env bash
# Starfix — SessionStart hook
# Outputs additionalContext JSON if post-compaction marker is present.
set -euo pipefail
exec "$HOME/.local/bin/starfix" hook sessionstart
