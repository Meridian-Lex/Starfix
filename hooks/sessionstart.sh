#!/usr/bin/env bash
# Starfix — SessionStart hook
# Outputs additionalContext JSON if post-compaction marker is present.
exec "$HOME/.local/bin/starfix" hook sessionstart
