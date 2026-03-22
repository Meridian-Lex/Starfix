#!/usr/bin/env bash
# Starfix — PreCompact hook
# Delegates to starfix binary. Runs in strict mode — exits on errors or unset variables.
set -euo pipefail
exec "$HOME/.local/bin/starfix" hook precompact
