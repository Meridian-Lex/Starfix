# Starfix — Design Document

**Date**: 2026-02-28
**Author**: Meridian Lex
**Status**: Approved

---

## Overview

Starfix is a post-compaction context restoration system for Meridian Lex. When the context
window is compacted, Lex loses orientation — active tasks, rank, directives, project state.
Starfix detects compaction, restores critical context, tracks compaction frequency, and
escalates to Fleet Admiral Lunar Laurus via Telegram when sessions are under pressure.

Named after the celestial navigation technique: taking a star fix re-establishes your
exact position after navigating blind.

**Base reference**: https://github.com/Dicklesworthstone/post_compact_reminder
(two-hook workaround pattern; superseded by SessionStart fix in Meridian Lex v2.0.76+)

---

## Goals

- Automatically restore Lex orientation after every compaction
- Track compaction frequency per session
- Notify Fleet Admiral at threshold; escalate with triage recommendation when pressure is high
- Support multiple concurrent Lex/Lex instances without collision
- Be installable by any user from a cloned repo via a simple menu-driven bash script

---

## Non-Goals (v1)

- Stratavore integration (future: compaction events as first-class session records)
- Hot-swap hook configuration without restart
- Dynamic context selection via Qdrant vector search
- Multi-user support

---

## Architecture

### Components

```
Starfix/
├── install.sh                    # Menu-driven bash installer (install / remove)
├── Makefile                      # build, test, lint, vet
├── cmd/starfix/main.go           # CLI entrypoint
├── internal/
│   ├── hook/                     # Hook subcommand handlers
│   │   ├── precompact.go         # PreCompact handler
│   │   ├── sessionstart.go       # SessionStart (compact) handler
│   │   └── userpromptsubmit.go   # UserPromptSubmit fallback handler
│   ├── context/                  # Context payload assembly
│   │   ├── core.go               # MEMORY.md, TASK-QUEUE.md, STATE.md
<!-- IDENTITY-EXCEPTION: functional internal reference — not for public exposure -->
│   │   └── project.go            # CLAUDE.md, README.md, recent git log
│   ├── triage/                   # Situation assessment
│   │   └── triage.go             # continue|park decision + reason
│   ├── telegram/                 # Telegram send/receive
│   │   └── telegram.go           # Wraps ~/.local/bin/telegram-notify; reads inbound log
│   ├── state/                    # Per-session state
│   │   └── state.go              # Read/write ~/.config/starfix/sessions/<session_id>/
│   └── config/                   # Config loader
│       └── config.go             # Reads ~/.config/starfix/starfix.cfg
├── hooks/
│   ├── precompact.sh             # Thin wrapper: starfix hook precompact
│   ├── sessionstart.sh           # Thin wrapper: starfix hook sessionstart
│   └── userpromptsubmit.sh       # Thin wrapper: starfix hook userpromptsubmit
└── config/
    └── starfix.cfg               # Default config (copied to ~/.config/starfix/ at install)
```

### On-Disk State (runtime)

```
~/.config/starfix/
├── starfix.cfg                           # User config (written by install.sh)
└── sessions/
    └── <session_id>/
        ├── state.json                    # Compaction count, escalation state, triage default
        └── compact-pending               # Marker file (present = compaction occurred)
```

All session state is namespaced by `session_id` (provided in every hook stdin payload).
Multiple concurrent Lex/Lex instances operate in separate session directories with no
shared mutable state. The log file is shared but append-only with session_id per line.

### Log

`~/meridian-home/logs/starfix.log`

Format: `2026-02-28T14:32:00Z [session_id_prefix] EVENT message`

---

## Hook Design

### Hook: PreCompact

**Trigger**: Before context compaction
**Binary call**: `starfix hook precompact`

Sequence:
1. Read `session_id` from stdin
2. Load/initialise session state (`~/.config/starfix/sessions/<session_id>/state.json`)
3. Write `compact-pending` marker file
4. Increment `compaction_count`
5. Log event to starfix.log
6. If `compaction_count >= 2`: send Telegram summary (count, timestamp, session)
7. If `compaction_count >= threshold` (config, default 3):
   - Run triage → `{action: continue|park, reason: string}`
   - Send Telegram escalation: summary + recommendation + "will \<action\> in \<timeout\>s if no reply"
   - Write `escalation_pending: true` + `triage_default: continue|park` to state.json

### Hook: SessionStart (compact matcher)

**Trigger**: After compaction (session resume)
**Binary call**: `starfix hook sessionstart`
**Output format**: `{"hookSpecificOutput": {"hookEventName": "SessionStart", "additionalContext": "<payload>"}}`

Sequence:
1. Read `session_id` from stdin
2. Check for `compact-pending` marker
3. If absent: exit 0 (normal session start, not post-compaction)
4. Assemble context payload:
   - Core (always): MEMORY.md + TASK-QUEUE.md + STATE.md
   <!-- IDENTITY-EXCEPTION: functional internal reference — not for public exposure -->
   - Project layer (if `project_context: true` and `$CLAUDE_PROJECT_DIR` detected):
     <!-- IDENTITY-EXCEPTION: functional internal reference — not for public exposure -->
     CLAUDE.md + README.md (first 50 lines) + last 10 git log entries
5. Output JSON with `additionalContext`
6. Delete `compact-pending` marker
7. If `escalation_pending`: start reply watcher goroutine
   - Poll `telegram-inbound.log` for Admiral reply
   - On reply: follow instruction (continue or park), clear escalation state
   - On timeout: execute `triage_default`, notify Admiral of action taken

