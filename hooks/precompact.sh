#!/usr/bin/env bash
# Starfix — PreCompact hook
# Delegates to starfix binary. Safe to fail silently.
exec "$HOME/.local/bin/starfix" hook precompact
