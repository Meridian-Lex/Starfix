#!/usr/bin/env bash
# Starfix — UserPromptSubmit fallback hook
# Injects context if SessionStart missed the marker.
set -euo pipefail
exec "$HOME/.local/bin/starfix" hook userpromptsubmit