### Hook: UserPromptSubmit (fallback)

**Trigger**: User submits next message
**Binary call**: `starfix hook userpromptsubmit`

Sequence:
1. Check for `compact-pending` marker (same session_id)
2. If absent: exit 0
3. If present: same inject + delete logic as SessionStart
   (handles edge case where SessionStart did not fire post-compaction)

---

## Triage Logic

Reads from session state and TASK-QUEUE.md:

| Signal | Weight |
|--------|--------|
| compaction_count | High — repeated compaction = context pressure |
| TASK-QUEUE.md has active in-progress task | Medium |
| In-progress task has clear completion criteria | Medium |

Output: `{action: "continue"|"park", reason: string}`

- **continue**: task in progress, clear end in sight, or no active task
- **park**: no clear completion, open-ended exploration, or very high compaction count (>= 5)

The reason string goes directly into the Telegram escalation message.

---

## Telegram Integration

Uses existing `~/.local/bin/telegram-notify` binary. Credentials read from
`~/.config/secrets.yaml` (standard fleet pattern).

**Summary message** (compaction_count >= 2):
```
[Starfix] Compaction #<N> — <session_id_prefix>
Session duration: <elapsed>
Context injected: core + project|core only
```

**Escalation message** (compaction_count >= threshold):
```
[Starfix] Context pressure — <session_id_prefix>
Compaction #<N> this session.
Triage: <reason>
Recommended action: <continue|park>
Will <action> in <timeout>s — reply to override.
```

**Timeout notification** (no reply received):
```
[Starfix] No reply — proceeding to <action>
Session: <session_id_prefix>
```

Inbound reply detection: poll `~/meridian-home/logs/telegram-inbound.log` for new entries
since escalation timestamp. Match on any message from known chat_id.

---

## Configuration

`~/.config/starfix/starfix.cfg` (YAML):

```yaml
# Starfix configuration

# Context injection
<!-- IDENTITY-EXCEPTION: functional internal reference — not for public exposure -->
project_context: true          # Include project layer (CLAUDE.md, README, git log)

# Telegram
telegram_enabled: true
telegram_notify_binary: ~/.local/bin/telegram-notify
telegram_inbound_log: ~/meridian-home/logs/telegram-inbound.log

# Compaction thresholds
summary_threshold: 2           # Send summary on Nth compaction and above
escalation_threshold: 3        # Send escalation (with triage) on Nth compaction and above

# Escalation timeout
timeout_seconds: 300           # Wait this long for Admiral reply before executing triage default

# Logging
log_path: ~/meridian-home/logs/starfix.log

# Lex-specific paths (context assembly)
memory_path: ~/meridian-home/lex-internal/state/MEMORY.md
task_queue_path: ~/meridian-home/lex-internal/state/TASK-QUEUE.md
state_path: ~/meridian-home/lex-internal/state/STATE.md
```

---

## Installer

`install.sh` — menu-driven, no flags required, installs from current directory.

```
Starfix — Post-Compaction Context Restoration

  1) Install for current user
  2) Remove

  > _
```

**Install sequence**:
1. Build Go binary (`go build ./cmd/starfix/`)
2. Copy binary to `~/.local/bin/starfix`
3. Copy hook scripts to `~/.local/bin/` (make executable)
4. Create `~/.config/starfix/` directory
5. Copy `config/starfix.cfg` to `~/.config/starfix/starfix.cfg` (skip if exists)
<!-- IDENTITY-EXCEPTION: functional internal reference — not for public exposure -->
6. Register hooks in `~/.claude/settings.json`:
   - PreCompact
   - SessionStart (compact matcher)
   - UserPromptSubmit
7. Confirm success

**Remove sequence**:
<!-- IDENTITY-EXCEPTION: functional internal reference — not for public exposure -->
1. Remove hooks from `~/.claude/settings.json`
2. Remove binary and hook scripts from `~/.local/bin/`
3. Optionally remove `~/.config/starfix/` (prompt user)

---

## Deployment Model

No dev/live split in config. The installer installs from the current directory — the
developer (Fleet Admiral) maintains a working tree and runs `install.sh` to update the
live installation after changes. The Go binary is rebuilt on each install.

For active development: edit → `go build` → test hooks directly → `install.sh` to
register updated scripts.

---

## Concurrency

All state is namespaced by `session_id`. Multiple concurrent Lex instances each get
their own `~/.config/starfix/sessions/<session_id>/` directory. No mutexes required for
state files (one writer per session). The shared log file uses append-only writes which
are atomic on Linux for lines under PIPE_BUF (4096 bytes).

---

## Future / Out of Scope for v1

- **Stratavore integration**: compaction events as session records in Stratavore's
  PostgreSQL store; triage driven by Qdrant knowledge retrieval
- **Dynamic context selection**: semantic search over knowledge base to inject only
  the most relevant context for the current task
- **Session cleanup**: cron or daemon to purge stale session directories
- **Metrics**: compaction frequency trends across sessions

---

## Reference

- Base implementation: `projects/post_compact_reminder/`
<!-- IDENTITY-EXCEPTION: functional internal reference — not for public exposure -->
- SessionStart fix: https://github.com/anthropics/claude-code/issues/13650 (fixed v2.0.76+)
- Hook documentation: Meridian Lex hooks skill
- Fleet Telegram stack: `lex-internal/knowledge/telegram-stack.md`
- Existing hooks: `lex-internal/enforcement/hooks/`
