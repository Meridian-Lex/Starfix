# Starfix

Post-compaction context restoration for Meridian Lex.

After a context compaction event, Lex loses orientation — active tasks, rank,
directives, project state. Starfix detects compaction and injects critical context
back automatically. It also tracks compaction frequency and escalates to Fleet Admiral
via Telegram when sessions are under sustained pressure.

## Requirements

- Go 1.24+
- jq
- ~/.local/bin/telegram-notify (fleet Telegram binary)

## Install

Clone the repository then run the installer:

    ./install.sh

Select option 1. Restart your Lex session to activate hooks.

## What It Does

| Hook | Trigger | Action |
|------|---------|--------|
| PreCompact | Before compaction | Write marker, track count, send Telegram at threshold |
| SessionStart | After compaction | Inject MEMORY.md + TASK-QUEUE.md + STATE.md + project context |
| UserPromptSubmit | User message | Fallback injection if SessionStart missed; handle reply/timeout flags |

## Telegram Escalation

On the Nth compaction (configurable, default 3), Starfix:

1. Runs triage — assesses session pressure from compaction count and task queue
2. Sends Telegram message with recommendation and timeout default
3. Spawns background reply watcher
4. On reply: follows instruction; on timeout: executes triage default

## Configuration

Edit `~/.config/starfix/starfix.cfg` after install. Full options documented inline.

## Concurrent Sessions

All state is namespaced by session ID. Multiple concurrent Lex instances operate
independently with no shared mutable state.

## Design

See `docs/plans/2026-02-28-starfix-design.md`.
